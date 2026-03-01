package worker

import (
	"context"
	"errors"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"
)

// Job represents a unit of work that can be executed by a worker.
type Job interface {
	Execute(ctx context.Context) error
}

// IntervalJob is an optional interface that jobs can implement to specify
// a delay between executions. The worker will wait this duration (context-aware)
// after each execution before starting the next one.
type IntervalJob interface {
	Interval() time.Duration
}

// Worker manages the execution of jobs with graceful shutdown capabilities.
type Worker struct {
	job           Job
	shutdownDelay time.Duration
	logger        *slog.Logger

	mu        sync.RWMutex
	wg        sync.WaitGroup
	isRunning bool
}

type Config struct {
	Job           Job
	ShutdownDelay time.Duration
	Logger        *slog.Logger
}

func NewWorker(config Config) *Worker {
	if config.ShutdownDelay == 0 {
		config.ShutdownDelay = 30 * time.Second
	}

	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &Worker{
		job:           config.Job,
		shutdownDelay: config.ShutdownDelay,
		logger:        logger,
	}
}

// Run starts the worker and blocks until the context is cancelled.
// After cancellation, it waits for the current job to finish (up to shutdownDelay).
func (w *Worker) Run(ctx context.Context) error {
	w.mu.Lock()
	if w.isRunning {
		w.mu.Unlock()
		return errors.New("worker is already running")
	}
	w.isRunning = true
	w.mu.Unlock()

	w.logger.InfoContext(ctx, "worker starting")

	defer func() {
		w.mu.Lock()
		w.isRunning = false
		w.mu.Unlock()
		w.logger.InfoContext(ctx, "worker stopped")
	}()

	w.wg.Add(1)
	go w.processJobs(ctx)

	<-ctx.Done()
	w.logger.InfoContext(ctx, "worker received shutdown signal")
	w.awaitShutdown(ctx)

	return ctx.Err()
}

func (w *Worker) processJobs(ctx context.Context) {
	defer w.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			w.executeJobSafely(ctx)
		}
	}
}

func (w *Worker) executeJobSafely(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			w.logger.ErrorContext(ctx, "job panicked",
				"panic", r,
				"stack", string(debug.Stack()),
			)
		}
	}()

	if err := w.job.Execute(ctx); err != nil {
		w.logger.ErrorContext(ctx, "job returned error", "error", err)
	}

	// If the job specifies an interval, wait before next execution (context-aware)
	if ij, ok := w.job.(IntervalJob); ok {
		interval := ij.Interval()
		if interval > 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(interval):
			}
		}
	}
}

func (w *Worker) awaitShutdown(ctx context.Context) {
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		w.logger.InfoContext(ctx, "graceful shutdown completed")
	case <-time.After(w.shutdownDelay):
		w.logger.WarnContext(ctx, "graceful shutdown timeout exceeded", "timeout", w.shutdownDelay)
	}
}
