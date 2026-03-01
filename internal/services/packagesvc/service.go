package packagesvc

import (
	"context"
	"fmt"

	"github.com/bziks/gitlab-package-finder/internal/domain/entity"
	"github.com/bziks/gitlab-package-finder/internal/domain/search"
	"github.com/bziks/gitlab-package-finder/internal/ports"
)

// Result is the return value of GetPackagesBySearch.
type Result struct {
	Dependencies     []entity.ProjectDependency
	SearchFinished   bool
	Total            int
	RepositoriesLeft int64
	Status           string
}

// Service orchestrates package and package-search storage calls.
type Service struct {
	packageRepo          ports.PackageRepository
	packageSearchStorage ports.PackageSearchStorage
}

// NewService creates a new PackageService.
func NewService(packageRepo ports.PackageRepository, packageSearchStorage ports.PackageSearchStorage) *Service {
	return &Service{
		packageRepo:          packageRepo,
		packageSearchStorage: packageSearchStorage,
	}
}

// GetPackagesBySearch returns paginated dependencies for a search, plus search status.
func (s *Service) GetPackagesBySearch(ctx context.Context, searchID string, page int) (*Result, error) {
	const op = "get packages by search"

	if searchID == "" {
		return nil, entity.ValidationError{
			Err:    entity.ErrInvalidQuery,
			Field:  "search_id",
			Reason: "search_id is required",
		}
	}
	if page < 1 {
		return nil, entity.ValidationError{
			Err:    entity.ErrInvalidQuery,
			Field:  "page",
			Reason: "page must be greater than 0",
		}
	}

	isRunning, err := s.packageSearchStorage.CheckIfSearchIsRunning(ctx, searchID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	searchDetails, err := s.packageSearchStorage.GetSearchDetails(ctx, searchID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if searchDetails == nil {
		return nil, fmt.Errorf("%s: search not found: %s", op, searchID)
	}

	var deps []entity.ProjectDependency
	var total int

	if searchDetails.Version != "" {
		deps, total, err = s.findFilteredDependencies(ctx, searchDetails, page)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
	} else {
		depsWithPagination, err := s.packageRepo.FindDependencies(ctx, page, searchDetails.Type, searchDetails.Name)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		deps = depsWithPagination.Dependencies
		total = depsWithPagination.Total
	}

	repositoriesLeft, err := s.packageSearchStorage.GetProjectsQueueLength(ctx, searchID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	status, err := s.packageSearchStorage.GetSearchStatus(ctx, searchID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Result{
		Dependencies:     deps,
		SearchFinished:   !isRunning,
		Total:            total,
		RepositoriesLeft: repositoriesLeft,
		Status:           status,
	}, nil
}

// findFilteredDependencies fetches all dependencies for a package and filters
// them by version constraint using cursor-based over-fetching. It returns the
// page of matching results and the total number of matches.
func (s *Service) findFilteredDependencies(ctx context.Context, sp *entity.SearchPackage, page int) ([]entity.ProjectDependency, int, error) {
	const batchSize = 100
	const pageSize = 10

	offset := (page - 1) * pageSize
	var matched []entity.ProjectDependency
	total := 0
	afterID := 0

	for {
		batch, err := s.packageRepo.FindDependenciesCursor(ctx, sp.Type, sp.Name, afterID, batchSize)
		if err != nil {
			return nil, 0, err
		}

		for _, dep := range batch {
			if !search.MatchesVersion(sp.Version, dep.ResolvedVersion) {
				continue
			}
			total++
			// Skip until we reach the requested page offset.
			if total <= offset {
				continue
			}
			// Collect up to pageSize results for the current page.
			if len(matched) < pageSize {
				matched = append(matched, dep)
			}
		}

		if len(batch) < batchSize {
			break
		}
		afterID = batch[len(batch)-1].ID
	}

	return matched, total, nil
}

// GetPackagesCount returns the total number of packages.
func (s *Service) GetPackagesCount(ctx context.Context) (int, error) {
	return s.packageRepo.GetCount(ctx)
}
