package lifecycle

import (
	"fmt"
)

var (
	// ErrInitializeAfterStartup is provided to shutdown when Initialize is called after Run or Start.
	ErrInitializeAfterStartup = fmt.Errorf("cannot initialize application after startup")
	// ErrRunOrStart is provided to shutdown when both Run and Start are invoked on an Application.
	ErrRunOrStart = fmt.Errorf("cannot start and run an application in the same execution context")
)
