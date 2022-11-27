package govalin

// Plugin is a way of interacting on a Govalin instance.
type Plugin interface {
	// Apply will provide the Plugin with the Govalin app instance so it can
	// add new routes or act upon the Govalin instance in other plugin'y ways.
	Apply(app *App)
}
