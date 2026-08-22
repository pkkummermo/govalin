package govalin

import (
	"math"
	"strings"

	"github.com/pkkummermo/govalin/internal/routing"
)

// routeIndex answers which route serves a URL without asking every route: a
// tree over path segments, walked by the URL's pieces. Registration order
// still decides which of several matching routes wins, and wildcard routes,
// which the tree cannot key on, stay in an ordered list — see ADR 0015.
type routeIndex struct {
	root      *routeNode
	wildcards []*pathHandler
}

type routeNode struct {
	literals map[string]*routeNode
	param    *routeNode
	routes   []*pathHandler
	minOrder int
}

func (index *routeIndex) add(handler *pathHandler) {
	segments, indexable := handler.PathMatcher.Segments()
	if !indexable {
		index.wildcards = append(index.wildcards, handler)
		return
	}

	if index.root == nil {
		index.root = &routeNode{minOrder: handler.order}
	}

	node := index.root
	for _, segment := range segments {
		node = node.child(segment, handler.order)
	}

	node.routes = append(node.routes, handler)
}

// child returns the node a segment leads to, creating it if this is the first
// route to take that step. Routes arrive in registration order, so the order
// that creates a node is the lowest any route below it can have.
func (node *routeNode) child(segment routing.Segment, order int) *routeNode {
	if segment.Kind == routing.ParameterSegment {
		if node.param == nil {
			node.param = &routeNode{minOrder: order}
		}

		return node.param
	}

	if child, exists := node.literals[segment.Text]; exists {
		return child
	}

	child := &routeNode{minOrder: order}
	if node.literals == nil {
		node.literals = map[string]*routeNode{}
	}
	node.literals[segment.Text] = child

	return child
}

// match returns the route that serves the URL for the method — the first
// registered of those that match — or nil when none does.
func (index *routeIndex) match(url string, method string) *pathHandler {
	walk := routeWalk{method: method, bestOrder: math.MaxInt}

	if index.root != nil && len(url) > 0 && url[0] == '/' {
		pieces := url[1:]

		// A route path holds its trailing slash as a rule about how the path ends
		// rather than as a piece of its own, so the URL's is read off before the walk.
		walk.trailingSlash = strings.HasSuffix(pieces, "/")
		if walk.trailingSlash {
			pieces = pieces[:len(pieces)-1]
		}

		if pieces == "" {
			walk.acceptAt(index.root)
		} else {
			walk.visit(index.root, pieces)
		}
	}

	for _, handler := range index.wildcards {
		if handler.order >= walk.bestOrder {
			break
		}

		if handler.GetHandlerByMethod(method) != nil && handler.PathMatcher.MatchesURL(url) {
			walk.best = handler
			walk.bestOrder = handler.order

			break
		}
	}

	return walk.best
}

type routeWalk struct {
	method        string
	best          *pathHandler
	bestOrder     int
	trailingSlash bool
}

// visit descends the node by the first of the URL pieces it has left.
func (walk *routeWalk) visit(node *routeNode, pieces string) {
	if node.minOrder >= walk.bestOrder {
		return
	}

	piece, rest, more := strings.Cut(pieces, "/")
	child := node.literals[piece]

	// A parameter takes any piece there is, but never the absence of one.
	param := node.param
	if piece == "" {
		param = nil
	}

	if !more {
		if child != nil {
			walk.acceptAt(child)
		}
		if param != nil {
			walk.acceptAt(param)
		}

		return
	}

	if child != nil {
		walk.visit(child, rest)
	}
	if param != nil {
		walk.visit(param, rest)
	}
}

// acceptAt takes the best route ending at a node the walk reached, which means
// every segment of it matched. What the segments do not carry is how the path
// spells its end, so the trailing slash is settled here.
func (walk *routeWalk) acceptAt(node *routeNode) {
	if node.minOrder >= walk.bestOrder {
		return
	}

	for _, handler := range node.routes {
		if handler.order >= walk.bestOrder {
			return
		}

		if walk.trailingSlash && !handler.PathMatcher.OptionalTrailingSlash() {
			continue
		}

		if handler.GetHandlerByMethod(walk.method) != nil {
			walk.best = handler
			walk.bestOrder = handler.order

			return
		}
	}
}
