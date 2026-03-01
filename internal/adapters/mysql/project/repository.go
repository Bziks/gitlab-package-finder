package project

import (
	"context"
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

func (r *Repository) GetWithPagination(ctx context.Context, page, limit int) (entity.ProjectsWithPagination, error) {
	const op = "get projects from mysql"

	var (
		offset int = (page - 1) * limit
		res        = []Project{}
		result     = entity.ProjectsWithPagination{}
	)

	err := r.client.QueryRowContext(ctx, "SELECT COUNT(*) FROM projects").Scan(&result.Total)
	if err != nil {
		return entity.ProjectsWithPagination{}, fmt.Errorf("%s: %w", op, err)
	}

	err = r.client.SelectContext(
		ctx,
		&res,
		`SELECT * FROM projects LIMIT ? OFFSET ?`,
		limit,
		offset,
	)
	if err != nil {
		return entity.ProjectsWithPagination{}, fmt.Errorf("%s: %w", op, err)
	}

	result.Projects = make([]entity.Project, len(res))
	for i := range res {
		result.Projects[i] = toDomain(res[i])
	}

	return result, nil
}

func (r *Repository) GetWithPackageTypesAndPagination(ctx context.Context, page, limit int) (entity.ProjectsWithPackageTypesAndPagination, error) {
	const op = "get projects with package types from mysql"

	var (
		offset int = (page - 1) * limit
		res        = []ProjectWithPackageTypes{}
		result     = entity.ProjectsWithPackageTypesAndPagination{}
	)

	err := r.client.QueryRowContext(ctx, "SELECT COUNT(*) FROM projects").Scan(&result.Total)
	if err != nil {
		return entity.ProjectsWithPackageTypesAndPagination{}, fmt.Errorf("%s: %w", op, err)
	}

	err = r.client.SelectContext(
		ctx,
		&res,
		`SELECT p.id, p.name, p.url,
			GROUP_CONCAT(DISTINCT pt.name ORDER BY pt.name) as package_types
		FROM projects p
		LEFT JOIN project_package_types ppt ON p.id = ppt.project_id
		LEFT JOIN package_types pt ON ppt.package_type_id = pt.id
		GROUP BY p.id, p.name, p.url
		LIMIT ? OFFSET ?`,
		limit,
		offset,
	)
	if err != nil {
		return entity.ProjectsWithPackageTypesAndPagination{}, fmt.Errorf("%s: %w", op, err)
	}

	result.Projects = make([]entity.ProjectWithPackageTypes, len(res))
	for i := range res {
		result.Projects[i] = toProjectWithPackageTypesDomain(res[i])
	}

	return result, nil
}

func (r *Repository) SearchByNameWithPackageTypes(ctx context.Context, query string, page, limit int) (entity.ProjectsWithPackageTypesAndPagination, error) {
	const op = "search projects by name with package types from mysql"

	var (
		offset int = (page - 1) * limit
		res        = []ProjectWithPackageTypes{}
		result     = entity.ProjectsWithPackageTypesAndPagination{}
	)

	likeQuery := "%" + query + "%"

	err := r.client.QueryRowContext(ctx, "SELECT COUNT(*) FROM projects WHERE name LIKE ?", likeQuery).Scan(&result.Total)
	if err != nil {
		return entity.ProjectsWithPackageTypesAndPagination{}, fmt.Errorf("%s: %w", op, err)
	}

	err = r.client.SelectContext(
		ctx,
		&res,
		`SELECT p.id, p.name, p.url,
			GROUP_CONCAT(DISTINCT pt.name ORDER BY pt.name) as package_types
		FROM projects p
		LEFT JOIN project_package_types ppt ON p.id = ppt.project_id
		LEFT JOIN package_types pt ON ppt.package_type_id = pt.id
		WHERE p.name LIKE ?
		GROUP BY p.id, p.name, p.url
		LIMIT ? OFFSET ?`,
		likeQuery,
		limit,
		offset,
	)
	if err != nil {
		return entity.ProjectsWithPackageTypesAndPagination{}, fmt.Errorf("%s: %w", op, err)
	}

	result.Projects = make([]entity.ProjectWithPackageTypes, len(res))
	for i := range res {
		result.Projects[i] = toProjectWithPackageTypesDomain(res[i])
	}

	return result, nil
}

func (r *Repository) GetCount(ctx context.Context) (int, error) {
	const op = "get projects count from mysql"

	var count int
	err := r.client.QueryRowContext(ctx, "SELECT COUNT(*) FROM projects").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return count, nil
}

func (r *Repository) UpSert(ctx context.Context, project entity.Project) error {
	const op = "insert or update project to mysql"

	_, err := r.client.NamedExecContext(
		ctx,
		`INSERT INTO projects (id, name, url) VALUES (:id, :name, :url)
		ON DUPLICATE KEY UPDATE name=:name, url=:url`,
		fromDomain(project),
	)

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *Repository) SearchByName(ctx context.Context, query string, page, limit int) (entity.ProjectsWithPagination, error) {
	const op = "search projects by name from mysql"

	var (
		offset int = (page - 1) * limit
		res        = []Project{}
		result     = entity.ProjectsWithPagination{}
	)

	likeQuery := "%" + query + "%"

	err := r.client.QueryRowContext(ctx, "SELECT COUNT(*) FROM projects WHERE name LIKE ?", likeQuery).Scan(&result.Total)
	if err != nil {
		return entity.ProjectsWithPagination{}, fmt.Errorf("%s: %w", op, err)
	}

	err = r.client.SelectContext(
		ctx,
		&res,
		"SELECT * FROM projects WHERE name LIKE ? LIMIT ? OFFSET ?",
		likeQuery,
		limit,
		offset,
	)
	if err != nil {
		return entity.ProjectsWithPagination{}, fmt.Errorf("%s: %w", op, err)
	}

	result.Projects = make([]entity.Project, len(res))
	for i := range res {
		result.Projects[i] = toDomain(res[i])
	}

	return result, nil
}

func (r *Repository) Delete(ctx context.Context, id int) error {
	const op = "delete project from mysql"

	_, err := r.client.ExecContext(ctx, "DELETE FROM projects WHERE id=?", id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *Repository) GetDetailedWithPagination(ctx context.Context, page, limit int) (entity.DetailedProjectsWithPagination, error) {
	const op = "get projects from mysql"

	var (
		offset int = (page - 1) * limit
		res        = []DetailedProject{}
		result     = entity.DetailedProjectsWithPagination{}
	)

	err := r.client.QueryRowContext(ctx, "SELECT COUNT(*) FROM projects").Scan(&result.Total)
	if err != nil {
		return entity.DetailedProjectsWithPagination{}, fmt.Errorf("%s: %w", op, err)
	}

	err = r.client.SelectContext(
		ctx,
		&res,
		`SELECT
			p.id,
			p.name,
			p.url,
			GROUP_CONCAT(b.id SEPARATOR '||') as branch_ids,
			GROUP_CONCAT(b.name SEPARATOR '||') as branches
		FROM projects p
		LEFT JOIN project_branches b ON p.id = b.project_id
		GROUP BY p.id, p.name, p.url LIMIT ? OFFSET ?`,
		limit,
		offset,
	)
	if err != nil {
		return entity.DetailedProjectsWithPagination{}, fmt.Errorf("%s: %w", op, err)
	}

	result.Projects = make([]entity.DetailedProject, len(res))
	for i := range res {
		result.Projects[i] = toDetailedDomain(res[i])
	}

	return result, nil
}

func (r *Repository) GetDetailedByPackageTypeWithPagination(ctx context.Context, packageTypeName string, page, limit int) (entity.DetailedProjectsWithPagination, error) {
	const op = "get projects by package type from mysql"

	var (
		offset = (page - 1) * limit
		res    = []DetailedProject{}
		result = entity.DetailedProjectsWithPagination{}
	)

	err := r.client.QueryRowContext(
		ctx,
		`SELECT COUNT(DISTINCT p.id) FROM projects p
		LEFT JOIN project_package_types ppt ON p.id = ppt.project_id
		LEFT JOIN package_types pt ON ppt.package_type_id = pt.id
		WHERE pt.name = ? OR ppt.project_id IS NULL`,
		packageTypeName,
	).Scan(&result.Total)
	if err != nil {
		return entity.DetailedProjectsWithPagination{}, fmt.Errorf("%s: %w", op, err)
	}

	err = r.client.SelectContext(
		ctx,
		&res,
		`SELECT p.id, p.name, p.url,
			GROUP_CONCAT(DISTINCT b.id SEPARATOR '||') as branch_ids,
			GROUP_CONCAT(DISTINCT b.name SEPARATOR '||') as branches
		FROM projects p
		LEFT JOIN project_branches b ON p.id = b.project_id
		LEFT JOIN project_package_types ppt ON p.id = ppt.project_id
		LEFT JOIN package_types pt ON ppt.package_type_id = pt.id
		WHERE pt.name = ? OR ppt.project_id IS NULL
		GROUP BY p.id, p.name, p.url
		LIMIT ? OFFSET ?`,
		packageTypeName,
		limit,
		offset,
	)
	if err != nil {
		return entity.DetailedProjectsWithPagination{}, fmt.Errorf("%s: %w", op, err)
	}

	result.Projects = make([]entity.DetailedProject, len(res))
	for i := range res {
		result.Projects[i] = toDetailedDomain(res[i])
	}

	return result, nil
}

func (r *Repository) SyncPackageTypes(ctx context.Context, projectID int, packageTypeIDs []int) error {
	const op = "sync project package types in mysql"

	tx, err := r.client.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: begin tx: %w", op, err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, "DELETE FROM project_package_types WHERE project_id = ?", projectID)
	if err != nil {
		return fmt.Errorf("%s: delete: %w", op, err)
	}

	if len(packageTypeIDs) == 0 {
		return tx.Commit()
	}

	query := "INSERT INTO project_package_types (project_id, package_type_id) VALUES "
	args := make([]interface{}, 0, len(packageTypeIDs)*2)
	for i, typeID := range packageTypeIDs {
		if i > 0 {
			query += ", "
		}
		query += "(?, ?)"
		args = append(args, projectID, typeID)
	}

	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: insert: %w", op, err)
	}

	return tx.Commit()
}
