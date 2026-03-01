package entity

type Project struct {
	ID   int
	Name string
	URL  string
}

type ProjectsWithPagination struct {
	Total    int
	Projects []Project
}

type ProjectWithPackageTypes struct {
	Project
	PackageTypes []string
}

type ProjectsWithPackageTypesAndPagination struct {
	Total    int
	Projects []ProjectWithPackageTypes
}

type DetailedProject struct {
	Project
	Branches []Branch
}

type DetailedProjectsWithPagination struct {
	Total    int
	Projects []DetailedProject
}
