package gitlab

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	gitlabapi "gitlab.com/gitlab-org/api/client-go"

	"github.com/bziks/gitlab-package-finder/internal/domain/entity"
)

type FileFinder struct {
	gitlabClient *gitlabapi.Client
}

func NewFileFinder(gitlabClient *gitlabapi.Client) *FileFinder {
	return &FileFinder{
		gitlabClient: gitlabClient,
	}
}

func (f *FileFinder) FindDependencyFiles(ctx context.Context, packageName string, projectID int, branchName string, patterns entity.FilePatterns) ([]entity.DependencyFile, error) {
	const op = "find dependency files"

	searchOptions := &gitlabapi.SearchOptions{
		Ref: gitlabapi.Ptr(branchName),
	}

	slog.InfoContext(ctx, "find dependency files", "projectID", projectID, "branchName", branchName, "packageName", packageName)

	allowedFiles := make(map[string]entity.FileType)
	for _, manifest := range patterns.Manifests {
		allowedFiles[manifest] = entity.FileTypeManifest
	}
	for _, lockfile := range patterns.Lockfiles {
		allowedFiles[lockfile] = entity.FileTypeLockfile
	}

	var allBlobs []*gitlabapi.Blob
	var reqOpts []gitlabapi.RequestOptionFunc

	for {
		blobs, resp, err := f.gitlabClient.Search.BlobsByProject(projectID, packageName, searchOptions, reqOpts...)
		if err != nil {
			return make([]entity.DependencyFile, 0), fmt.Errorf("%s: %w", op, err)
		}

		allBlobs = append(allBlobs, blobs...)

		nextPage, hasNext := gitlabapi.WithNext(resp)
		if !hasNext {
			break
		}
		reqOpts = []gitlabapi.RequestOptionFunc{nextPage}
	}

	if len(allBlobs) == 0 {
		slog.InfoContext(ctx, "no files found", "packageName", packageName)
		return make([]entity.DependencyFile, 0), nil
	}

	slog.InfoContext(ctx, "found files", "count", len(allBlobs))

	dependencyFiles := make([]entity.DependencyFile, 0)
	for _, blob := range allBlobs {
		filePath := blob.Path
		fileName := filepath.Base(filePath)

		fileType, isAllowed := allowedFiles[fileName]
		if !isAllowed {
			slog.InfoContext(ctx, "skipping file", "filePath", filePath, "fileName", fileName)
			continue
		}

		slog.InfoContext(ctx, "adding file", "filePath", filePath, "fileName", fileName, "fileType", fileType)

		dependencyFiles = append(dependencyFiles, entity.DependencyFile{
			FileName: fileName,
			FilePath: filePath,
			Content:  blob.Data,
			FileType: fileType,
		})
	}

	return dependencyFiles, nil
}
