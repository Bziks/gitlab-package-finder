package packagetype

import "github.com/bziks/gitlab-package-finder/internal/domain/entity"

type PackageType struct {
	ID    int
	Name  string
	Label string
}

func toDomain(m PackageType) entity.PackageType {
	return entity.PackageType{
		ID:    m.ID,
		Name:  m.Name,
		Label: m.Label,
	}
}
