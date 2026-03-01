package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/bziks/gitlab-package-finder/internal/domain/entity"
	oapi "github.com/bziks/gitlab-package-finder/pkg/oapi"
)

const (
	defaultProjectsLimit = 10
	maxProjectsLimit     = 50
)

func (api *API) InternalGetBranches(ctx context.Context, request oapi.InternalGetBranchesRequestObject) (oapi.InternalGetBranchesResponseObject, error) {
	branches, err := api.projectService.GetBranches(ctx, request.ProjectId)
	if err != nil {
		slog.ErrorContext(ctx, "get branches", "error", err)
		return oapi.InternalGetBranches500JSONResponse{
			Error: oapi.ErrorWithDetailsResponse{
				Message: internalErrorMessage,
			},
			Meta: &oapi.MetaResponse{
				Timestamp: time.Now(),
			},
		}, nil
	}

	bl := make([]oapi.BranchForProjectResponse, len(branches))
	for i := range branches {
		bl[i] = oapi.BranchForProjectResponse{
			Id:   branches[i].ID,
			Name: branches[i].Name,
		}
	}

	return oapi.InternalGetBranches200JSONResponse{
		Data: bl,
		Meta: oapi.MetaResponse{
			Timestamp: time.Now(),
		},
	}, nil
}

func (api *API) InternalCreateBranch(ctx context.Context, request oapi.InternalCreateBranchRequestObject) (oapi.InternalCreateBranchResponseObject, error) {
	if request.Body == nil || request.Body.Name == "" {
		return oapi.InternalCreateBranch400JSONResponse{
			Error: oapi.ErrorWithValidationDetailsResponse{
				Message: "branch name is required",
				Details: []oapi.ValidationError{
					{Field: "name", Reason: "must not be empty"},
				},
			},
			Meta: oapi.MetaResponse{
				Timestamp: time.Now(),
			},
		}, nil
	}

	branch, err := api.projectService.CreateBranch(ctx, request.ProjectId, request.Body.Name)
	if err != nil {
		var ve entity.ValidationError
		if errors.As(err, &ve) {
			return oapi.InternalCreateBranch400JSONResponse{
				Error: oapi.ErrorWithValidationDetailsResponse{
					Message: err.Error(),
					Details: []oapi.ValidationError{
						{Field: ve.Field, Reason: ve.Reason},
					},
				},
				Meta: oapi.MetaResponse{
					Timestamp: time.Now(),
				},
			}, nil
		}

		slog.ErrorContext(ctx, "create branch", "error", err)
		return oapi.InternalCreateBranch500JSONResponse{
			Error: oapi.ErrorWithDetailsResponse{
				Message: internalErrorMessage,
			},
			Meta: &oapi.MetaResponse{
				Timestamp: time.Now(),
			},
		}, nil
	}

	return oapi.InternalCreateBranch200JSONResponse{
		Data: oapi.BranchForProjectResponse{
			Id:   branch.ID,
			Name: branch.Name,
		},
		Meta: oapi.MetaResponse{
			Timestamp: time.Now(),
		},
	}, nil
}

func (api *API) InternalDeleteBranch(ctx context.Context, request oapi.InternalDeleteBranchRequestObject) (oapi.InternalDeleteBranchResponseObject, error) {
	err := api.projectService.DeleteBranch(ctx, request.Id)
	if err != nil {
		if errors.Is(err, entity.ErrBranchNotFound) {
			return oapi.InternalDeleteBranch404JSONResponse{
				Error: oapi.ErrorResponse{
					Message: "branch not found",
				},
				Meta: &oapi.MetaResponse{
					Timestamp: time.Now(),
				},
			}, nil
		}

		slog.ErrorContext(ctx, "delete branch", "error", err)
		return oapi.InternalDeleteBranch500JSONResponse{
			Error: oapi.ErrorWithDetailsResponse{
				Message: internalErrorMessage,
			},
			Meta: &oapi.MetaResponse{
				Timestamp: time.Now(),
			},
		}, nil
	}

	return oapi.InternalDeleteBranch204Response{}, nil
}

func (api *API) InternalGetProjects(ctx context.Context, request oapi.InternalGetProjectsRequestObject) (oapi.InternalGetProjectsResponseObject, error) {
	page := 1
	if request.Params.Page != nil {
		page = *request.Params.Page
	}

	limit := defaultProjectsLimit
	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}

	if limit < 1 || limit > maxProjectsLimit {
		return oapi.InternalGetProjects400JSONResponse{
			Error: oapi.ErrorWithValidationDetailsResponse{
				Message: fmt.Sprintf("`limit` must be between 1 and %d", maxProjectsLimit),
			},
			Meta: oapi.MetaResponse{
				Timestamp: time.Now(),
			},
		}, nil
	}

	if page < 1 {
		return oapi.InternalGetProjects400JSONResponse{
			Error: oapi.ErrorWithValidationDetailsResponse{
				Message: "`page` can't be less than 1",
			},
			Meta: oapi.MetaResponse{
				Timestamp: time.Now(),
			},
		}, nil
	}

	result, err := api.projectRepo.GetWithPackageTypesAndPagination(ctx, page, limit)
	if err != nil {
		slog.ErrorContext(ctx, "get projects", "error", err)
		return oapi.InternalGetProjects500JSONResponse{
			Error: oapi.ErrorWithDetailsResponse{
				Message: internalErrorMessage,
			},
			Meta: &oapi.MetaResponse{
				Timestamp: time.Now(),
			},
		}, nil
	}

	pl := make([]oapi.ProjectResponse, len(result.Projects))
	for i := range result.Projects {
		pt := result.Projects[i].PackageTypes
		pl[i] = oapi.ProjectResponse{
			Id:           result.Projects[i].ID,
			Name:         result.Projects[i].Name,
			Url:          result.Projects[i].URL,
			PackageTypes: &pt,
		}
	}

	totalPages := math.Ceil(float64(result.Total) / float64(limit))
	if totalPages == 0 {
		totalPages = 1
	}

	return oapi.InternalGetProjects200JSONResponse{
		Data: pl,
		Meta: oapi.MetaPaginationResponse{
			Timestamp: time.Now(),
			Pagination: oapi.PaginationResponse{
				Limit:      limit,
				Page:       page,
				Total:      result.Total,
				TotalPages: int(totalPages),
			},
		},
	}, nil
}

func (api *API) InternalSearchProjects(ctx context.Context, request oapi.InternalSearchProjectsRequestObject) (oapi.InternalSearchProjectsResponseObject, error) {
	page := 1
	if request.Params.Page != nil {
		page = *request.Params.Page
	}

	limit := defaultProjectsLimit
	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}

	if limit < 1 || limit > maxProjectsLimit {
		return oapi.InternalSearchProjects400JSONResponse{
			Error: oapi.ErrorWithValidationDetailsResponse{
				Message: fmt.Sprintf("`limit` must be between 1 and %d", maxProjectsLimit),
			},
			Meta: oapi.MetaResponse{
				Timestamp: time.Now(),
			},
		}, nil
	}

	if page < 1 {
		return oapi.InternalSearchProjects400JSONResponse{
			Error: oapi.ErrorWithValidationDetailsResponse{
				Message: "`page` can't be less than 1",
			},
			Meta: oapi.MetaResponse{
				Timestamp: time.Now(),
			},
		}, nil
	}

	result, err := api.projectService.SearchByName(ctx, request.Params.Query, page, limit)
	if err != nil {
		slog.ErrorContext(ctx, "search projects", "error", err)
		return oapi.InternalSearchProjects500JSONResponse{
			Error: oapi.ErrorWithDetailsResponse{
				Message: internalErrorMessage,
			},
			Meta: &oapi.MetaResponse{
				Timestamp: time.Now(),
			},
		}, nil
	}

	pl := make([]oapi.ProjectResponse, len(result.Projects))
	for i := range result.Projects {
		pt := result.Projects[i].PackageTypes
		pl[i] = oapi.ProjectResponse{
			Id:           result.Projects[i].ID,
			Name:         result.Projects[i].Name,
			Url:          result.Projects[i].URL,
			PackageTypes: &pt,
		}
	}

	totalPages := math.Ceil(float64(result.Total) / float64(limit))
	if totalPages == 0 {
		totalPages = 1
	}

	return oapi.InternalSearchProjects200JSONResponse{
		Data: pl,
		Meta: oapi.MetaPaginationResponse{
			Timestamp: time.Now(),
			Pagination: oapi.PaginationResponse{
				Limit:      limit,
				Page:       page,
				Total:      result.Total,
				TotalPages: int(totalPages),
			},
		},
	}, nil
}

func (api *API) InternalStatistics(ctx context.Context, request oapi.InternalStatisticsRequestObject) (oapi.InternalStatisticsResponseObject, error) {
	projectsCount, err := api.projectRepo.GetCount(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "get projects count", "error", err)
		return oapi.InternalStatistics500JSONResponse{
			Error: oapi.ErrorWithDetailsResponse{
				Message: internalErrorMessage,
			},
			Meta: &oapi.MetaResponse{
				Timestamp: time.Now(),
			},
		}, nil
	}

	packagesCount, err := api.packageService.GetPackagesCount(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "get packages count", "error", err)
		return oapi.InternalStatistics500JSONResponse{
			Error: oapi.ErrorWithDetailsResponse{
				Message: internalErrorMessage,
			},
			Meta: &oapi.MetaResponse{
				Timestamp: time.Now(),
			},
		}, nil
	}

	return oapi.InternalStatistics200JSONResponse{
		Data: oapi.StatisticsResponse{
			Projects: int64(projectsCount),
			Packages: int64(packagesCount),
		},
		Meta: oapi.MetaResponse{
			Timestamp: time.Now(),
		},
	}, nil
}

func (api *API) InternalDeleteProject(ctx context.Context, request oapi.InternalDeleteProjectRequestObject) (oapi.InternalDeleteProjectResponseObject, error) {
	if err := api.projectService.DeleteProject(ctx, request.ProjectId); err != nil {
		slog.ErrorContext(ctx, "delete project", "error", err)
		return oapi.InternalDeleteProject500JSONResponse{
			Error: oapi.ErrorWithDetailsResponse{
				Message: internalErrorMessage,
			},
			Meta: &oapi.MetaResponse{
				Timestamp: time.Now(),
			},
		}, nil
	}

	return oapi.InternalDeleteProject204Response{}, nil
}
