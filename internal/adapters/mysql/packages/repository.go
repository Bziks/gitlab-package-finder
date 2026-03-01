package packages

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

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

func (r *Repository) FindDependencies(ctx context.Context, page int, packageType string, packageName string) (entity.ProjectDependencyWithPagination, error) {
	const op = "find dependencies by package criteria from mysql"

	var res []ProjectDependencyWithDetails
	var err error
	var limit int = 10
	var offset int = (page - 1) * limit
	var result = entity.ProjectDependencyWithPagination{}

	baseSelect := `SELECT
		pd.id,
		pd.source_file,
		pd.version_constraint,
		pd.resolved_version,
		pd.last_sync,
		p.name as package_name,
		p.type_id as package_type_id,
		pt.name as package_type_name,
		pt.label as package_type_label,
		p.last_sync as package_last_sync,
		pr.id as project_id,
		pr.name as project_name,
		pr.url as project_url,
		pb.name as project_branch_name`

	baseFrom := `
	FROM project_dependencies pd
	INNER JOIN packages p ON pd.package_id = p.id
	INNER JOIN package_types pt ON p.type_id = pt.id
	INNER JOIN project_branches pb ON pd.project_branch_id = pb.id
	INNER JOIN projects pr ON pb.project_id = pr.id`

	err = r.client.QueryRowContext(
		ctx,
		`SELECT COUNT(*)`+baseFrom+`
		WHERE p.name = ? AND pt.name = ?`,
		packageName, packageType,
	).Scan(&result.Total)
	if err != nil {
		return entity.ProjectDependencyWithPagination{}, fmt.Errorf("%s: %w", op, err)
	}

	err = r.client.SelectContext(
		ctx,
		&res,
		baseSelect+baseFrom+`
		WHERE p.name = ? AND pt.name = ?
		ORDER BY pd.id ASC
		LIMIT ? OFFSET ?`,
		packageName, packageType, limit, offset,
	)

	if err != nil {
		return entity.ProjectDependencyWithPagination{}, fmt.Errorf("%s: %w", op, err)
	}

	result.Dependencies = make([]entity.ProjectDependency, len(res))
	for i := range res {
		result.Dependencies[i] = dependencyWithDetailsToDomain(res[i])
	}

	return result, nil
}

func (r *Repository) FindDependenciesCursor(ctx context.Context, packageType string, packageName string, afterID int, limit int) ([]entity.ProjectDependency, error) {
	const op = "find dependencies by cursor from mysql"

	baseSelect := `SELECT
		pd.id,
		pd.source_file,
		pd.version_constraint,
		pd.resolved_version,
		pd.last_sync,
		p.name as package_name,
		p.type_id as package_type_id,
		pt.name as package_type_name,
		pt.label as package_type_label,
		p.last_sync as package_last_sync,
		pr.id as project_id,
		pr.name as project_name,
		pr.url as project_url,
		pb.name as project_branch_name`

	baseFrom := `
	FROM project_dependencies pd
	INNER JOIN packages p ON pd.package_id = p.id
	INNER JOIN package_types pt ON p.type_id = pt.id
	INNER JOIN project_branches pb ON pd.project_branch_id = pb.id
	INNER JOIN projects pr ON pb.project_id = pr.id`

	var res []ProjectDependencyWithDetails
	err := r.client.SelectContext(
		ctx,
		&res,
		baseSelect+baseFrom+`
		WHERE p.name = ? AND pt.name = ? AND pd.id > ?
		ORDER BY pd.id ASC
		LIMIT ?`,
		packageName, packageType, afterID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	deps := make([]entity.ProjectDependency, len(res))
	for i := range res {
		deps[i] = dependencyWithDetailsToDomain(res[i])
	}

	return deps, nil
}

func (r *Repository) GetByNameAndType(ctx context.Context, name string, typeID int) (*entity.Package, error) {
	const op = "get package by name and type from mysql"

	res := Package{}

	err := r.client.GetContext(ctx, &res, "SELECT * FROM packages WHERE name = ? AND type_id = ?", name, typeID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, entity.ErrPackageNotFound
		}

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	slog.InfoContext(ctx, "package", "package", res)

	return &entity.Package{
		ID:       res.ID,
		Name:     res.Name,
		TypeID:   res.TypeID,
		LastSync: res.LastSync,
	}, nil
}

func (r *Repository) Create(ctx context.Context, name string, typeID int) (*entity.Package, error) {
	const op = "create package from mysql"

	res, err := r.client.ExecContext(ctx, "INSERT INTO packages (name, type_id, last_sync) VALUES (?, ?, ?)", name, typeID, time.Now())
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &entity.Package{
		ID:       int(id),
		Name:     name,
		TypeID:   typeID,
		LastSync: time.Now(),
	}, nil
}

func (r *Repository) AttachPackageToProjectDependency(ctx context.Context, projectBranchID int, packageID int, sourceFile string, versionConstraint string, resolvedVersion string) error {
	const op = "attach package to project dependency from mysql"

	now := time.Now()

	_, err := r.client.ExecContext(
		ctx,
		`INSERT INTO project_dependencies
			(project_branch_id, package_id, source_file, version_constraint, resolved_version, last_sync)
		VALUES (?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			last_sync=VALUES(last_sync),
			version_constraint=VALUES(version_constraint),
			resolved_version=VALUES(resolved_version)`,
		projectBranchID, packageID, sourceFile, versionConstraint, resolvedVersion, now,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *Repository) GetCount(ctx context.Context) (int, error) {
	const op = "get packages count from mysql"

	var count int
	err := r.client.QueryRowContext(ctx, "SELECT COUNT(*) FROM packages").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return count, nil
}
