package ports

import (
	"context"

	"github.com/bziks/gitlab-package-finder/internal/domain/entity"
)

type ProjectRepository interface {
	GetWithPagination(ctx context.Context, page, limit int) (entity.ProjectsWithPagination, error)
	GetWithPackageTypesAndPagination(ctx context.Context, page, limit int) (entity.ProjectsWithPackageTypesAndPagination, error)
	GetDetailedWithPagination(ctx context.Context, page, limit int) (entity.DetailedProjectsWithPagination, error)
	GetDetailedByPackageTypeWithPagination(ctx context.Context, packageTypeName string, page, limit int) (entity.DetailedProjectsWithPagination, error)
	SearchByName(ctx context.Context, query string, page, limit int) (entity.ProjectsWithPagination, error)
	SearchByNameWithPackageTypes(ctx context.Context, query string, page, limit int) (entity.ProjectsWithPackageTypesAndPagination, error)
	UpSert(ctx context.Context, project entity.Project) error
	GetCount(ctx context.Context) (int, error)
	Delete(ctx context.Context, id int) error
	SyncPackageTypes(ctx context.Context, projectID int, packageTypeIDs []int) error
}
