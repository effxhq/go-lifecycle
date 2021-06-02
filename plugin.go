package lifecycle

type Plugin interface {
	Initialize(app *Application) error
	Run(app *Application) error
	Start(app *Application) error
	Shutdown(app *Application) error
}

// PluginFuncs can be used to write partial stateless plugins.
type PluginFuncs struct {
	InitializeFunc func(app *Application) error
	RunFunc        func(app *Application) error
	StartFunc      func(app *Application) error
	ShutdownFunc   func(app *Application) error
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
