package lifecycle

import (
	"fmt"
)

var (
	ErrInitializeAfterStartup = fmt.Errorf("cannot initialize application after startup")
	ErrRunOrStart             = fmt.Errorf("invalid state: cannot Start and Run an application in the same execution context")
)
