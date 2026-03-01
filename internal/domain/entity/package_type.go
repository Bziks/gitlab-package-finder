package entity

const (
	PackageTypeComposer = "composer"
	PackageTypeGo       = "go"
	PackageTypeNpm      = "npm"
)

type PackageType struct {
	ID    int
	Name  string
	Label string
}
