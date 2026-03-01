package packages

import (
	"database/sql"
	"time"

	"github.com/bziks/gitlab-package-finder/internal/domain/entity"
)

type Package struct {
	ID       int       `db:"id"`
	Name     string    `db:"name"`
	TypeID   int       `db:"type_id"`
	LastSync time.Time `db:"last_sync"`
}

type ProjectDependencyWithDetails struct {
	ID                int            `db:"id"`
	ProjectBranchID   int            `db:"project_branch_id"`
	PackageID         int            `db:"package_id"`
	SourceFile        string         `db:"source_file"`
	VersionConstraint string         `db:"version_constraint"`
	ResolvedVersion   sql.NullString `db:"resolved_version"`
	LastSync          time.Time      `db:"last_sync"`
	PackageName       string         `db:"package_name"`
	PackageTypeID     int            `db:"package_type_id"`
	PackageTypeName   string         `db:"package_type_name"`
	PackageTypeLabel  string         `db:"package_type_label"`
	PackageLastSync   time.Time      `db:"package_last_sync"`
	ProjectID         int            `db:"project_id"`
	ProjectName       string         `db:"project_name"`
	ProjectURL        string         `db:"project_url"`
	ProjectBranchName string         `db:"project_branch_name"`
}

func dependencyWithDetailsToDomain(m ProjectDependencyWithDetails) entity.ProjectDependency {
	return entity.ProjectDependency{
		ID:                m.ID,
		ProjectBranchID:   m.ProjectBranchID,
		PackageID:         m.PackageID,
		VersionConstraint: m.VersionConstraint,
		ResolvedVersion:   nullStringToString(m.ResolvedVersion),
		SourceFile:        m.SourceFile,
		LastSync:          m.LastSync,
		Package: &entity.Package{
			ID:       m.PackageID,
			Name:     m.PackageName,
			TypeID:   m.PackageTypeID,
			LastSync: m.PackageLastSync,
		},
		PackageType: &entity.PackageType{
			ID:    m.PackageTypeID,
			Name:  m.PackageTypeName,
			Label: m.PackageTypeLabel,
		},
		Project: &entity.Project{
			ID:   m.ProjectID,
			Name: m.ProjectName,
			URL:  m.ProjectURL,
		},
		Branch: &entity.Branch{
			Name: m.ProjectBranchName,
		},
	}
}

func nullStringToString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}
