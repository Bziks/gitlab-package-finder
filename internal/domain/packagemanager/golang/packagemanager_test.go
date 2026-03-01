package golang

import (
	"context"
	"testing"

	"github.com/bziks/gitlab-package-finder/internal/domain/entity"
)

func TestGetFilePatterns(t *testing.T) {
	pm := NewPackageManager()
	patterns := pm.GetFilePatterns()

	expectedManifests := []string{"go.mod"}
	expectedLockfiles := []string{"go.sum"}

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
			name: "found in manifest only - single line require",
			files: []entity.DependencyFile{
				{
					FileName: "go.mod",
					FileType: entity.FileTypeManifest,
					Content:  "require github.com/stretchr/testify v1.8.4",
				},
			},
			packageName:    "github.com/stretchr/testify",
			wantFound:      true,
			wantConstraint: "v1.8.4",
			wantSourceFile: "go.mod",
		},
		{
			name: "found in manifest only - require block",
			files: []entity.DependencyFile{
				{
					FileName: "go.mod",
					FileType: entity.FileTypeManifest,
					Content:  "require (\n\tgithub.com/stretchr/testify v1.8.4\n)",
				},
			},
			packageName:    "github.com/stretchr/testify",
			wantFound:      true,
			wantConstraint: "v1.8.4",
			wantSourceFile: "go.mod",
		},
		{
			name: "found in lockfile only",
			files: []entity.DependencyFile{
				{
					FileName: "go.sum",
					FileType: entity.FileTypeLockfile,
					Content:  "github.com/stretchr/testify v1.8.4 h1:abc123=\ngithub.com/stretchr/testify v1.8.4/go.mod h1:def456=",
				},
			},
			packageName:    "github.com/stretchr/testify",
			wantFound:      true,
			wantResolved:   "v1.8.4",
			wantSourceFile: "go.sum",
		},
		{
			name: "found in both prefers lockfile source",
			files: []entity.DependencyFile{
				{
					FileName: "go.mod",
					FileType: entity.FileTypeManifest,
					Content:  "\tgithub.com/stretchr/testify v1.8.4",
				},
				{
					FileName: "go.sum",
					FileType: entity.FileTypeLockfile,
					Content:  "github.com/stretchr/testify v1.8.4 h1:abc123=",
				},
			},
			packageName:    "github.com/stretchr/testify",
			wantFound:      true,
			wantConstraint: "v1.8.4",
			wantResolved:   "v1.8.4",
			wantSourceFile: "go.sum",
		},
		{
			name: "not found",
			files: []entity.DependencyFile{
				{
					FileName: "go.mod",
					FileType: entity.FileTypeManifest,
					Content:  "\tgithub.com/other/package v1.0.0",
				},
			},
			packageName: "github.com/stretchr/testify",
			wantFound:   false,
		},
		{
			name:        "empty files",
			files:       []entity.DependencyFile{},
			packageName: "github.com/stretchr/testify",
			wantFound:   false,
		},
		{
			name: "malformed content",
			files: []entity.DependencyFile{
				{
					FileName: "go.mod",
					FileType: entity.FileTypeManifest,
					Content:  "this is not a valid go.mod file",
				},
			},
			packageName: "github.com/stretchr/testify",
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
