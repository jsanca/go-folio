package runtime

import (
	"errors"
	"io"
)

// CompositeRuntime manages the lifecycle of multiple io.Closer instances.
type CompositeRuntime struct {
	closers []io.Closer
}

// NewComposite creates a CompositeRuntime from the provided closers.
// Shutdown order is the reverse of registration order.
func NewComposite(closers ...io.Closer) *CompositeRuntime {
	return &CompositeRuntime{closers: closers}
}

// Close shuts down all registered closers in reverse registration order.
// All closers are attempted even if one returns an error.
// Implements io.Closer.
func (c *CompositeRuntime) Close() error {
	var errs []error
	for i := len(c.closers) - 1; i >= 0; i-- {
		if c.closers[i] == nil {
			continue
		}
		if err := c.closers[i].Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
