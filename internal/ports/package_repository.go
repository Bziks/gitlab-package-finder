package ports

import (
	"context"

	"github.com/bziks/gitlab-package-finder/internal/domain/entity"
)

type PackageRepository interface {
	FindDependencies(ctx context.Context, page int, packageType string, packageName string) (entity.ProjectDependencyWithPagination, error)
	FindDependenciesCursor(ctx context.Context, packageType string, packageName string, afterID int, limit int) ([]entity.ProjectDependency, error)
	GetByNameAndType(ctx context.Context, name string, typeID int) (*entity.Package, error)
	Create(ctx context.Context, name string, typeID int) (*entity.Package, error)
	AttachPackageToProjectDependency(ctx context.Context, projectBranchID int, packageID int, sourceFile string, versionConstraint string, resolvedVersion string) error
	GetCount(ctx context.Context) (int, error)
}
