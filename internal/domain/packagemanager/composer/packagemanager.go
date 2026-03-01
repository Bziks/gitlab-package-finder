package composer

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"

	"github.com/bziks/gitlab-package-finder/internal/domain/entity"
)

type ComposerPackageManager struct{}

func NewPackageManager() *ComposerPackageManager {
	return &ComposerPackageManager{}
}

func (pm *ComposerPackageManager) GetLanguages() []string {
	return []string{"PHP"}
}

func (pm *ComposerPackageManager) GetFilePatterns() entity.FilePatterns {
	return entity.FilePatterns{
		Manifests: []string{"composer.json"},
		Lockfiles: []string{"composer.lock"},
	}
}

func (pm *ComposerPackageManager) ParseDependency(ctx context.Context, files []entity.DependencyFile, packageName string) (*entity.ParseResult, error) {
	result := &entity.ParseResult{
		Found: false,
	}

	var versionConstraint, resolvedVersion, sourceFile string

	// Parse snippets in order, taking first match for each file type
	for _, file := range files {
		if file.FileType == entity.FileTypeManifest && versionConstraint == "" {
			// Try to parse manifest snippet
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
			// Try to parse lockfile snippet
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

func (pm *ComposerPackageManager) parseManifest(content string, packageName string) (string, error) {
	// Content contains only a few lines around the package name
	// Pattern: "package-name": "version-constraint",
	escapedPackage := regexp.QuoteMeta(packageName)
	pattern := fmt.Sprintf(`"%s"\s*:\s*"([^"]+)"`, escapedPackage)

	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("failed to compile regex: %w", err)
	}

	matches := re.FindStringSubmatch(content)
	if len(matches) < 2 {
		return "", nil // Package not found
	}

	return matches[1], nil
}

func (pm *ComposerPackageManager) parseLockfile(content string, packageName string) (string, error) {
	// Content contains only a few lines around the package name
	// Pattern in composer.lock:
	// "name": "package-name",
	// ...
	// "version": "v1.2.3",

	// First, verify the package name is present
	escapedPackage := regexp.QuoteMeta(packageName)
	namePattern := fmt.Sprintf(`"name"\s*:\s*"%s"`, escapedPackage)
	if matched, _ := regexp.MatchString(namePattern, content); !matched {
		return "", nil
	}

	// Extract version from the content snippet
	versionPattern := `"version"\s*:\s*"([^"]+)"`
	re, err := regexp.Compile(versionPattern)
	if err != nil {
		return "", fmt.Errorf("failed to compile regex: %w", err)
	}

	matches := re.FindStringSubmatch(content)
	if len(matches) < 2 {
		return "", nil
	}

	return matches[1], nil
}
