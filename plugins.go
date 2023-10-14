package govalin

// Plugin is a way of interacting on a Govalin instance.
type Plugin interface {
	// Name should return the name of the plugin for logging and debugging
	// purposes
	Name() string

	// OnInit supplies the configuraiton will be run before any handlers are
	// added
	OnInit(*Config)

	// Apply will provide the Plugin with the Govalin app instance so it can
	// add new routes or act upon the Govalin instance in other plugin'y ways.
	Apply(app *App)
}
