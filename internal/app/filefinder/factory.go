package filefinder

import (
	"errors"
	"log/slog"

	"github.com/bziks/gitlab-package-finder/internal/ports"
)

var ErrFileFinderIsNotRegistered = errors.New("file finder is not registered")

type Factory struct {
	fileFinders map[string]ports.FileFinder
}

func NewFactory() *Factory {
	return &Factory{
		fileFinders: make(map[string]ports.FileFinder, 0),
	}
}

func (f *Factory) Register(name string, fileFinder ports.FileFinder) {
	slog.Info("register file finder", "name", name)

	f.fileFinders[name] = fileFinder
}

func (f *Factory) Get(name string) (ports.FileFinder, error) {
	slog.Info("get file finder", "name", name)

	if _, ok := f.fileFinders[name]; !ok {
		return nil, ErrFileFinderIsNotRegistered
	}

	return f.fileFinders[name], nil
}
