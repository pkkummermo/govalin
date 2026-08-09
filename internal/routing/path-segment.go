package routing

import (
	"fmt"
	"regexp"
	"strings"
)

type pathSegment struct {
	PathPiece    string
	GroupedRegex string
	PathNames    []string
}

var (
	delimiterStart = "{"
	delimiterEnd   = "}"
	wildcard       = "*"

	wildcardPathSegment = pathSegment{
		PathPiece:    "*",
		PathNames:    []string{},
		GroupedRegex: ".+?",
	}
	rootPathSegment = pathSegment{
		PathPiece:    "/",
		PathNames:    []string{},
		GroupedRegex: "/",
	}
)

func newPathSegment(pathPiece string) (pathSegment, error) {
	delimiterStartCount := strings.Count(pathPiece, delimiterStart)
	delimiterEndCount := strings.Count(pathPiece, delimiterEnd)
	totalDelimiters := delimiterStartCount + delimiterEndCount

	// Error in number of delimiters
	if delimiterStartCount != delimiterEndCount {
		return pathSegment{}, fmt.Errorf("number of '%d' and '%d' is not the same", delimiterStartCount, delimiterEndCount)
	}

	// Wildcard
	if pathPiece == wildcard {
		return wildcardPathSegment, nil
	}

	// No matcher
	if totalDelimiters == 0 {
		return createNormalPathSegment(pathPiece), nil
	}

	// Simple matcher
	if totalDelimiters == 2 && pathPiece[0:1] == delimiterStart && pathPiece[len(pathPiece)-1:] == delimiterEnd {
		return createParameterPathSegment(pathPiece), nil
	}

	return pathSegment{}, nil
}

func createNormalPathSegment(pathPiece string) pathSegment {
	return pathSegment{
		PathPiece:    pathPiece,
		PathNames:    []string{},
		GroupedRegex: regexp.QuoteMeta(pathPiece),
	}
}

func createParameterPathSegment(pathPiece string) pathSegment {
	return pathSegment{
		PathPiece:    pathPiece,
		PathNames:    []string{strings.Trim(strings.Trim(pathPiece, delimiterStart), delimiterEnd)},
		GroupedRegex: "([^/]+?)",
	}
}
