package ports

import (
	"context"

	"github.com/bziks/gitlab-package-finder/internal/domain/entity"
)

type PackageTypeRepository interface {
	GetAll(ctx context.Context) ([]entity.PackageType, error)
	GetByName(ctx context.Context, name string) (*entity.PackageType, error)
}
