package govalin

import (
	"math/rand/v2"
	"net/http"
	"strings"
	"testing"
)

// linearMatch is the scan the index replaced: walk the route table in
// registration order and take the first route that matches and answers the
// method.
func linearMatch(handlers []*pathHandler, url string, method string) *pathHandler {
	for _, handler := range handlers {
		if handler.GetHandlerByMethod(method) != nil && handler.PathMatcher.MatchesURL(url) {
			return handler
		}
	}

	return nil
}

// TestRouteIndexAgreesWithAScanOfTheTable is the property the index has to
// hold: it returns the route the table walk would have, whatever the table.
// Random tables reach the overlaps a written-out case list does not — a literal
// registered behind a parameter, two routes on the same node differing only in
// a trailing slash, a wildcard cutting in ahead of a tree route.
func TestRouteIndexAgreesWithAScanOfTheTable(t *testing.T) {
	pieces := []string{"a", "b", "ab", "{id}", "{name}", "*", "c"}
	urlPieces := []string{"a", "b", "ab", "c", "x", "yy", ""}
	methods := []string{http.MethodGet, http.MethodPost, http.MethodDelete}

	random := rand.New(rand.NewPCG(1, 2))

	for range 20000 {
		app := New(func(config *Config) {
			config.EnableAccessLog(false)
			config.EnableStartupLog(false)
		})

		registered := map[string]bool{}
		for range random.IntN(8) + 1 {
			path := ""
			for range random.IntN(3) + 1 {
				path += "/" + pieces[random.IntN(len(pieces))]
			}
			if random.IntN(4) == 0 {
				path += "/"
			}

			method := methods[random.IntN(len(methods))]
			if registered[method+" "+path] {
				continue
			}
			registered[method+" "+path] = true

			app.addMethod(method, path, func(_ *Call) {})
		}

		url := ""
		for range random.IntN(4) + 1 {
			url += "/" + urlPieces[random.IntN(len(urlPieces))]
		}
		if random.IntN(4) == 0 {
			url += "/"
		}

		for _, method := range methods {
			want := linearMatch(app.pathHandlers, url, method)
			got := app.routes.match(url, method)

			if want != got {
				t.Fatalf(
					"%s %s over table [%s]: table walk picked %v, index picked %v",
					method, url, describe(app.pathHandlers), fragmentOf(want), fragmentOf(got),
				)
			}
		}
	}
}

func describe(handlers []*pathHandler) string {
	fragments := make([]string, 0, len(handlers))
	for _, handler := range handlers {
		fragments = append(fragments, handler.PathFragment)
	}

	return strings.Join(fragments, " ")
}

func fragmentOf(handler *pathHandler) string {
	if handler == nil {
		return "<none>"
	}

	return handler.PathFragment
}
