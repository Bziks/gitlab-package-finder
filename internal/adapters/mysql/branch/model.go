package branch

import "github.com/bziks/gitlab-package-finder/internal/domain/entity"

type Branch struct {
	ID        int
	ProjectID int `db:"project_id"`
	Name      string
}

func toDomain(m Branch) entity.Branch {
	return entity.Branch{
		ID:   m.ID,
		Name: m.Name,
	}
}
