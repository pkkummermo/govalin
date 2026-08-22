package govalin

import (
	"fmt"
	"net/http"
	"testing"
)

var routeTableSizes = []int{1, 20, 100, 200}

// benchRoute is one entry in a generated table: the pattern it is registered
// under and a concrete URL that matches it.
type benchRoute struct {
	pattern string
	sample  string
}

// restRoutes generates a parameter-heavy table shaped like a real REST API — a
// collection, an item, a sub-collection, a sub-item and a nested action,
// repeated across resources until it holds size routes.
func restRoutes(size int) []benchRoute {
	shapes := []benchRoute{
		{"/v1/%s", "/v1/%s"},
		{"/v1/%s/{id}", "/v1/%s/4711"},
		{"/v1/%s/{id}/comments", "/v1/%s/4711/comments"},
		{"/v1/%s/{id}/comments/{commentId}", "/v1/%s/4711/comments/8822"},
		{"/v1/%s/{id}/collaborators/{login}/permission", "/v1/%s/4711/collaborators/octocat/permission"},
	}

	routes := make([]benchRoute, 0, size)
	for resourceIndex := 0; len(routes) < size; resourceIndex++ {
		resource := fmt.Sprintf("resource%d", resourceIndex)

		for _, shape := range shapes {
			if len(routes) == size {
				break
			}

			routes = append(routes, benchRoute{
				pattern: fmt.Sprintf(shape.pattern, resource),
				sample:  fmt.Sprintf(shape.sample, resource),
			})
		}
	}

	return routes
}

// BenchmarkRouteTableScaling serves one request against tables of growing size,
// at both ends of the table.
//
// Route matching walks the table in registration order, so what a match costs
// depends on how far down it sits: the first and last route bracket the range.
// Allocations, which are what the budget in perf_test.go gates, must not depend
// on table size at all.
func BenchmarkRouteTableScaling(b *testing.B) {
	for _, size := range routeTableSizes {
		routes := restRoutes(size)

		positions := []struct {
			name  string
			route benchRoute
		}{
			{"first", routes[0]},
			{"last", routes[len(routes)-1]},
		}

		for _, position := range positions {
			b.Run(fmt.Sprintf("routes=%d/%s", size, position.name), func(b *testing.B) {
				app := benchApp(func(app *App) {
					for _, route := range routes {
						app.Get(route.pattern, func(call *Call) { call.Text("Hello world") })
					}
				})

				benchServe(b, app, http.MethodGet, position.route.sample, http.StatusOK, 11)
			})
		}
	}
}
