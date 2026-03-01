package http

import (
	"github.com/bziks/gitlab-package-finder/internal/adapters/metrics"
	"github.com/bziks/gitlab-package-finder/internal/ports"
	"github.com/bziks/gitlab-package-finder/internal/services/packagesvc"
	projectservice "github.com/bziks/gitlab-package-finder/internal/services/project"
	"github.com/bziks/gitlab-package-finder/internal/services/search"
	oapi "github.com/bziks/gitlab-package-finder/pkg/oapi"
)

var _ oapi.StrictServerInterface = (*API)(nil)

type API struct {
	packageTypeRepo      ports.PackageTypeRepository
	projectRepo          ports.ProjectRepository
	packageSearchStorage ports.PackageSearchStorage
	searchService        *search.Service
	packageService       *packagesvc.Service
	projectService       *projectservice.Service
	metrics              *metrics.Metrics
	corsAllowOrigin      string
}

func NewAPI(
	packageTypeRepo ports.PackageTypeRepository,
	projectRepo ports.ProjectRepository,
	packageSearchStorage ports.PackageSearchStorage,
	searchService *search.Service,
	packageService *packagesvc.Service,
	projectService *projectservice.Service,
	m *metrics.Metrics,
	corsAllowOrigin string,
) *API {
	return &API{
		packageTypeRepo:      packageTypeRepo,
		projectRepo:          projectRepo,
		packageSearchStorage: packageSearchStorage,
		searchService:        searchService,
		packageService:       packageService,
		projectService:       projectService,
		metrics:              m,
		corsAllowOrigin:      corsAllowOrigin,
	}
}
