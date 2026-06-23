package plugeproc

import "github.com/pkg/errors"

var (
	// ErrUnsupportedDriver is returned when the manifest specifies an unknown type.
	ErrUnsupportedDriver = errors.New("unsupported driver")

	// ErrProcNotFound is returned when a named proc does not exist in the store.
	ErrProcNotFound = errors.New("proc not found")

	// ErrInvalidCountOfParams is returned when the number of supplied params
	// does not match the manifest definition.
	ErrInvalidCountOfParams = errors.New("invalid number of params, required")
)
