// Package lifecycle is used to manage the various phases of an applications life. Elements of this package ensure
// plugins are run, started, and shutdown according to how the application was invoked. Companies can wrap an
// Application with their CLI wrapper of choice. We intentionally left the CLI glue layer off as it allows consumers
// to bring in their tooling of choice.
package lifecycle
