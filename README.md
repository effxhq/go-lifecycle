# go-lifecycle

A state-based application lifecycle library for go. `go-lifecycle` helps manage the complexity around the
initialization, startup, and shutdown of applications. It abstracts away the need to manage any lifecycle hooks and
provides app devs with a plugin based interface. It also helps ensure that plugins are properly started and shutdown
in the event of an error.

**Influences**

* [go-kit](https://github.com/go-kit/kit)

## Usage

Here's an example of how the lifecycle library is used at effx. It's important to note that we haven't released any of
our plugins as they are very company specific. You will need to write your own plugins. Should enough interest develop,
we would be happy to work with the community.

```go
package main

import (
	"github.com/effxhq/go-lifecycle"
)

func main() {
	app := new(lifecycle.Application)

	// add plugins to the application
	app.Initialize(
		http_plugin.ServerPlugin(),
		grpc_plugin.ServerPlugin(),
		grpc_plugin.ClientPlugin("target"),
	)

	// do one of these
	app.Start() // runs application as a long running agent
	app.Run()   // runs application as a short lived job
}
```

### Passing resources through app.Context()

Plugins are free to decorate the application with resources. This allows plugins to expose pre-configured resources to
app developers. This is often useful for common boilerplate like setting up an HTTP or gRPC server.

```go
app.WithValue(lifecycle.ContextKey("grpc.server"), grpcServer)
```

For retrieval, plugins should provide helper functions for obtaining their resources from the app.

```go
grpcServer := grpc_plugin.ServerFromContext(app.Context())
// add grpc services

targetClientConn := grpc_plugin.ClientFromContext(app.Context(), "target")
// create clients
```

### Handling configuration

This system is configuration agnostic. Your organization is free to choose its own configuration language. We largely
use environment variables which makes setup rather easy.

## Plugin Development

Before diving into writing your own plugin, it is useful to first understand the `Application` state machine. It can
exist in one of the following states:

1. **Initialization** - The application is idle. Developers are free to install and configure plugins as they need.
   Should the application encounter any errors, all registered plugins are shutdown.

1. **Running** - The application runs each plugins `Run` step (if it has one). Should the application encounter any
   errors, all plugins are shutdown. Particularly useful for running database migrations or any other pre-work you might
   need to perform.

1. **Started** - The application runs each plugins `Start` step (if it has one). Should the application encounter any
   errors when starting, all plugins are shutdown. Once all plugins have been started, the main thread blocks and waits
   for shut down.

1. **Shutdown** - Triggered one of three ways. The first two deal with the prior two states. Should an application
   encounter any errors when running or starting up, they trigger a shutdown. The last way an application can be
   triggers is by sending either a `SIGTERM` or `SIGINT` signal. Once shutdown, the application runs each
   plugins `Shutdown` step.

1. **Terminated** - Once all plugins have been shutdown, the application goes into a terminated state. This happens just
   prior to system exist. If an error occurred, the system will exit with an unhealthy status code. If there were no
   errors, then system exists cleanly.

The diagram below shows how transitions occur between these states.

[![State Machine](https://mermaid.ink/img/eyJjb2RlIjoiZ3JhcGggTFJcbiAgIFxuICAgKiAtLSBJbml0aWFsaXplIC0tPiAqXG4gICAqIC0tIFN0YXJ0IC0tPiBzdGFydGVkXG4gICAqIC0tIFJ1biAtLT4gcnVubmluZ1xuICAgKiAtLSBlcnIgLS0-IHNodXRkb3duXG5cbiAgIG9zLlNJR1RFUk0gLS0-IHNodXRkb3duXG4gICBzdGFydGVkIC0tIGVyciAtLT4gc2h1dGRvd25cbiAgIHJ1bm5pbmcgLS0gZXJyPyAtLT4gc2h1dGRvd25cblxuICAgc2h1dGRvd24gLS0-IHRlcm1pbmF0ZWRcbiIsIm1lcm1haWQiOnt9LCJ1cGRhdGVFZGl0b3IiOmZhbHNlLCJhdXRvU3luYyI6dHJ1ZSwidXBkYXRlRGlhZ3JhbSI6ZmFsc2V9)](https://mermaid-js.github.io/mermaid-live-editor/edit##eyJjb2RlIjoiZ3JhcGggTFJcbiAgIFxuICAgKiAtLSBJbml0aWFsaXplIC0tPiAqXG4gICAqIC0tIFN0YXJ0IC0tPiBzdGFydGVkXG4gICAqIC0tIFJ1biAtLT4gcnVubmluZ1xuICAgKiAtLSBlcnIgLS0-IHNodXRkb3duXG5cbiAgIG9zLlNJR1RFUk0gLS0-IHNodXRkb3duXG4gICBzdGFydGVkIC0tIGVyciAtLT4gc2h1dGRvd25cbiAgIG1pZ3JhdGluZyAtLSBlcnI_IC0tPiBzaHV0ZG93blxuXG4gICBzaHV0ZG93biAtLT4gdGVybWluYXRlZFxuIiwibWVybWFpZCI6Int9IiwidXBkYXRlRWRpdG9yIjpmYWxzZSwiYXV0b1N5bmMiOnRydWUsInVwZGF0ZURpYWdyYW0iOmZhbHNlfQ)

### Using `lifecycle.PluginFuncs`

Using the `lifecycle.PluginFuncs` object is the easiest way to develop a plugin. It allows you to build partial,
stateless plugins rather easily. For example, the code block below shows how you can write a logger plugin.

```go
package logger_plugin

import (
	"context"
	"log"

	"github.com/effxhq/go-lifecycle"
)

var contextKey = lifecycle.ContextKey("logger")

func FromContext(ctx context.Context) *log.Logger {
	val := ctx.Value(contextKey)
	if val == nil {
		return nil // or default logger
	}
	return val.(*log.Logger)
}

func Plugin() lifecycle.Plugin {
	logger := log.Default()

	return &lifecycle.PluginFuncs{
		InitializeFunc: func(app *lifecycle.Application) error {
			app.WithValue(contextKey, logger)

			// the hook is used to report errors encountered during lifecycle steps.
			// applications should only have one hook.
			app.WithHook(func(phase string, err error) {
				if err != nil {
					logger.Printf("[%s] encountered err: %v", phase, err)
				}
			})

			return nil
		},
		RunFunc: func(app *lifecycle.Application) error {
			logger.Printf("running application as job")
			return nil
		},
		StartFunc: func(app *lifecycle.Application) error {
			logger.Printf("starting application as daemon")
			return nil
		},
		ShutdownFunc: func(app *lifecycle.Application) error {
			logger.Printf("shutting down")
			return nil
		},
	}
}
```

For cases where you might want to track some state, there's a `Plugin` interface that can be implemented.

### Composing plugins

Plugins support composition. This allows components to be bundled and installed together.

```go
package company_plugin

import (
	"github.com/effxhq/go-lifecycle"
)

func DefaultCompanyPlugin() lifecycle.Plugin {
	return &lifecycle.PluginFuncs{
		InitializeFunc: func(app *lifecycle.Application) error {
			app.Initialize(
				http_plugin.ServerPlugin(),
				grpc_plugin.ServerPlugin(),
				grpc_plugin.ClientPlugin("target"),
			)
			return nil
		},
	}
}
```
