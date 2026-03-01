package search

import (
	"strings"

	"github.com/Masterminds/semver/v3"
)

// MatchesVersion checks whether resolvedVersion satisfies the given semver constraint.
// An empty constraint matches everything. An empty or unparseable resolvedVersion never matches.
func MatchesVersion(constraint string, resolvedVersion string) bool {
	if constraint == "" {
		return true
	}
	if resolvedVersion == "" {
		return false
	}

	resolvedVersion = strings.TrimPrefix(resolvedVersion, "v")

	v, err := semver.NewVersion(resolvedVersion)
	if err != nil {
		return false
	}

	c, err := semver.NewConstraint(constraint)
	if err != nil {
		return false
	}

	return c.Check(v)
}
