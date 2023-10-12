package routing

import (
	"regexp"
	"strings"

	"log/slog"
)

type PathMatcher struct {
	path           string
	segments       []pathSegment
	pathParamNames []string
	regexp         regexp.Regexp
	matchRegexp    regexp.Regexp
}

func NewPathMatcherFromString(path string) (PathMatcher, error) {
	var pathSegments = []pathSegment{}

	for _, pathPiece := range strings.Split(path, "/") {
		trimmedString := strings.Trim(pathPiece, " ")

		if trimmedString != "" {
			pathSegment, err := newPathSegment(trimmedString)
			if err != nil {
				return PathMatcher{}, err
			}

			pathSegments = append(pathSegments, pathSegment)
		}
	}

	var pathParamNames = []string{}
	// Extract path param names
	for _, ps := range pathSegments {
		pathParamNames = append(pathParamNames, ps.PathNames...)
	}

	groupRegexpParts := []string{}
	regexpParts := []string{}

	for _, ps := range pathSegments {
		regexpParts = append(regexpParts, ps.Regex)
		groupRegexpParts = append(groupRegexpParts, ps.GroupedRegex)
	}

	fullGroupedRegexpString := strings.Join(groupRegexpParts, "/") + "$"
	fullRegexpString := strings.Join(regexpParts, "/") + "$"

	return PathMatcher{
		path:           path,
		pathParamNames: pathParamNames,
		segments:       pathSegments,
		regexp:         *regexp.MustCompile(fullGroupedRegexpString),
		matchRegexp:    *regexp.MustCompile(fullRegexpString),
	}, nil
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
