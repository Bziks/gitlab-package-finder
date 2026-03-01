package projectssync

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/bziks/gitlab-package-finder/internal/domain/entity"
	projectservice "github.com/bziks/gitlab-package-finder/internal/services/project"
)

const gitlabPerPage = 100

type Command struct {
	gitlabClient   *gitlab.Client
	projectService *projectservice.Service
	languageMap    map[string]string
}

func New(gitlabClient *gitlab.Client, projectService *projectservice.Service, languageMap map[string]string) *Command {
	return &Command{
		gitlabClient:   gitlabClient,
		projectService: projectService,
		languageMap:    languageMap,
	}
}

func (c *Command) Execute(ctx context.Context) error {
	// get total pages
	membership := true
	_, response, err := c.gitlabClient.Projects.ListProjects(&gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: gitlabPerPage,
			Page:    1,
		},
		Membership: &membership,
	})
	if err != nil {
		return err
	}

	slog.InfoContext(ctx, fmt.Sprintf("Total items: %d, Total pages: %d", response.TotalItems, response.TotalPages))

	semaphore := make(chan struct{}, 2)
	var wg sync.WaitGroup

	for i := int64(1); i <= response.TotalPages; i++ {
		// Block if max concurrency is reached, respect context cancellation
		select {
		case semaphore <- struct{}{}:
		case <-ctx.Done():
			wg.Wait()
			return ctx.Err()
		}

		wg.Add(1)
		go func(ctx context.Context, currentPage int64) {
			defer wg.Done()
			defer func() { <-semaphore }()
			c.checkProjectsChunk(ctx, currentPage)
		}(ctx, i)
	}

	wg.Wait()

	return nil
}

func (c *Command) checkProjectsChunk(ctx context.Context, page int64) {
	const op = "check projects"

	slog.InfoContext(ctx, fmt.Sprintf("Page: %d", page))

	membership := true
	projects, _, err := c.gitlabClient.Projects.ListProjects(
		&gitlab.ListProjectsOptions{
			ListOptions: gitlab.ListOptions{
				PerPage: gitlabPerPage,
				Page:    page,
			},
			Membership: &membership,
		},
		gitlab.WithContext(ctx),
	)
	if err != nil {
		slog.ErrorContext(ctx, fmt.Errorf("%s: %w", op, err).Error())
		return
	}

	if len(projects) == 0 {
		slog.DebugContext(ctx, "No projects found")
		return
	}

	for _, project := range projects {
		slog.DebugContext(ctx, "process project", "name", project.Name, "url", project.WebURL)

		languages, _, langErr := c.gitlabClient.Projects.GetProjectLanguages(project.ID)
		if langErr != nil {
			slog.WarnContext(ctx, "failed to get project languages", "project", project.Name, "error", langErr)
			continue
		}

		seen := make(map[string]struct{})
		var packageTypeNames []string
		if languages != nil {
			for lang := range *languages {
				if pkgType, ok := c.languageMap[lang]; ok {
					if _, exists := seen[pkgType]; !exists {
						seen[pkgType] = struct{}{}
						packageTypeNames = append(packageTypeNames, pkgType)
					}
				}
			}
		}

		if len(packageTypeNames) == 0 {
			slog.DebugContext(ctx, "skipping project without supported package types", "name", project.Name)
			continue
		}

		err := c.projectService.UpsertWithDefaultBranch(ctx, entity.Project{
			ID:   int(project.ID),
			Name: project.Name,
			URL:  project.WebURL,
		}, project.DefaultBranch)
		if err != nil {
			slog.WarnContext(ctx, err.Error())
			continue
		}

		if syncErr := c.projectService.SyncPackageTypes(ctx, int(project.ID), packageTypeNames); syncErr != nil {
			slog.WarnContext(ctx, "failed to sync package types", "project", project.Name, "error", syncErr)
		}
	}
}
