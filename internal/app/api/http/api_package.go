package http

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"time"

	"github.com/bziks/gitlab-package-finder/internal/domain/entity"
	oapi "github.com/bziks/gitlab-package-finder/pkg/oapi"
)

const internalErrorMessage = "internal server error"

func (api *API) InternalGetPackageTypes(ctx context.Context, request oapi.InternalGetPackageTypesRequestObject) (oapi.InternalGetPackageTypesResponseObject, error) {
	items, err := api.packageTypeRepo.GetAll(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "get package types", "error", err)
		return oapi.InternalGetPackageTypes500JSONResponse{
			Error: oapi.ErrorWithDetailsResponse{
				Message: internalErrorMessage,
			},
			Meta: &oapi.MetaResponse{
				Timestamp: time.Now(),
			},
		}, nil
	}

	ptl := make([]oapi.PackageTypeResponse, len(items))
	for i := range items {
		ptl[i] = oapi.PackageTypeResponse{
			Name:  items[i].Name,
			Label: items[i].Label,
		}
	}

	return oapi.InternalGetPackageTypes200JSONResponse{
		Data: ptl,
		Meta: oapi.MetaResponse{
			Timestamp: time.Now(),
		},
	}, nil
}

func (api *API) InternalGetPackagesBySearch(ctx context.Context, request oapi.InternalGetPackagesBySearchRequestObject) (oapi.InternalGetPackagesBySearchResponseObject, error) {
	page := 1
	if request.Params.Page != nil {
		page = *request.Params.Page
	}

	if page < 1 {
		return oapi.InternalGetPackagesBySearch400JSONResponse{
			Error: oapi.ErrorWithValidationDetailsResponse{
				Message: "`page` can't be less than 1",
			},
			Meta: oapi.MetaResponse{
				Timestamp: time.Now(),
			},
		}, nil
	}

	output, err := api.packageService.GetPackagesBySearch(ctx, request.SearchId, page)
	if err != nil {
		var ve entity.ValidationError
		if errors.As(err, &ve) {
			return oapi.InternalGetPackagesBySearch400JSONResponse{
				Error: oapi.ErrorWithValidationDetailsResponse{
					Message: err.Error(),
					Details: []oapi.ValidationError{
						{
							Field:  ve.Field,
							Reason: ve.Reason,
						},
					},
				},
				Meta: oapi.MetaResponse{
					Timestamp: time.Now(),
				},
			}, nil
		}

		slog.ErrorContext(ctx, "get packages by search", "error", err)
		return oapi.InternalGetPackagesBySearch500JSONResponse{
			Error: oapi.ErrorWithDetailsResponse{
				Message: internalErrorMessage,
			},
			Meta: &oapi.MetaResponse{
				Timestamp: time.Now(),
			},
		}, nil
	}

	packages := make([]oapi.PackageResponse, len(output.Dependencies))
	for i := range output.Dependencies {
		dep := output.Dependencies[i]

		packages[i] = oapi.PackageResponse{
			Name:              dep.Package.Name,
			VersionConstraint: dep.VersionConstraint,
			ResolvedVersion:   &dep.ResolvedVersion,
			SourceFile:        dep.SourceFile,
			LastSync:          dep.LastSync,
			Project: oapi.ProjectForPackageResponse{
				Id:     dep.Project.ID,
				Name:   dep.Project.Name,
				Url:    dep.Project.URL,
				Branch: dep.Branch.Name,
			},
		}
	}

	limit := 10 // TODO: Get limit from one place
	totalPages := math.Ceil(float64(output.Total) / float64(limit))

	return oapi.InternalGetPackagesBySearch200JSONResponse{
		Data: oapi.PackageSearchResultResponse{
			Packages:         packages,
			SearchFinished:   output.SearchFinished,
			RepositoriesLeft: output.RepositoriesLeft,
			Status:           output.Status,
		},
		Meta: oapi.MetaPaginationResponse{
			Timestamp: time.Now(),
			Pagination: oapi.PaginationResponse{
				Limit:      limit,
				Page:       page,
				Total:      output.Total,
				TotalPages: int(totalPages),
			},
		},
	}, nil
}

func (api *API) InternalSearchPackages(ctx context.Context, request oapi.InternalSearchPackagesRequestObject) (oapi.InternalSearchPackagesResponseObject, error) {
	searchID, err := api.searchService.StartSearch(ctx, request.Params.Type, request.Params.Query)
	if err != nil {
		var ve entity.ValidationError
		if errors.As(err, &ve) {
			return oapi.InternalSearchPackages400JSONResponse{
				Error: oapi.ErrorWithValidationDetailsResponse{
					Message: err.Error(),
					Details: []oapi.ValidationError{
						{
							Field:  ve.Field,
							Reason: ve.Reason,
						},
					},
				},
				Meta: oapi.MetaResponse{
					Timestamp: time.Now(),
				},
			}, nil
		}

		slog.ErrorContext(ctx, "search packages", "error", err)
		return oapi.InternalSearchPackages500JSONResponse{
			Error: oapi.ErrorWithDetailsResponse{
				Message: internalErrorMessage,
			},
			Meta: &oapi.MetaResponse{
				Timestamp: time.Now(),
			},
		}, nil
	}

	return oapi.InternalSearchPackages200JSONResponse{
		Data: oapi.PackageSearchResponse{
			SearchId: searchID,
		},
		Meta: oapi.MetaResponse{
			Timestamp: time.Now(),
		},
	}, nil
}

func (api *API) InternalGetFailedRepositories(ctx context.Context, request oapi.InternalGetFailedRepositoriesRequestObject) (oapi.InternalGetFailedRepositoriesResponseObject, error) {
	repos, err := api.packageSearchStorage.GetFailedRepositories(ctx, request.SearchId)
	if err != nil {
		slog.ErrorContext(ctx, "get failed repositories", "error", err)
		return oapi.InternalGetFailedRepositories500JSONResponse{
			Error: oapi.ErrorWithDetailsResponse{
				Message: internalErrorMessage,
			},
			Meta: &oapi.MetaResponse{
				Timestamp: time.Now(),
			},
		}, nil
	}

	failedRepos := make([]oapi.FailedRepositoryResponse, len(repos))
	for i, repo := range repos {
		failedRepos[i] = oapi.FailedRepositoryResponse{
			BranchId:    int32(repo.BranchID),
			BranchName:  repo.BranchName,
			Error:       repo.Error,
			ProjectId:   int32(repo.ProjectID),
			ProjectName: repo.ProjectName,
			ProjectUrl:  repo.ProjectURL,
			Timestamp:   repo.Timestamp,
		}
	}

	return oapi.InternalGetFailedRepositories200JSONResponse{
		Data: failedRepos,
		Meta: oapi.MetaResponse{
			Timestamp: time.Now(),
		},
	}, nil
}
