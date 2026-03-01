package npm

import (
	"context"
	"testing"

	"github.com/bziks/gitlab-package-finder/internal/domain/entity"
)

func TestGetFilePatterns(t *testing.T) {
	pm := NewPackageManager()
	patterns := pm.GetFilePatterns()

	expectedManifests := []string{"package.json"}
	expectedLockfiles := []string{"package-lock.json", "yarn.lock", "pnpm-lock.yaml", "bun.lock"}

	if len(patterns.Manifests) != len(expectedManifests) {
		t.Fatalf("expected %d manifests, got %d", len(expectedManifests), len(patterns.Manifests))
	}
	for i, m := range patterns.Manifests {
		if m != expectedManifests[i] {
			t.Errorf("manifest[%d]: expected %q, got %q", i, expectedManifests[i], m)
		}
	}

	if len(patterns.Lockfiles) != len(expectedLockfiles) {
		t.Fatalf("expected %d lockfiles, got %d", len(expectedLockfiles), len(patterns.Lockfiles))
	}
	for i, l := range patterns.Lockfiles {
		if l != expectedLockfiles[i] {
			t.Errorf("lockfile[%d]: expected %q, got %q", i, expectedLockfiles[i], l)
		}
	}
}

func TestParseDependency(t *testing.T) {
	pm := NewPackageManager()
	ctx := context.Background()

	tests := []struct {
		name           string
		files          []entity.DependencyFile
		packageName    string
		wantFound      bool
		wantConstraint string
		wantResolved   string
		wantSourceFile string
	}{
		{
			name: "found in manifest only",
			files: []entity.DependencyFile{
				{
					FileName: "package.json",
					FileType: entity.FileTypeManifest,
					Content:  `"lodash": "^4.17.21"`,
				},
			},
			packageName:    "lodash",
			wantFound:      true,
			wantConstraint: "^4.17.21",
			wantSourceFile: "package.json",
		},
		{
			name: "found in package-lock.json only",
			files: []entity.DependencyFile{
				{
					FileName: "package-lock.json",
					FileType: entity.FileTypeLockfile,
					Content:  `"version": "4.17.21"`,
				},
			},
			packageName:    "lodash",
			wantFound:      true,
			wantResolved:   "4.17.21",
			wantSourceFile: "package-lock.json",
		},
		{
			name: "found in yarn.lock only",
			files: []entity.DependencyFile{
				{
					FileName: "yarn.lock",
					FileType: entity.FileTypeLockfile,
					Content:  "lodash@^4.17.21:\n  version \"4.17.21\"",
				},
			},
			packageName:    "lodash",
			wantFound:      true,
			wantResolved:   "4.17.21",
			wantSourceFile: "yarn.lock",
		},
		{
			name: "found in pnpm-lock.yaml only",
			files: []entity.DependencyFile{
				{
					FileName: "pnpm-lock.yaml",
					FileType: entity.FileTypeLockfile,
					Content:  "/lodash/4.17.21:",
				},
			},
			packageName:    "lodash",
			wantFound:      true,
			wantResolved:   "4.17.21",
			wantSourceFile: "pnpm-lock.yaml",
		},
		{
			name: "found in both prefers lockfile source",
			files: []entity.DependencyFile{
				{
					FileName: "package.json",
					FileType: entity.FileTypeManifest,
					Content:  `"lodash": "^4.17.21"`,
				},
				{
					FileName: "package-lock.json",
					FileType: entity.FileTypeLockfile,
					Content:  `"version": "4.17.21"`,
				},
			},
			packageName:    "lodash",
			wantFound:      true,
			wantConstraint: "^4.17.21",
			wantResolved:   "4.17.21",
			wantSourceFile: "package-lock.json",
		},
		{
			name: "package-lock has higher priority than yarn.lock",
			files: []entity.DependencyFile{
				{
					FileName: "yarn.lock",
					FileType: entity.FileTypeLockfile,
					Content:  "lodash@^4.17.21:\n  version \"4.17.20\"",
				},
				{
					FileName: "package-lock.json",
					FileType: entity.FileTypeLockfile,
					Content:  `"version": "4.17.21"`,
				},
			},
			packageName:    "lodash",
			wantFound:      true,
			wantResolved:   "4.17.21",
			wantSourceFile: "package-lock.json",
		},
		{
			name: "not found",
			files: []entity.DependencyFile{
				{
					FileName: "package.json",
					FileType: entity.FileTypeManifest,
					Content:  `"express": "^4.18.0"`,
				},
			},
			packageName: "lodash",
			wantFound:   false,
		},
		{
			name:        "empty files",
			files:       []entity.DependencyFile{},
			packageName: "lodash",
			wantFound:   false,
		},
		{
			name: "malformed content",
			files: []entity.DependencyFile{
				{
					FileName: "package.json",
					FileType: entity.FileTypeManifest,
					Content:  "not valid json",
				},
			},
			packageName: "lodash",
			wantFound:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := pm.ParseDependency(ctx, tt.files, tt.packageName)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Found != tt.wantFound {
				t.Errorf("Found: got %v, want %v", result.Found, tt.wantFound)
			}
			if result.VersionConstraint != tt.wantConstraint {
				t.Errorf("VersionConstraint: got %q, want %q", result.VersionConstraint, tt.wantConstraint)
			}
			if result.ResolvedVersion != tt.wantResolved {
				t.Errorf("ResolvedVersion: got %q, want %q", result.ResolvedVersion, tt.wantResolved)
			}
			if result.SourceFile != tt.wantSourceFile {
				t.Errorf("SourceFile: got %q, want %q", result.SourceFile, tt.wantSourceFile)
			}
		})
	}
}
