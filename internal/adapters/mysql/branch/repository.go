package branch

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

func (r *Repository) Find(ctx context.Context, projectID int, name string) (*entity.Branch, error) {
	const op = "find branch in mysql"

	var model Branch
	err := r.client.GetContext(ctx, &model, "SELECT * FROM project_branches WHERE project_id=? AND name=?", projectID, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	b := toDomain(model)

	return &b, nil
}

func (r *Repository) GetByProjectID(ctx context.Context, projectID int) ([]entity.Branch, error) {
	const op = "get branches by project id from mysql"

	var models []Branch
	err := r.client.SelectContext(ctx, &models, "SELECT * FROM project_branches WHERE project_id=?", projectID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	branches := make([]entity.Branch, len(models))
	for i := range models {
		branches[i] = toDomain(models[i])
	}

	return branches, nil
}

func (r *Repository) GetByID(ctx context.Context, id int) (*entity.Branch, error) {
	const op = "get branch by id from mysql"

	var model Branch
	err := r.client.GetContext(ctx, &model, "SELECT * FROM project_branches WHERE id=?", id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	b := toDomain(model)

	return &b, nil
}

func (r *Repository) Create(ctx context.Context, projectID int, name string) (*entity.Branch, error) {
	const op = "create branch in mysql"

	result, err := r.client.ExecContext(ctx, "INSERT INTO project_branches (project_id, name) VALUES (?, ?)", projectID, name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("%s: get last insert id: %w", op, err)
	}

	return &entity.Branch{
		ID:   int(id),
		Name: name,
	}, nil
}

func (r *Repository) Delete(ctx context.Context, id int) error {
	const op = "delete branch from mysql"

	_, err := r.client.ExecContext(ctx, "DELETE FROM project_branches WHERE id=?", id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
