package routing_test

import (
	"testing"

	"github.com/pkkummermo/govalin/internal/routing"
	"github.com/stretchr/testify/assert"
)

func TestSimplePathMatching(t *testing.T) {
	pathMatcher, err := routing.NewPathMatcherFromString("/govalin")

	assert.Nil(t, err)
	assert.Equal(t, true, pathMatcher.MatchesURL("/govalin"), "Should match on exact match")
	assert.Equal(t, false, pathMatcher.MatchesURL("/go"), "Should not match on partial match")
	assert.Equal(t, false, pathMatcher.MatchesURL("/govalintest"), "Should not match on partial match")
	assert.Equal(t, false, pathMatcher.MatchesURL("/somethingelse"), "Should not match on partial match")
}

func TestRootMatching(t *testing.T) {
	pathMatcher, err := routing.NewPathMatcherFromString("/")

	assert.Nil(t, err)
	assert.Equal(t, true, pathMatcher.MatchesURL("/"), "Should match on exact match")
	assert.Equal(t, false, pathMatcher.MatchesURL("/govalin"), "Should not match on partial match")
	assert.Equal(t, false, pathMatcher.MatchesURL("/govalin/"), "Should not match on trailing slash")
}

// TestAllSlashPathIsTheRootPath pins the normalization the parser has always
// warned about but never performed: a path of nothing but slashes is the root
// path, so the root is what it matches. It used to compile to '^///?$', matching
// every spelling except the one it said the path had become. A route fragment
// and a path that both end in a slash produce exactly that, so it is reachable
// from `app.Route("/", …)` around an `app.Get("/", …)`.
func TestAllSlashPathIsTheRootPath(t *testing.T) {
	for _, path := range []string{"//", "///", ""} {
		pathMatcher, err := routing.NewPathMatcherFromString(path)

		assert.Nil(t, err)
		assert.Equal(t, true, pathMatcher.MatchesURL("/"), "'%s' should match the root it was converted to", path)
		assert.Equal(t, false, pathMatcher.MatchesURL("//"), "'%s' should not match a doubled slash", path)
	}
}

func TestSimpleWildcardMatch(t *testing.T) {
	pathMatcher, err := routing.NewPathMatcherFromString("*")
	assert.Nil(t, err)
	assert.Equal(t, true, pathMatcher.MatchesURL("/"), "Should match on root request")
	assert.Equal(t, true, pathMatcher.MatchesURL("/test"), "Should match on more specific requests")
}

func TestEndingRouteWildcardMatch(t *testing.T) {
	pathMatcher, err := routing.NewPathMatcherFromString("/govalin/*")
	assert.Nil(t, err)
	assert.Equal(t, false, pathMatcher.MatchesURL("/govalin/"), "Should not match on root request")
	assert.Equal(t, true, pathMatcher.MatchesURL("/govalin/test"), "Should match on more specific requests")
	assert.Equal(t, true, pathMatcher.MatchesURL("/govalin/test/with/sub/path"), "Should match on nested requests")
	assert.Equal(t, false, pathMatcher.MatchesURL("/govalintest"), "Should not match on subpartial match")
}

// TestOptionalTrailingSlashMatch pins both spellings of the path as the route's.
// Which of them is canonical is the handler's to decide, not the matcher's.
func TestOptionalTrailingSlashMatch(t *testing.T) {
	pathMatcher, err := routing.NewPathMatcherFromString("/govalin/")
	assert.Nil(t, err)
	assert.Equal(t, true, pathMatcher.MatchesURL("/govalin"), "Should match without the optional trailing slash")
	assert.Equal(t, true, pathMatcher.MatchesURL("/govalin/"), "Should match on exact")
	assert.Equal(t, false, pathMatcher.MatchesURL("/govalin//"), "Should not match a doubled slash")
	assert.Equal(t, false, pathMatcher.MatchesURL("/govalin/sub"), "Should not match below the path")
}

func TestNestedWildcardMatch(t *testing.T) {
	pathMatcher, err := routing.NewPathMatcherFromString("foo/*/bar")
	assert.Nil(t, err)
	assert.Equal(t, true, pathMatcher.MatchesURL("/foo/baz/bar"), "Should match wildcard match")
	assert.Equal(t, false, pathMatcher.MatchesURL("/baz/baz/foo"), "Should not match on mismatched wildcard")
	assert.Equal(t, false, pathMatcher.MatchesURL("/foo/baz"), "Should not match on mismatched wildcard")
}

// A segment mixing literal text and a parameter used to silently compile to a
// pattern matching nothing its author meant; now it fails at registration.
func TestMalformedParameterSegmentIsRejected(t *testing.T) {
	for _, path := range []string{"/a/{b}c", "/a/x{b}", "/a/{b}-{c}", "/a/{}", "/a/{{b}}", "/a/{b", "/a/b}"} {
		_, err := routing.NewPathMatcherFromString(path)

		assert.Error(t, err, "'%s' is not a path the matcher can express", path)
	}
}

// TestParameterSegmentSpansTheWholePiece keeps the shapes either side of the
// rejection above matching what they always did.
func TestParameterSegmentSpansTheWholePiece(t *testing.T) {
	pathMatcher, err := routing.NewPathMatcherFromString("/a/{b}")

	assert.Nil(t, err)
	assert.Equal(t, true, pathMatcher.MatchesURL("/a/foo"), "Should match a value in the parameter position")
	assert.Equal(t, map[string]string{"b": "foo"}, pathMatcher.PathParams("/a/foo"), "Should capture the value")
	assert.Equal(t, false, pathMatcher.MatchesURL("/a/"), "Should not match an empty parameter")
	assert.Equal(t, false, pathMatcher.MatchesURL("/a/foo/bar"), "Should not span a separator")
}
