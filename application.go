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
	StateMigrating
	StateStarted
	StateShutdown
	StateTerminated
)

// Hook is used to log semi-fatal errors encountered during state transitions.
type Hook func(phase string, err error)

// Application provides a pluggable container that manages a systems lifecycle. It ensures that plugins are initialized,
// started, and shutdown properly. Should an error occur during initialization or startup, any previous plugin needs to
// be shutdown to ensure it's cleaned up properly. To do this, the Application manages a simple state-machine. The
// diagram at the link below shows how an application moves through the various states.
//
// https://mermaid.ink/img/eyJjb2RlIjoiZ3JhcGggTFJcbiAgIFxuICAgKiAtLSBJbml0aWFsaXplIC0tPiAqXG4gICAqIC0tIFN0YXJ0IC0tPiBzdGFydGVkXG4gICAqIC0tIE1pZ3JhdGUgLS0-IG1pZ3JhdGluZ1xuICAgKiAtLSBlcnIgLS0-IHNodXRkb3duXG5cbiAgIG9zLlNJR1RFUk0gLS0-IHNodXRkb3duXG4gICBzdGFydGVkIC0tIGVyciAtLT4gc2h1dGRvd25cbiAgIG1pZ3JhdGluZyAtLSBlcnI_IC0tPiBzaHV0ZG93blxuXG4gICBzaHV0ZG93biAtLT4gdGVybWluYXRlZFxuIiwibWVybWFpZCI6e30sInVwZGF0ZUVkaXRvciI6ZmFsc2V9
//
// This should largely be transparent to folks. It's useful for anyone who's developing a plugin and wants to understand
// these transitions and how it relates to how they should leverage the various phases of their plugin.
//
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

func (app *Application) Migrate() {
	app.on.Do(app.init)

	if !atomic.CompareAndSwapInt32(&app.state, StateInitial, StateMigrating) {
		app.shutdown(ErrMigrateOrStart)
	}

	for _, plugin := range app.plugins {
		err := plugin.Migrate(app)
		if err != nil {
			app.hook("migration", err)
			app.shutdown(err)
			return
		}
	}

	app.shutdown(nil)
}

func (app *Application) Start() {
	app.on.Do(app.init)

	if !atomic.CompareAndSwapInt32(&app.state, StateInitial, StateStarted) {
		app.shutdown(ErrMigrateOrStart)
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
