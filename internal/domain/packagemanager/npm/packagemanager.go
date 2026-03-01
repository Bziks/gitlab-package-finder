package npm

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"

	"github.com/bziks/gitlab-package-finder/internal/domain/entity"
)

type NPMPackageManager struct{}

func NewPackageManager() *NPMPackageManager {
	return &NPMPackageManager{}
}

func (pm *NPMPackageManager) GetLanguages() []string {
	return []string{"JavaScript", "TypeScript"}
}

func (pm *NPMPackageManager) GetFilePatterns() entity.FilePatterns {
	return entity.FilePatterns{
		Manifests: []string{"package.json"},
		Lockfiles: []string{"package-lock.json", "yarn.lock", "pnpm-lock.yaml", "bun.lock"},
	}
}

func (pm *NPMPackageManager) ParseDependency(ctx context.Context, files []entity.DependencyFile, packageName string) (*entity.ParseResult, error) {
	result := &entity.ParseResult{
		Found: false,
	}

	var versionConstraint, resolvedVersion, sourceFile string
	lockFilePriority := map[string]int{
		"package-lock.json": 1,
		"yarn.lock":         2,
		"pnpm-lock.yaml":    3,
		"bun.lock":          4,
	}
	currentLockPriority := 999

	// Parse snippets in order, taking first match for manifest and best priority for lockfile
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
		} else if file.FileType == entity.FileTypeLockfile {
			// Check if this lockfile has better priority
			if priority, ok := lockFilePriority[file.FileName]; ok && priority < currentLockPriority {
				version, err := pm.parseLockfile(file.FileName, file.Content, packageName)
				if err != nil {
					slog.ErrorContext(ctx, "error parsing lockfile snippet", "error", err, "fileName", file.FileName)
					continue
				}
				if version != "" {
					resolvedVersion = version
					sourceFile = file.FileName // Prefer lockfile as source
					currentLockPriority = priority
				}
			}
		}

		// Early exit if we found everything with best lockfile priority
		if versionConstraint != "" && currentLockPriority == 1 {
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

func (pm *NPMPackageManager) parseManifest(content string, packageName string) (string, error) {
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

func (pm *NPMPackageManager) parseLockfile(fileName string, content string, packageName string) (string, error) {
	switch fileName {
	case "package-lock.json":
		return pm.parsePackageLock(content, packageName)
	case "yarn.lock":
		return pm.parseYarnLock(content, packageName)
	case "pnpm-lock.yaml":
		return pm.parsePnpmLock(content, packageName)
	case "bun.lock":
		return pm.parseBunLock(content, packageName)
	default:
		return "", fmt.Errorf("unknown lockfile type: %s", fileName)
	}
}

func (pm *NPMPackageManager) parsePackageLock(content string, packageName string) (string, error) {
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

func (pm *NPMPackageManager) parseYarnLock(content string, packageName string) (string, error) {
	versionPattern := `version\s+"([^"]+)"`

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

func (pm *NPMPackageManager) parsePnpmLock(content string, packageName string) (string, error) {
	escapedPackage := regexp.QuoteMeta(packageName)
	pattern := fmt.Sprintf(`/%s/([^/:]+):`, escapedPackage)

	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("failed to compile regex: %w", err)
	}

	matches := re.FindStringSubmatch(content)
	if len(matches) < 2 {
		return "", nil
	}

	return matches[1], nil
}

func (pm *NPMPackageManager) parseBunLock(content string, packageName string) (string, error) {
	escapedPackage := regexp.QuoteMeta(packageName)

	// Try pattern: "package-name@version" or "package-name@npm:version"
	pattern := fmt.Sprintf(`"%s@(?:npm:)?([^"@\s]+)"`, escapedPackage)

	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("failed to compile regex: %w", err)
	}

	matches := re.FindStringSubmatch(content)
	if len(matches) >= 2 {
		return matches[1], nil
	}

	// Alternative pattern: look for version field near package name
	versionPattern := `version:\s*"([^"]+)"`
	reVersion, err := regexp.Compile(versionPattern)
	if err != nil {
		return "", fmt.Errorf("failed to compile version regex: %w", err)
	}

	versionMatches := reVersion.FindStringSubmatch(content)
	if len(versionMatches) >= 2 {
		return versionMatches[1], nil
	}

	return "", nil
}
