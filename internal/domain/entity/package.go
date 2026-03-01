package entity

import "time"

type Package struct {
	ID       int
	Name     string
	TypeID   int
	LastSync time.Time
}

type ProjectDependency struct {
	ID                int
	ProjectBranchID   int
	PackageID         int
	VersionConstraint string
	ResolvedVersion   string
	SourceFile        string
	LastSync          time.Time
	Package           *Package
	PackageType       *PackageType
	Project           *Project
	Branch            *Branch
}

type ProjectDependencyWithPagination struct {
	Dependencies []ProjectDependency
	Total        int
}
