package routing

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"
)

type PathMatcher struct {
	pathParamNames []string
	regexp         regexp.Regexp
}

func NewPathMatcherFromString(path string) (PathMatcher, error) {
	// A path of nothing but slashes is the root spelled oddly, and a route fragment
	// joined to a path produces it. The matcher has to match what the warning says it became.
	if strings.Trim(path, "/ ") == "" {
		if path != "/" {
			slog.Warn(fmt.Sprintf("The path '%s' was converted to /", path))
		}

		return PathMatcher{
			pathParamNames: []string{},
			regexp:         *regexp.MustCompile("^/$"),
		}, nil
	}

	if path == "*" {
		return PathMatcher{
			pathParamNames: []string{},
			regexp:         *regexp.MustCompile(".*?"),
		}, nil
	}

	pathSegments, err := getPathSegments(path)
	if err != nil {
		return PathMatcher{}, err
	}

	pathParamNames := []string{}
	for _, ps := range pathSegments {
		pathParamNames = append(pathParamNames, ps.PathNames...)
	}

	groupRegexpParts := []string{}

	for _, ps := range pathSegments {
		groupRegexpParts = append(groupRegexpParts, ps.GroupedRegex)
	}

	// Appended after the join, not as a segment: a segment carries a mandatory slash in
	// front of the optional one, leaving the pattern matching only the doubled form.
	optionalTrailingSlash := ""
	if strings.HasSuffix(path, "/") {
		optionalTrailingSlash = "/?"
	}

	fullGroupedRegexpString := "^/" + strings.Join(groupRegexpParts, "/") + optionalTrailingSlash + "$"

	return PathMatcher{
		pathParamNames: pathParamNames,
		regexp:         *regexp.MustCompile(fullGroupedRegexpString),
	}, nil
}

func getPathSegments(path string) ([]pathSegment, error) {
	pathSegments := []pathSegment{}
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
	return path.regexp.MatchString(url)
}

// PathParams extracts the path parameters from given string url according
// to path configuration. Make sure that the path first matches the URL
// before trying to extract the path parameters.
func (path *PathMatcher) PathParams(url string) map[string]string {
	pathparamMap := map[string]string{}
	pathParams := path.regexp.FindStringSubmatch(url)

	if len(pathParams) != len(path.pathParamNames)+1 {
		slog.Error("The number of path params is not the same as configured path names")
		return pathparamMap
	}

	for i, v := range path.pathParamNames {
		pathparamMap[v] = pathParams[i+1]
	}

	return pathparamMap
}
