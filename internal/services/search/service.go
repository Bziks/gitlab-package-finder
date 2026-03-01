package search

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bziks/gitlab-package-finder/internal/app/filefinder"
	"github.com/bziks/gitlab-package-finder/internal/domain/entity"
	"github.com/bziks/gitlab-package-finder/internal/domain/packagemanager"
	domainsearch "github.com/bziks/gitlab-package-finder/internal/domain/search"
	"github.com/bziks/gitlab-package-finder/internal/ports"
)

type Service struct {
	packageSearchStorage  ports.PackageSearchStorage
	projectRepository     ports.ProjectRepository
	packageTypeRepo       ports.PackageTypeRepository
	packageRepo           ports.PackageRepository
	fileFinderFactory     *filefinder.Factory
	packageManagerFactory *packagemanager.Factory
}

func NewService(
	packageSearchStorage ports.PackageSearchStorage,
	projectRepo ports.ProjectRepository,
	packageTypeRepo ports.PackageTypeRepository,
	packageRepo ports.PackageRepository,
	fileFinderFactory *filefinder.Factory,
	packageManagerFactory *packagemanager.Factory,
) *Service {
	return &Service{
		packageSearchStorage:  packageSearchStorage,
		projectRepository:     projectRepo,
		packageTypeRepo:       packageTypeRepo,
		packageRepo:           packageRepo,
		fileFinderFactory:     fileFinderFactory,
		packageManagerFactory: packageManagerFactory,
	}
}

func (s *Service) StartSearch(ctx context.Context, packageType, query string) (string, error) {
	if packageType == "" {
		return "", entity.ValidationError{Err: entity.ErrInvalidType, Field: "type", Reason: "type is required"}
	}

	if query == "" {
		return "", entity.ValidationError{Err: entity.ErrInvalidQuery, Field: "query", Reason: "query is required"}
	}

	searchPackage, err := domainsearch.ParseQuery(query)
	if err != nil {
		return "", entity.ValidationError{Err: err, Field: "query", Reason: "invalid query"}
	}

	searchPackage.Type = packageType

	searchID := domainsearch.GenerateSearchID(searchPackage.Type, searchPackage.Name, searchPackage.Version)

	// Atomically try to claim this search — prevents duplicate scans from concurrent requests
	acquired, err := s.packageSearchStorage.AcquireSearch(ctx, searchID)
	if err != nil {
		return "", fmt.Errorf("acquire search: %w", err)
	}

	if !acquired {
		// Another request already owns this search (or it's still cached)
		return searchID, nil
	}

	// Check if DB already has results for this package (cached from a previous search)
	deps, err := s.packageRepo.FindDependencies(ctx, 1, packageType, searchPackage.Name)
	if err == nil && deps.Total > 0 {
		// Results exist in DB — mark as completed (no scan needed)
		if err = s.packageSearchStorage.AddSearchToQueue(ctx, searchID, searchPackage); err != nil {
			return "", fmt.Errorf("add search to queue: %w", err)
		}
		if err = s.packageSearchStorage.CompleteSearch(ctx, searchID); err != nil {
			return "", fmt.Errorf("complete search: %w", err)
		}
		slog.InfoContext(ctx, "returning cached results from DB", "searchID", searchID, "total", deps.Total)
		return searchID, nil
	}

	if err := s.addProjectsToQueue(ctx, searchID, packageType); err != nil {
		return "", fmt.Errorf("add projects to queue: %w", err)
	}

	err = s.packageSearchStorage.AddSearchToQueue(ctx, searchID, searchPackage)
	if err != nil {
		return "", fmt.Errorf("add search to queue: %w", err)
	}

	return searchID, nil
}

func (s *Service) ProcessProject(ctx context.Context, searchID string, searchPackage *entity.SearchPackage, project *entity.DetailedProject) {
	// Get FileFinder for searching dependency files
	fileFinder, err := s.fileFinderFactory.Get(searchPackage.Type)
	if err != nil {
		slog.ErrorContext(ctx, "get file finder", "error", err)
		return
	}

	// Get PackageManager for parsing files
	pkgManager, err := s.packageManagerFactory.Get(searchPackage.Type)
	if err != nil {
		slog.ErrorContext(ctx, "get package manager", "error", err)
		return
	}

	// Get file patterns for search
	patterns := pkgManager.GetFilePatterns()

	for _, branch := range project.Branches {
		// Find all dependency files (manifest + lockfiles)
		dependencyFiles, err := fileFinder.FindDependencyFiles(ctx, searchPackage.Name, project.ID, branch.Name, patterns)
		if err != nil {
			slog.ErrorContext(ctx, "find dependency files", "error", err)
			s.packageSearchStorage.AddFailedRepository(ctx, searchID, project, branch, fmt.Sprintf("find dependency files: %v", err))
			continue
		}

		if len(dependencyFiles) == 0 {
			slog.InfoContext(ctx, "no dependency files found", "branch", branch.Name)
			continue
		}

		slog.InfoContext(ctx, "found dependency file snippets", "count", len(dependencyFiles), "branch", branch.Name)

		// Parse dependencies through PackageManager
		parseResult, err := pkgManager.ParseDependency(ctx, dependencyFiles, searchPackage.Name)
		if err != nil {
			slog.ErrorContext(ctx, "parse dependency", "error", err, "branch", branch.Name)
			s.packageSearchStorage.AddFailedRepository(ctx, searchID, project, branch, fmt.Sprintf("parse dependency: %v", err))
			continue
		}

		if !parseResult.Found {
			slog.InfoContext(ctx, "package not found", "branch", branch.Name)
			continue
		}

		slog.InfoContext(ctx, "package found",
			"package", searchPackage.Name,
			"searchVersion", searchPackage.Version,
			"versionConstraint", parseResult.VersionConstraint,
			"resolvedVersion", parseResult.ResolvedVersion,
			"sourceFile", parseResult.SourceFile,
		)

		// Get package type from DB
		dbType, err := s.packageTypeRepo.GetByName(ctx, searchPackage.Type)
		if err != nil {
			slog.ErrorContext(ctx, "get package type by name", "error", err)
			s.packageSearchStorage.AddFailedRepository(ctx, searchID, project, branch, fmt.Sprintf("get package type by name: %v", err))
			continue
		}

		// Get or create package in DB
		dbPackage, err := s.packageRepo.GetByNameAndType(ctx, searchPackage.Name, dbType.ID)
		if err != nil {
			if err == entity.ErrPackageNotFound {
				dbPackage, err = s.packageRepo.Create(ctx, searchPackage.Name, dbType.ID)
				if err != nil {
					slog.ErrorContext(ctx, "create package", "error", err)
					s.packageSearchStorage.AddFailedRepository(ctx, searchID, project, branch, fmt.Sprintf("create package: %v", err))
					continue
				}
			} else {
				slog.ErrorContext(ctx, "get package by name and type", "error", err)
				s.packageSearchStorage.AddFailedRepository(ctx, searchID, project, branch, fmt.Sprintf("get package by name and type: %v", err))
				continue
			}
		}

		// Save dependency
		err = s.packageRepo.AttachPackageToProjectDependency(
			ctx,
			branch.ID,
			dbPackage.ID,
			parseResult.SourceFile,
			parseResult.VersionConstraint,
			parseResult.ResolvedVersion,
		)
		if err != nil {
			slog.ErrorContext(ctx, "attach package to project dependency", "error", err)
			s.packageSearchStorage.AddFailedRepository(ctx, searchID, project, branch, fmt.Sprintf("attach package to project dependency: %v", err))
			continue
		}
	}
}

func (s *Service) addProjectsToQueue(ctx context.Context, searchID string, packageType string) error {
	slog.InfoContext(ctx, "add projects to queue", "searchID", searchID, "packageType", packageType)

	limit := 50
	page := 1

	for {
		projects, err := s.projectRepository.GetDetailedByPackageTypeWithPagination(ctx, packageType, page, limit)
		if err != nil {
			return fmt.Errorf("get projects page %d: %w", page, err)
		}

		if len(projects.Projects) == 0 {
			break
		}

		for _, project := range projects.Projects {
			if err := s.packageSearchStorage.AddProjectsToQueue(ctx, searchID, project); err != nil {
				slog.ErrorContext(ctx, "add projects to queue", "error", err)
				continue
			}
		}

		page++
	}

	return nil
}
