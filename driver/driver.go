package driver

import (
	"context"
	"io"
)

// Driver is the execution backend for a procedure.
type Driver interface {
	io.Closer
	Exec(ctx context.Context, params []*Param, out *Output) error
}
