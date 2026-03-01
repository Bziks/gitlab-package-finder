package composer

import (
	"context"
	"testing"

	"github.com/bziks/gitlab-package-finder/internal/domain/entity"
)

func TestGetFilePatterns(t *testing.T) {
	pm := NewPackageManager()
	patterns := pm.GetFilePatterns()

	expectedManifests := []string{"composer.json"}
	expectedLockfiles := []string{"composer.lock"}

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
					FileName: "composer.json",
					FileType: entity.FileTypeManifest,
					Content:  `"monolog/monolog": "^2.0"`,
				},
			},
			packageName:    "monolog/monolog",
			wantFound:      true,
			wantConstraint: "^2.0",
			wantSourceFile: "composer.json",
		},
		{
			name: "found in lockfile only",
			files: []entity.DependencyFile{
				{
					FileName: "composer.lock",
					FileType: entity.FileTypeLockfile,
					Content:  "\"name\": \"monolog/monolog\",\n\"version\": \"2.9.1\"",
				},
			},
			packageName:    "monolog/monolog",
			wantFound:      true,
			wantResolved:   "2.9.1",
			wantSourceFile: "composer.lock",
		},
		{
			name: "found in both prefers lockfile source",
			files: []entity.DependencyFile{
				{
					FileName: "composer.json",
					FileType: entity.FileTypeManifest,
					Content:  `"monolog/monolog": "^2.0"`,
				},
				{
					FileName: "composer.lock",
					FileType: entity.FileTypeLockfile,
					Content:  "\"name\": \"monolog/monolog\",\n\"version\": \"2.9.1\"",
				},
			},
			packageName:    "monolog/monolog",
			wantFound:      true,
			wantConstraint: "^2.0",
			wantResolved:   "2.9.1",
			wantSourceFile: "composer.lock",
		},
		{
			name: "not found",
			files: []entity.DependencyFile{
				{
					FileName: "composer.json",
					FileType: entity.FileTypeManifest,
					Content:  `"other/package": "^1.0"`,
				},
			},
			packageName: "monolog/monolog",
			wantFound:   false,
		},
		{
			name:        "empty files",
			files:       []entity.DependencyFile{},
			packageName: "monolog/monolog",
			wantFound:   false,
		},
		{
			name: "malformed content",
			files: []entity.DependencyFile{
				{
					FileName: "composer.json",
					FileType: entity.FileTypeManifest,
					Content:  `not valid json at all`,
				},
			},
			packageName: "monolog/monolog",
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
