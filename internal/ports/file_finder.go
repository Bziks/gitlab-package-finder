package ports

import (
	"context"

	"github.com/bziks/gitlab-package-finder/internal/domain/entity"
)

type FileFinder interface {
	FindDependencyFiles(ctx context.Context, packageName string, projectID int, branchName string, patterns entity.FilePatterns) ([]entity.DependencyFile, error)
}
