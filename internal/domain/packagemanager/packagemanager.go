package packagemanager

import (
	"context"
	"errors"
	"log/slog"

	"github.com/bziks/gitlab-package-finder/internal/domain/entity"
)

var ErrPackageManagerIsNotRegistered = errors.New("package manager is not registered")

type PackageManager interface {
	GetFilePatterns() entity.FilePatterns
	GetLanguages() []string

	ParseDependency(ctx context.Context, files []entity.DependencyFile, packageName string) (*entity.ParseResult, error)
}

type Factory struct {
	managers map[string]PackageManager
}

func NewFactory() *Factory {
	return &Factory{
		managers: make(map[string]PackageManager, 0),
	}
}

func (f *Factory) Register(name string, manager PackageManager) {
	slog.Info("register package manager", "name", name)

	f.managers[name] = manager
}

func (f *Factory) Get(name string) (PackageManager, error) {
	slog.Info("get package manager", "name", name)

	if _, ok := f.managers[name]; !ok {
		return nil, ErrPackageManagerIsNotRegistered
	}

	return f.managers[name], nil
}

func (f *Factory) LanguageMap() map[string]string {
	result := make(map[string]string)
	for name, manager := range f.managers {
		for _, lang := range manager.GetLanguages() {
			result[lang] = name
		}
	}
	return result
}
