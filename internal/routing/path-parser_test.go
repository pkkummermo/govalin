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
