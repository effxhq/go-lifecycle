package lifecycle

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	initialize = "initialize"
	run        = "run"
	start      = "start"
	shutdown   = "shutdown"
)

func countingPlugin() (map[string]int, *PluginFuncs) {
	counts := make(map[string]int)

	return counts, &PluginFuncs{
		InitializeFunc: func(app *Application) error {
			counts[initialize]++
			return nil
		},
		RunFunc: func(app *Application) error {
			counts[run]++
			return nil
		},
		StartFunc: func(app *Application) error {
			counts[start]++
			return nil
		},
		ShutdownFunc: func(app *Application) error {
			counts[shutdown]++
			return nil
		},
	}
}

func newTestApp(term func(err error)) *Application {
	app := &Application{}
	app.on.Do(app.init) // force initialization
	app.term = term
	return app
}

func Test_ApplicationInitialize_Error(t *testing.T) {
	app := newTestApp(func(err error) {
		require.NoError(t, err, "application unexpectedly failed with error")
	})

	counts, executionCountPlugin := countingPlugin()

	app.Initialize(
		executionCountPlugin,
	)

	app.Run()

	require.Equal(t, 1, counts[initialize], "unexpected initialize count")
	require.Equal(t, 0, counts[start], "unexpected start count")
	require.Equal(t, 1, counts[run], "unexpected run count")
	require.Equal(t, 1, counts[shutdown], "unexpected shutdown count")
}

func Test_ApplicationRun(t *testing.T) {
	app := newTestApp(func(err error) {
		require.NoError(t, err, "application unexpectedly failed with error")
	})

	counts, executionCountPlugin := countingPlugin()

	app.Initialize(
		executionCountPlugin,
	)

	app.Run()

	require.Equal(t, 1, counts[initialize], "unexpected initialize count")
	require.Equal(t, 0, counts[start], "unexpected start count")
	require.Equal(t, 1, counts[run], "unexpected run count")
	require.Equal(t, 1, counts[shutdown], "unexpected shutdown count")
}

func Test_ApplicationRun_Error(t *testing.T) {
	app := newTestApp(func(err error) {
		require.Error(t, err, "application did not fail with error")
		require.Equal(t, "something went wrong", err.Error())
	})

	counts, executionCountPlugin := countingPlugin()

	app.Initialize(
		executionCountPlugin,
		&PluginFuncs{
			RunFunc: func(app *Application) error {
				return fmt.Errorf("something went wrong")
			},
		},
	)

	app.Run()

	require.Equal(t, 1, counts[initialize], "unexpected initialize count")
	require.Equal(t, 0, counts[start], "unexpected start count")
	require.Equal(t, 1, counts[run], "unexpected run count")
	require.Equal(t, 1, counts[shutdown], "unexpected shutdown count")
}

func Test_ApplicationStart(t *testing.T) {
	app := newTestApp(func(err error) {
		require.NoError(t, err, "application unexpectedly failed with error")
	})

	counts, executionCountPlugin := countingPlugin()

	done := make(chan bool, 1)
	defer close(done)

	app.Initialize(
		executionCountPlugin,
		&PluginFuncs{
			StartFunc: func(app *Application) error {
				require.Equal(t, 1, counts[initialize], "unexpected initialize count")
				require.Equal(t, 1, counts[start], "unexpected start count")
				require.Equal(t, 0, counts[run], "unexpected run count")
				require.Equal(t, 0, counts[shutdown], "unexpected shutdown count")
				done <- true

				return nil
			},
		},
	)

	go app.Start()

	// wait for start
	<-done

	// manually trigger shutdown
	app.shutdown(nil)

	require.Equal(t, 1, counts[initialize], "unexpected initialize count")
	require.Equal(t, 1, counts[start], "unexpected start count")
	require.Equal(t, 0, counts[run], "unexpected run count")
	require.Equal(t, 1, counts[shutdown], "unexpected shutdown count")
}

func Test_ApplicationStart_Error(t *testing.T) {
	app := newTestApp(func(err error) {
		require.Error(t, err, "application did not fail with error")
		require.Equal(t, "something went wrong", err.Error())
	})

	counts, executionCountPlugin := countingPlugin()

	done := make(chan bool, 1)
	defer close(done)

	app.Initialize(
		&PluginFuncs{
			ShutdownFunc: func(app *Application) error {
				// trigger here since the prior start marks the end of the start loop
				done <- true
				return nil
			},
		},
		executionCountPlugin,
		&PluginFuncs{
			StartFunc: func(app *Application) error {
				return fmt.Errorf("something went wrong")
			},
		},
	)

	go app.Start()

	// wait for start
	<-done

	require.Equal(t, 1, counts[initialize], "unexpected initialize count")
	require.Equal(t, 1, counts[start], "unexpected start count")
	require.Equal(t, 0, counts[run], "unexpected run count")
	require.Equal(t, 1, counts[shutdown], "unexpected shutdown count")
}
