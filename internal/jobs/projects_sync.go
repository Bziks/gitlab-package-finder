package jobs

import (
	"context"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"

	"github.com/bziks/gitlab-package-finder/internal/command/projectssync"
)

type ProjectsSyncJob struct {
	syncCommand *projectssync.Command
	interval    time.Duration
}

func NewProjectsSyncJob(syncCommand *projectssync.Command, interval time.Duration) *ProjectsSyncJob {
	return &ProjectsSyncJob{
		syncCommand: syncCommand,
		interval:    interval,
	}
}

func (j *ProjectsSyncJob) Interval() time.Duration {
	return j.interval
}

func (j *ProjectsSyncJob) Execute(ctx context.Context) error {
	tracer := otel.Tracer("projects-sync")
	ctx, span := tracer.Start(ctx, "ProjectsSyncJob.Execute")
	defer span.End()

	slog.InfoContext(ctx, "execute projects sync")

	if err := j.syncCommand.Execute(ctx); err != nil {
		return err
	}

	slog.InfoContext(ctx, "projects sync completed")

	return nil
}
