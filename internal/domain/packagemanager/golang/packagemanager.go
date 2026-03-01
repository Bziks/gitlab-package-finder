package golang

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"

	"github.com/bziks/gitlab-package-finder/internal/domain/entity"
)

type GoPackageManager struct{}

func NewPackageManager() *GoPackageManager {
	return &GoPackageManager{}
}

func (pm *GoPackageManager) GetLanguages() []string {
	return []string{"Go"}
}

func (pm *GoPackageManager) GetFilePatterns() entity.FilePatterns {
	return entity.FilePatterns{
		Manifests: []string{"go.mod"},
		Lockfiles: []string{"go.sum"},
	}
}

func (pm *GoPackageManager) ParseDependency(ctx context.Context, files []entity.DependencyFile, packageName string) (*entity.ParseResult, error) {
	result := &entity.ParseResult{
		Found: false,
	}

	var versionConstraint, resolvedVersion, sourceFile string

	// Parse snippets in order, taking first match for each file type
	for _, file := range files {
		if file.FileType == entity.FileTypeManifest && versionConstraint == "" {
			slog.DebugContext(ctx, "parsing manifest snippet", "file", file.FileName)
			// Try to parse go.mod snippet
			constraint, err := pm.parseManifest(file.Content, packageName)
			if err != nil {
				slog.ErrorContext(ctx, "error parsing manifest snippet", "error", err)
				continue
			}
			if constraint != "" {
				versionConstraint = constraint
				if sourceFile == "" {
					sourceFile = file.FileName
				}
			}
		} else if file.FileType == entity.FileTypeLockfile && resolvedVersion == "" {
			// Try to parse go.sum snippet
			version, err := pm.parseLockfile(file.Content, packageName)
			if err != nil {
				slog.ErrorContext(ctx, "error parsing lockfile snippet", "error", err)
				continue
			}
			if version != "" {
				resolvedVersion = version
				sourceFile = file.FileName // Prefer lockfile as source
			}
		}

		// Early exit if we found everything
		if versionConstraint != "" && resolvedVersion != "" {
			break
		}
	}

	// Check if package was found - save all found packages regardless of version
	if versionConstraint != "" || resolvedVersion != "" {
		result.Found = true
		result.VersionConstraint = versionConstraint
		result.ResolvedVersion = resolvedVersion
		result.SourceFile = sourceFile

		return result, nil
	}

	return result, nil
}

func (pm *GoPackageManager) parseManifest(content string, packageName string) (string, error) {
	// Escape special regex characters in package name
	escapedPackage := regexp.QuoteMeta(packageName)

	// Pattern to match package with version in various formats:
	// 1. "require package version" (single line require)
	// 2. "package version" (inside require block)
	// 3. "\tpackage version" (tab-indented lines)
	pattern := fmt.Sprintf(`(?m)^(?:\s*require\s+%s\s+([^\s]+)|^\s*%s\s+([^\s]+))`, escapedPackage, escapedPackage)

	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("failed to compile regex: %w", err)
	}

	matches := re.FindStringSubmatch(content)
	if len(matches) < 2 {
		return "", nil // Package not found
	}

	// Return the first non-empty capture group (version)
	for i := 1; i < len(matches); i++ {
		if matches[i] != "" {
			return matches[i], nil
		}
	}

	return "", nil
}

func (pm *GoPackageManager) parseLockfile(content string, packageName string) (string, error) {
	escapedPackage := regexp.QuoteMeta(packageName)
	pattern := fmt.Sprintf(`^%s\s+(v[^\s]+)`, escapedPackage)

	reMultiline, err := regexp.Compile(`(?m)` + pattern)
	if err != nil {
		return "", fmt.Errorf("failed to compile regex: %w", err)
	}

	matches := reMultiline.FindStringSubmatch(content)
	if len(matches) < 2 {
		return "", nil // Package not found
	}

	return matches[1], nil
}
