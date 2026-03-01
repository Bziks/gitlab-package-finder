package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"

	"github.com/bziks/gitlab-package-finder/internal/domain/entity"
	"github.com/bziks/gitlab-package-finder/internal/ports"
	"github.com/bziks/gitlab-package-finder/internal/services/search"
)

type SearchProcessingJob struct {
	packageSearchStorage ports.PackageSearchStorage
	searchService        *search.Service
}

func NewSearchProcessingJob(
	packageSearchStorage ports.PackageSearchStorage,
	searchService *search.Service,
) *SearchProcessingJob {
	return &SearchProcessingJob{
		packageSearchStorage: packageSearchStorage,
		searchService:        searchService,
	}
}

func (j *SearchProcessingJob) Execute(ctx context.Context) error {
	tracer := otel.Tracer("search-processing")
	ctx, span := tracer.Start(ctx, "SearchProcessingJob.Execute")
	defer span.End()

	slog.DebugContext(ctx, "execute search processing")

	defer func() {
		slog.DebugContext(ctx, "finished processing")
	}()

	searchID, searchPackage, err := j.packageSearchStorage.GetSearchFromQueue(ctx)
	if err != nil {
		return fmt.Errorf("get search from queue: %w", err)
	}

	if searchPackage == nil {
		slog.InfoContext(ctx, "no search package found")
		select {
		case <-ctx.Done():
		case <-time.After(10 * time.Second):
		}
		return nil
	}

	slog.InfoContext(ctx, "search package", "searchPackage", searchPackage)

	project, err := j.packageSearchStorage.GetProjectFromQueue(ctx, searchID)
	if err != nil {
		return fmt.Errorf("get project from queue: %w", err)
	}

	if project == nil {
		slog.InfoContext(ctx, "no project found")

		if err = j.packageSearchStorage.CompleteSearch(ctx, searchID); err != nil {
			return fmt.Errorf("complete search: %w", err)
		}

		return nil
	}

	status, err := j.packageSearchStorage.GetSearchStatus(ctx, searchID)
	if err != nil {
		return fmt.Errorf("get search status: %w", err)
	}
	if status == entity.SearchStatusPending {
		if err = j.packageSearchStorage.UpdateSearchStatus(ctx, searchID, entity.SearchStatusProcessing); err != nil {
			slog.ErrorContext(ctx, "update search status", "error", err)
		}
	}

	slog.InfoContext(ctx, "project", "project", project)

	// Delegate to SearchService for actual business logic
	j.searchService.ProcessProject(ctx, searchID, searchPackage, project)

	return nil
}
