package lifecycle

// Plugin defines an abstraction to developers to tie into the various lifecycle events of an application. It's
// important that plugins be written in such a way where some of their common resources may not exist.
type Plugin interface {
	// Initialize initializes the plugin given an application. During this phase you can create resources and attach
	// them to the Application so developers can pull them off and consume them.
	Initialize(app *Application) error
	// Run will execute any one-off runtime logic the plugin requires.
	Run(app *Application) error
	// Start forks and necessary go-routines and spins up any long lived components (like servers).
	Start(app *Application) error
	// Shutdown is invoked to ensure the plugin is shutdown when an error occurs elsewhere.
	Shutdown(app *Application) error
}

// PluginFuncs implements Plugin and allows for consumers to write partial stateless plugins. These are the majority of
// plugins that we write at effx, but having the common interface has it's utility.
type PluginFuncs struct {
	// InitializeFunc is an optional function that can perform initialization logic for a plugin.
	InitializeFunc func(app *Application) error
	// RunFunc is an optional function that can perform execution logic for a plugin.
	RunFunc func(app *Application) error
	// StartFunc is an optional function that can start process within a plugin.
	StartFunc func(app *Application) error
	// ShutdownFunc is an optional function that can be used to gracefully disconnect client connections.
	ShutdownFunc func(app *Application) error
}

func (p PluginFuncs) Initialize(app *Application) error {
	if p.InitializeFunc == nil {
		return nil
	}
	return p.InitializeFunc(app)
}

func (p PluginFuncs) Run(app *Application) error {
	if p.RunFunc == nil {
		return nil
	}
	return p.RunFunc(app)
}

func (p PluginFuncs) Start(app *Application) error {
	if p.StartFunc == nil {
		return nil
	}
	return p.StartFunc(app)
}

func (p PluginFuncs) Shutdown(app *Application) error {
	if p.ShutdownFunc == nil {
		return nil
	}
	return p.ShutdownFunc(app)
}

var _ Plugin = PluginFuncs{}
