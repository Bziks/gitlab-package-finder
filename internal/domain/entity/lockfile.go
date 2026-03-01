package entity

type FileType string

const (
	FileTypeManifest FileType = "manifest"
	FileTypeLockfile FileType = "lockfile"
)

type DependencyFile struct {
	FileName string
	FilePath string
	Content  string
	FileType FileType
}

type ParseResult struct {
	Found             bool
	SourceFile        string
	VersionConstraint string
	ResolvedVersion   string
}

type FilePatterns struct {
	Manifests []string
	Lockfiles []string
}
