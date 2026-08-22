package routing

import (
	"fmt"
	"strings"
)

// SegmentKind is what one piece of a route path matches.
type SegmentKind uint8

const (
	// LiteralSegment matches exactly its text.
	LiteralSegment SegmentKind = iota
	// ParameterSegment matches any whole piece and captures it.
	ParameterSegment
	// WildcardSegment matches one or more characters across separators.
	WildcardSegment
)

// Segment is one slash-separated piece of a compiled route path.
type Segment struct {
	// Kind says what the segment matches.
	Kind SegmentKind
	// Text is the literal a literal segment matches, or the name a parameter
	// segment captures under. A wildcard segment has neither.
	Text string
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
func newPathSegment(pathPiece string) (Segment, error) {
	if pathPiece == wildcard {
		return Segment{Kind: WildcardSegment}, nil
	}

	if !strings.ContainsAny(pathPiece, delimiters) {
		return Segment{Kind: LiteralSegment, Text: pathPiece}, nil
	}

	name, hasStart := strings.CutPrefix(pathPiece, delimiterStart)
	name, hasEnd := strings.CutSuffix(name, delimiterEnd)

	if !hasStart || !hasEnd || name == "" || strings.ContainsAny(name, delimiters) {
		return Segment{}, fmt.Errorf(
			"path segment '%s' is neither a literal nor a '{name}' parameter", pathPiece,
		)
	}

	return Segment{Kind: ParameterSegment, Text: name}, nil
}
