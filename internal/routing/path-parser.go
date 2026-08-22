package routing

import (
	"fmt"
	"log/slog"
	"strings"
)

// PathMatcher matches request URLs against one compiled route path.
type PathMatcher struct {
	segments              []Segment
	pathParamNames        []string
	optionalTrailingSlash bool
	matchesEverything     bool
}

// NewPathMatcherFromString compiles a route path into a matcher. It fails on a
// path segment that is neither a literal, a '{name}' parameter nor a wildcard.
func NewPathMatcherFromString(path string) (PathMatcher, error) {
	// A path of nothing but slashes is the root spelled oddly, and a route fragment
	// joined to a path produces it. The matcher has to match what the warning says it became.
	if strings.Trim(path, "/ ") == "" {
		if path != "/" {
			slog.Warn(fmt.Sprintf("The path '%s' was converted to /", path))
		}

		return PathMatcher{}, nil
	}

	if path == wildcard {
		return PathMatcher{matchesEverything: true}, nil
	}

	pathSegments, err := getPathSegments(path)
	if err != nil {
		return PathMatcher{}, err
	}

	pathParamNames := []string{}
	for _, segment := range pathSegments {
		if segment.Kind == ParameterSegment {
			pathParamNames = append(pathParamNames, segment.Text)
		}
	}

	return PathMatcher{
		segments:              pathSegments,
		pathParamNames:        pathParamNames,
		optionalTrailingSlash: strings.HasSuffix(path, "/"),
	}, nil
}

func getPathSegments(path string) ([]Segment, error) {
	pathSegments := []Segment{}
	pathParts := strings.Split(path, "/")

	for _, pathPiece := range pathParts {
		trimmedString := strings.Trim(pathPiece, " ")

		if trimmedString != "" {
			pathSegment, err := newPathSegment(trimmedString)
			if err != nil {
				return pathSegments, err
			}

			pathSegments = append(pathSegments, pathSegment)
		}
	}

	return pathSegments, nil
}

// MatchesURL checks whether given string URL matches the path.
func (path *PathMatcher) MatchesURL(url string) bool {
	_, matches := path.match(url, nil)

	return matches
}

// PathParams extracts the path parameters from given string url according
// to path configuration. Make sure that the path first matches the URL
// before trying to extract the path parameters.
//
// A path that declares no parameters returns nil rather than an empty map, so
// matching one costs neither a second pass over the URL nor an allocation.
func (path *PathMatcher) PathParams(url string) map[string]string {
	if len(path.pathParamNames) == 0 {
		return nil
	}

	values, matches := path.match(url, make([]string, 0, len(path.pathParamNames)))
	if !matches {
		slog.Error(fmt.Sprintf("The URL '%s' does not match the path it was asked for path params on", url))
		return map[string]string{}
	}

	pathParamMap := make(map[string]string, len(values))
	for i, name := range path.pathParamNames {
		pathParamMap[name] = values[i]
	}

	return pathParamMap
}

// Segments returns the pieces the path compiled to, and whether the path is
// nothing but those pieces. A caller that indexes routes by their segments can
// key on the ones it gets; a path holding a wildcard, or one that matches every
// URL, matches by a rule its segments do not carry and reports false.
func (path *PathMatcher) Segments() ([]Segment, bool) {
	if path.matchesEverything {
		return nil, false
	}

	for _, segment := range path.segments {
		if segment.Kind == WildcardSegment {
			return nil, false
		}
	}

	return path.segments, true
}

// OptionalTrailingSlash reports whether the path was registered with a trailing
// slash, and so matches a URL spelled either way. A caller that decided a match
// by the segments alone still has to ask: the slash is a rule about the end of
// the path rather than a piece of it.
func (path *PathMatcher) OptionalTrailingSlash() bool {
	return path.optionalTrailingSlash
}

// match walks the URL across the compiled segments, appending the values it
// captures to values. A nil values skips capture, so the routes a request does
// not match cost nothing to rule out.
func (path *PathMatcher) match(url string, values []string) ([]string, bool) {
	if path.matchesEverything {
		return values, true
	}

	if len(url) == 0 || url[0] != '/' {
		return values, false
	}

	return path.matchSegments(path.segments, url[1:], values)
}

// matchSegments matches segments against a URL positioned at the first
// character of the first segment.
func (path *PathMatcher) matchSegments(segments []Segment, url string, values []string) ([]string, bool) {
	if len(segments) == 0 {
		return values, url == "" || (path.optionalTrailingSlash && url == "/")
	}

	head, rest := segments[0], segments[1:]

	switch head.Kind {
	case LiteralSegment:
		if !strings.HasPrefix(url, head.Text) {
			return values, false
		}

		return path.matchSeparator(rest, url[len(head.Text):], values)

	case ParameterSegment:
		end := strings.IndexByte(url, '/')
		if end < 0 {
			end = len(url)
		}
		if end == 0 {
			return values, false
		}
		if values != nil {
			values = append(values, url[:end])
		}

		return path.matchSeparator(rest, url[end:], values)
	}

	// A wildcard spans separators and takes as little as it can, so the shortest
	// remainder that lets the segments after it match is the one that wins.
	if len(rest) == 0 {
		return values, url != ""
	}

	for consumed := 1; consumed < len(url); consumed++ {
		if captured, matches := path.matchSeparator(rest, url[consumed:], values); matches {
			return captured, true
		}
	}

	return values, false
}

// matchSeparator consumes the slash between two segments.
func (path *PathMatcher) matchSeparator(segments []Segment, url string, values []string) ([]string, bool) {
	if len(segments) == 0 {
		return path.matchSegments(segments, url, values)
	}

	if len(url) == 0 || url[0] != '/' {
		return values, false
	}

	return path.matchSegments(segments, url[1:], values)
}
