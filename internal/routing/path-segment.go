package routing

import (
	"fmt"
	"regexp"
	"strings"
)

type pathSegment struct {
	GroupedRegex string
	PathNames    []string
}

const (
	delimiterStart = "{"
	delimiterEnd   = "}"
	delimiters     = delimiterStart + delimiterEnd
	wildcard       = "*"
)

var wildcardPathSegment = pathSegment{
	PathNames:    []string{},
	GroupedRegex: ".+?",
}

// newPathSegment reads one slash-separated piece of a route path. A piece is a
// wildcard, a literal, or a single '{name}' parameter that spans the whole
// piece; anything else is a path the matcher has no meaning for.
func newPathSegment(pathPiece string) (pathSegment, error) {
	if pathPiece == wildcard {
		return wildcardPathSegment, nil
	}

	if !strings.ContainsAny(pathPiece, delimiters) {
		return createNormalPathSegment(pathPiece), nil
	}

	name, hasStart := strings.CutPrefix(pathPiece, delimiterStart)
	name, hasEnd := strings.CutSuffix(name, delimiterEnd)

	if !hasStart || !hasEnd || name == "" || strings.ContainsAny(name, delimiters) {
		return pathSegment{}, fmt.Errorf(
			"path segment '%s' is neither a literal nor a '{name}' parameter", pathPiece,
		)
	}

	return createParameterPathSegment(name), nil
}

func createNormalPathSegment(pathPiece string) pathSegment {
	return pathSegment{
		PathNames:    []string{},
		GroupedRegex: regexp.QuoteMeta(pathPiece),
	}
}

func createParameterPathSegment(name string) pathSegment {
	return pathSegment{
		PathNames:    []string{name},
		GroupedRegex: "([^/]+?)",
	}
}
