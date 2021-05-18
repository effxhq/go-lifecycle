package lifecycle

import (
	"context"
)

// ContextKey is a generic structure that can be used to attach metadata from the context.
type ContextKey string

func (c ContextKey) String() string {
	return "lifecycle." + string(c)
}

// Contextual represents an object that caries a context accessible via a Context() method.
type Contextual interface {
	Context() context.Context
}
