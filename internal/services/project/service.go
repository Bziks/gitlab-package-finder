package project

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bziks/gitlab-package-finder/internal/domain/entity"
	"github.com/bziks/gitlab-package-finder/internal/ports"
)

type Service struct {
	projectRepository ports.ProjectRepository
	branchRepository  ports.BranchRepository
	packageTypeRepo   ports.PackageTypeRepository
}

func NewService(projectRepo ports.ProjectRepository, branchRepo ports.BranchRepository, packageTypeRepo ports.PackageTypeRepository) *Service {
	return &Service{
		projectRepository: projectRepo,
		branchRepository:  branchRepo,
		packageTypeRepo:   packageTypeRepo,
	}
}

func (s *Service) UpsertWithDefaultBranch(ctx context.Context, project entity.Project, defaultBranch string) error {
	if err := s.projectRepository.UpSert(ctx, project); err != nil {
		return fmt.Errorf("upsert project: %w", err)
	}

	branch, err := s.branchRepository.Find(ctx, project.ID, defaultBranch)
	if err != nil {
		return fmt.Errorf("find branch: %w", err)
	}

	if branch == nil {
		if _, err := s.branchRepository.Create(ctx, project.ID, defaultBranch); err != nil {
			return fmt.Errorf("create branch: %w", err)
		}
	}

	return nil
}

func (s *Service) GetBranches(ctx context.Context, projectID int) ([]entity.Branch, error) {
	branches, err := s.branchRepository.GetByProjectID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("get branches: %w", err)
	}

	return branches, nil
}

func (s *Service) CreateBranch(ctx context.Context, projectID int, name string) (*entity.Branch, error) {
	existing, err := s.branchRepository.Find(ctx, projectID, name)
	if err != nil {
		return nil, fmt.Errorf("find branch: %w", err)
	}

	if existing != nil {
		return nil, entity.ValidationError{
			Err:    fmt.Errorf("branch already exists"),
			Field:  "name",
			Reason: fmt.Sprintf("branch '%s' already exists for this project", name),
		}
	}

	branch, err := s.branchRepository.Create(ctx, projectID, name)
	if err != nil {
		return nil, fmt.Errorf("create branch: %w", err)
	}

	return branch, nil
}

func (s *Service) DeleteBranch(ctx context.Context, branchID int) error {
	branch, err := s.branchRepository.GetByID(ctx, branchID)
	if err != nil {
		return fmt.Errorf("get branch: %w", err)
	}

	if branch == nil {
		return entity.ErrBranchNotFound
	}

	if err := s.branchRepository.Delete(ctx, branchID); err != nil {
		return fmt.Errorf("delete branch: %w", err)
	}

	return nil
}

func (s *Service) SearchByName(ctx context.Context, query string, page, limit int) (entity.ProjectsWithPackageTypesAndPagination, error) {
	result, err := s.projectRepository.SearchByNameWithPackageTypes(ctx, query, page, limit)
	if err != nil {
		return entity.ProjectsWithPackageTypesAndPagination{}, fmt.Errorf("search projects: %w", err)
	}

	return result, nil
}

func (s *Service) DeleteProject(ctx context.Context, projectID int) error {
	if err := s.projectRepository.Delete(ctx, projectID); err != nil {
		return fmt.Errorf("delete project: %w", err)
	}

	return nil
}

func (s *Service) SyncPackageTypes(ctx context.Context, projectID int, packageTypeNames []string) error {
	if len(packageTypeNames) == 0 {
		return nil
	}

	ids := make([]int, 0, len(packageTypeNames))
	for _, name := range packageTypeNames {
		pt, err := s.packageTypeRepo.GetByName(ctx, name)
		if err != nil {
			slog.WarnContext(ctx, "unknown package type, skipping", "name", name, "error", err)
			continue
		}
		ids = append(ids, pt.ID)
	}

	if len(ids) == 0 {
		slog.WarnContext(ctx, "no valid package types resolved, skipping sync", "projectID", projectID)
		return nil
	}

	if err := s.projectRepository.SyncPackageTypes(ctx, projectID, ids); err != nil {
		return fmt.Errorf("sync package types: %w", err)
	}

	return nil
}
