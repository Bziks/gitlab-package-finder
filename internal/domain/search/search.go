package search

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/bziks/gitlab-package-finder/internal/domain/entity"
)

func GenerateSearchID(packageType string, packageName string, packageVersion string) string {
	h := md5.New()
	fmt.Fprintf(h, "%s:%s:%s", packageType, packageName, packageVersion)
	searchID := hex.EncodeToString(h.Sum(nil))

	return searchID
}

func ParseQuery(query string) (entity.SearchPackage, error) {
	if query == "" {
		return entity.SearchPackage{}, entity.ErrInvalidQuery
	}

	parts := strings.Split(query, ":")

	if len(parts) > 2 {
		return entity.SearchPackage{}, entity.ErrInvalidQuery
	}

	name := strings.TrimSpace(parts[0])
	if name == "" {
		return entity.SearchPackage{}, entity.ErrInvalidQuery
	}

	sp := entity.SearchPackage{
		Name: name,
	}

	if len(parts) > 1 {
		sp.Version = parts[1]
	}

	return sp, nil
}
