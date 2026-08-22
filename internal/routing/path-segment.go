package routing

import (
	"fmt"
	"strings"
)

type segmentKind uint8

const (
	literalSegment segmentKind = iota
	parameterSegment
	wildcardSegment
)

// pathSegment is one slash-separated piece of a compiled route path.
type pathSegment struct {
	kind segmentKind
	// text is the literal a literal segment matches, or the name a parameter
	// segment captures under. A wildcard segment has neither.
	text string
}

const (
	delimiterStart = "{"
	delimiterEnd   = "}"
	delimiters     = delimiterStart + delimiterEnd
	wildcard       = "*"
)

// newPathSegment reads one slash-separated piece of a route path. A piece is a
// wildcard, a literal, or a single '{name}' parameter that spans the whole
// piece; anything else is a path the matcher has no meaning for.
func newPathSegment(pathPiece string) (pathSegment, error) {
	if pathPiece == wildcard {
		return pathSegment{kind: wildcardSegment}, nil
	}

	if !strings.ContainsAny(pathPiece, delimiters) {
		return pathSegment{kind: literalSegment, text: pathPiece}, nil
	}

	name, hasStart := strings.CutPrefix(pathPiece, delimiterStart)
	name, hasEnd := strings.CutSuffix(name, delimiterEnd)

	if !hasStart || !hasEnd || name == "" || strings.ContainsAny(name, delimiters) {
		return pathSegment{}, fmt.Errorf(
			"path segment '%s' is neither a literal nor a '{name}' parameter", pathPiece,
		)
	}

	return pathSegment{kind: parameterSegment, text: name}, nil
}
