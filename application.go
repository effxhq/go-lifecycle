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

type State = int32

const (
	StateInvalid State = iota
	StateInitial
	StateRunning
	StateStarted
	StateShutdown
	StateTerminated
)

// Hook is used to log semi-fatal errors encountered during state transitions.
type Hook func(phase string, err error)

// Application provides a pluggable container that manages a systems lifecycle. It ensures that plugins are initialized,
// started, and shutdown properly. Should an error occur during initialization or startup, any previous plugin needs to
// be shutdown to ensure it's cleaned up properly. To do this, the Application manages a simple state-machine.
type Application struct {
	on sync.Once

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

func (app *Application) WithHook(hook Hook) {
	app.on.Do(app.init)
	app.hook = hook
}

func (app *Application) WithValue(key, value interface{}) {
	app.on.Do(app.init)
	app.context = context.WithValue(app.context, key, value)
}

func (app *Application) Context() context.Context {
	app.on.Do(app.init)
	return app.context
}

var _ Contextual = &Application{}

// core

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
	if err != nil {
		log.Fatal(err)
	}
}
