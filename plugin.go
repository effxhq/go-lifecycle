package lifecycle

type Plugin interface {
	Initialize(app *Application) error
	Migrate(app *Application) error
	Start(app *Application) error
	Shutdown(app *Application) error
}

// PluginFuncs can be used to write partial stateless plugins.
type PluginFuncs struct {
	InitializeFunc func(app *Application) error
	MigrateFunc    func(app *Application) error
	StartFunc      func(app *Application) error
	ShutdownFunc   func(app *Application) error
}

func (p PluginFuncs) Initialize(app *Application) error {
	if p.InitializeFunc == nil {
		return nil
	}
	return p.InitializeFunc(app)
}

func (p PluginFuncs) Migrate(app *Application) error {
	if p.MigrateFunc == nil {
		return nil
	}
	return p.MigrateFunc(app)
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
