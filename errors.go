package lifecycle

import (
	"fmt"
)

var (
	ErrInitializeAfterStartup = fmt.Errorf("cannot initialize application after startup")
	ErrMigrateOrStart         = fmt.Errorf("invalid state: cannot run both Start and Migrate")
)
