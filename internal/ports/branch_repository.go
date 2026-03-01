package ports

import (
	"context"

	"github.com/bziks/gitlab-package-finder/internal/domain/entity"
)

type BranchRepository interface {
	Find(ctx context.Context, projectID int, name string) (*entity.Branch, error)
	GetByProjectID(ctx context.Context, projectID int) ([]entity.Branch, error)
	GetByID(ctx context.Context, id int) (*entity.Branch, error)
	Create(ctx context.Context, projectID int, name string) (*entity.Branch, error)
	Delete(ctx context.Context, id int) error
}
