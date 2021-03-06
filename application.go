package lifecycle

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
)

// State defines a series of states that the given system may be in.
type State = int32

const (
	// StateInvalid marks when a system falls into an invalid state (unused).
	StateInvalid State = iota
	// StateInitial marks when a system is in the initial state and can accept plugins for initialization.
	StateInitial
	// StateRunning indicates the Run method has been invoked and the application is currently executing.
	StateRunning
	// StateStarted indicates the Start method has been invoked and the application is continuously executing.
	StateStarted
	// StateShutdown indicates the application is going into a terminated state, either due to an error or signal.
	StateShutdown
	// StateTerminated indicates the application is no longer running and typically set prior to exit.
	StateTerminated
)

// Hook is used to log semi-fatal errors encountered during state transitions.
type Hook func(phase string, err error)

// Application provides a pluggable container that manages a systems lifecycle. It ensures that plugins are initialized,
// started, and shutdown properly. Should an error occur during initialization or startup, any previous plugin needs to
// be shutdown to ensure it's cleaned up properly. To do this, the Application manages a simple state-machine.
type Application struct {
	on   sync.Once
	term func(err error)

	// components for managing state machine
	state  int32
	signal chan os.Signal
	done   chan struct{}

	// configurable elements of the application
	context context.Context
	cancel  context.CancelFunc

	hook    Hook
	plugins []Plugin
}

func (app *Application) init() {
	app.term = func(err error) {
		if err != nil {
			log.Fatal(err)
		}
	}

	app.context, app.cancel = context.WithCancel(context.Background())
	app.hook = func(phase string, err error) {}

	atomic.StoreInt32(&app.state, StateInitial)
	app.signal = make(chan os.Signal, 1)
	app.done = make(chan struct{}, 1)

	signal.Notify(app.signal, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-app.signal
		signal.Stop(app.signal)

		atomic.StoreInt32(&app.state, StateShutdown)

		for i := len(app.plugins); i > 0; i-- {
			err := app.plugins[i-1].Shutdown(app)
			if err != nil {
				app.hook("shutdown", err)
			}
		}

		app.cancel()
		close(app.done)
	}()
}

// use a context to share plugins

// WithHook configures a listener that's used to log semi-fatal errors encountered during state transitions. This is
// typically called from your logger plugin during its initialization phase.
func (app *Application) WithHook(hook Hook) {
	app.on.Do(app.init)
	app.hook = hook
}

// WithValue sets the key on the underlying application context to the provided value. This is used by plugins to pass
// objects back through to developers.
func (app *Application) WithValue(key, value interface{}) {
	app.on.Do(app.init)
	app.context = context.WithValue(app.context, key, value)
}

// Context returns the underlying context used by the application so that it make be shared with other systems. This
// intentionally protects users from accidentally overwriting it.
func (app *Application) Context() context.Context {
	app.on.Do(app.init)
	return app.context
}

var _ Contextual = &Application{}

// Initialize appends the provided list of plugins to the application and initializes each one. This method must be
// called before calling Run or Start.
func (app *Application) Initialize(plugins ...Plugin) {
	app.on.Do(app.init)

	if atomic.LoadInt32(&app.state) > StateInitial {
		app.shutdown(ErrInitializeAfterStartup)
	}

	app.plugins = append(app.plugins, plugins...)
	for _, plugin := range plugins {
		err := plugin.Initialize(app)
		if err != nil {
			app.hook("initialization", err)
			app.shutdown(err)
			return
		}
	}
}

// Run executes each plugins Run method. There is often only one of these, but some plugins (like a logger) might
// implement Run to log state transitions. Once this method is called, you will be unable to Initialize any more
// plugins. You will also be unable to call the Start method.
func (app *Application) Run() {
	app.on.Do(app.init)

	if !atomic.CompareAndSwapInt32(&app.state, StateInitial, StateRunning) {
		app.shutdown(ErrRunOrStart)
	}

	for _, plugin := range app.plugins {
		err := plugin.Run(app)
		if err != nil {
			app.hook("running", err)
			app.shutdown(err)
			return
		}
	}

	app.shutdown(nil)
}

// Start executes each plugins Start method. This is often used to start long running servers, begin stat emissions,
// or initialize control loops. Once this method is called, you will be unable to Initialize any more plugins. You will
// also be unable to call the Start method.
func (app *Application) Start() {
	app.on.Do(app.init)

	if !atomic.CompareAndSwapInt32(&app.state, StateInitial, StateStarted) {
		app.shutdown(ErrRunOrStart)
	}

	for _, plugin := range app.plugins {
		err := plugin.Start(app)
		if err != nil {
			app.hook("startup", err)
			app.shutdown(err)
			return
		}
	}

	<-app.done
}

func (app *Application) shutdown(err error) {
	app.signal <- os.Interrupt
	<-app.done

	atomic.StoreInt32(&app.state, StateTerminated)
	app.hook("terminated", err)

	app.term(err)
}
