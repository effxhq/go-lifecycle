package lifecycle

import (
	"fmt"
)

var (
	ErrInitializeAfterStartup = fmt.Errorf("cannot initialize application after startup")
	ErrRunOrStart             = fmt.Errorf("cannot start and run an application in the same execution context")
)
