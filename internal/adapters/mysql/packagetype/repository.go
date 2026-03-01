package packagetype

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/bziks/gitlab-package-finder/internal/domain/entity"
)

type Repository struct {
	client *sqlx.DB
}

func NewRepository(client *sqlx.DB) *Repository {
	return &Repository{
		client: client,
	}
}

func (r *Repository) GetAll(ctx context.Context) ([]entity.PackageType, error) {
	const op = "get all package types from mysql"

	res := []PackageType{}

	err := r.client.SelectContext(ctx, &res, "SELECT * FROM package_types")
	if err != nil {
		return make([]entity.PackageType, 0), fmt.Errorf("%s: %w", op, err)
	}

	var entities = make([]entity.PackageType, len(res))
	for i := range res {
		entities[i] = toDomain(res[i])
	}

	return entities, nil
}

func (r *Repository) GetByName(ctx context.Context, name string) (*entity.PackageType, error) {
	const op = "get package type by name from mysql"

	res := PackageType{}

	err := r.client.GetContext(ctx, &res, "SELECT * FROM package_types WHERE name = ?", name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%s: package type '%s' not found", op, name)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &entity.PackageType{
		ID:    res.ID,
		Name:  res.Name,
		Label: res.Label,
	}, nil
}
