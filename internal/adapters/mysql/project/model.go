package project

import (
	"strconv"
	"strings"

	"github.com/bziks/gitlab-package-finder/internal/domain/entity"
)

type Project struct {
	ID   int
	Name string
	URL  string
}

type ProjectWithPackageTypes struct {
	Project
	PackageTypes *string `db:"package_types"`
}

type DetailedProject struct {
	Project
	BranchIDs *string `db:"branch_ids"`
	Branches  *string `db:"branches"`
}

func toDomain(m Project) entity.Project {
	return entity.Project{
		ID:   m.ID,
		Name: m.Name,
		URL:  m.URL,
	}
}

func fromDomain(e entity.Project) Project {
	return Project{
		ID:   e.ID,
		Name: e.Name,
		URL:  e.URL,
	}
}

func toProjectWithPackageTypesDomain(m ProjectWithPackageTypes) entity.ProjectWithPackageTypes {
	var packageTypes []string
	if m.PackageTypes != nil && *m.PackageTypes != "" {
		packageTypes = strings.Split(*m.PackageTypes, ",")
	}

	return entity.ProjectWithPackageTypes{
		Project: entity.Project{
			ID:   m.ID,
			Name: m.Name,
			URL:  m.URL,
		},
		PackageTypes: packageTypes,
	}
}

func toDetailedDomain(m DetailedProject) entity.DetailedProject {
	var branches []entity.Branch

	if m.BranchIDs != nil && m.Branches != nil && *m.BranchIDs != "" && *m.Branches != "" {
		branchIDs := strings.Split(*m.BranchIDs, "||")
		branchNames := strings.Split(*m.Branches, "||")

		branches = make([]entity.Branch, 0, len(branchNames))
		for i, name := range branchNames {
			if i >= len(branchIDs) {
				break
			}
			branchID, _ := strconv.Atoi(branchIDs[i])
			branches = append(branches, entity.Branch{
				ID:   branchID,
				Name: name,
			})
		}
	}

	return entity.DetailedProject{
		Project: entity.Project{
			ID:   m.ID,
			Name: m.Name,
			URL:  m.URL,
		},
		Branches: branches,
	}
}
