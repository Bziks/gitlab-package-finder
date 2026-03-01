package worker

import (
	"context"
	"testing"
	"time"
)

func TestWorkerGracefulShutdown(t *testing.T) {
	job := &testJob{}

	w := NewWorker(Config{
		Job:           job,
		ShutdownDelay: 5 * time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())

	go w.Run(ctx)

	// Let the worker process at least one job
	time.Sleep(150 * time.Millisecond)

	start := time.Now()
	cancel()

	// Wait for the worker to finish — it should complete the in-flight job
	// and shut down within the job duration + some margin
	time.Sleep(200 * time.Millisecond)

	elapsed := time.Since(start)
	if elapsed > 500*time.Millisecond {
		t.Fatalf("shutdown took too long: %v", elapsed)
	}
}

type testJob struct{}

func (j *testJob) Execute(ctx context.Context) error {
	time.Sleep(100 * time.Millisecond)
	return nil
}
