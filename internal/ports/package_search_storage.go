package ports

import (
	"context"

	"github.com/bziks/gitlab-package-finder/internal/domain/entity"
)

type PackageSearchStorage interface {
	CheckIfSearchIsRunning(ctx context.Context, searchID string) (bool, error)
	AcquireSearch(ctx context.Context, searchID string) (bool, error)
	AddSearchToQueue(ctx context.Context, searchID string, searchPackage entity.SearchPackage) error
	GetSearchFromQueue(ctx context.Context) (string, *entity.SearchPackage, error)
	AddProjectsToQueue(ctx context.Context, searchID string, project entity.DetailedProject) error
	GetProjectFromQueue(ctx context.Context, searchID string) (*entity.DetailedProject, error)
	GetProjectsQueueLength(ctx context.Context, searchID string) (int64, error)
	UpdateSearchStatus(ctx context.Context, searchID string, status string) error
	GetSearchStatus(ctx context.Context, searchID string) (string, error)
	GetSearchDetails(ctx context.Context, searchID string) (*entity.SearchPackage, error)
	CompleteSearch(ctx context.Context, searchID string) error
	AddFailedRepository(ctx context.Context, searchID string, project *entity.DetailedProject, branch entity.Branch, errorMsg string) error
	GetFailedRepositories(ctx context.Context, searchID string) ([]entity.FailedRepository, error)
}
