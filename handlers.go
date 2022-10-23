package govalin

import (
	"fmt"
	"net/http"

	"github.com/pkkummermo/govalin/internal/routing"
)

type pathHandler struct {
	PathFragment string
	PathMatcher  routing.PathMatcher
	Before       BeforeFunc
	After        AfterFunc
	Head         HandlerFunc
	Get          HandlerFunc
	Post         HandlerFunc
	Patch        HandlerFunc
	Put          HandlerFunc
	Delete       HandlerFunc
	Options      HandlerFunc
}

func newPathHandlerFromPathFragment(pathFragment string) (pathHandler, error) {
	pathMatcher, err := routing.NewPathMatcherFromString(pathFragment)

	if err != nil {
		return pathHandler{}, fmt.Errorf(
			"failed to create path matcher for pathFragment '%s'. Err: %w", pathFragment, err,
		)
	}

	return pathHandler{
		PathFragment: pathFragment,
		PathMatcher:  pathMatcher,
		Head:         nil,
		Get:          nil,
		Post:         nil,
		Put:          nil,
		Delete:       nil,
		Options:      nil,
	}, nil
}

func (ph *pathHandler) GetHandlerByMethod(method string) HandlerFunc {
	switch method {
	case http.MethodHead:
		return ph.Head
	case http.MethodGet:
		return ph.Get
	case http.MethodPost:
		return ph.Post
	case http.MethodPut:
		return ph.Put
	case http.MethodPatch:
		return ph.Patch
	case http.MethodDelete:
		return ph.Delete
	default:
		return nil
	}
}
