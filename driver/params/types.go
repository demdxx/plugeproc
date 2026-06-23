// Package params provides constructor helpers for building driver.Param values.
package params

import (
	"bytes"
	"io"
	"os"

	"github.com/demdxx/plugeproc/driver"
)

// String creates a string-typed param.
func String(name, value string) *driver.Param {
	return &driver.Param{Name: name, Type: driver.TypeString, Value: value}
}

// JSON creates a JSON-typed param.
func JSON(name string, value any) *driver.Param {
	return &driver.Param{Name: name, Type: driver.TypeJSON, Value: value}
}

// Binary creates a binary-typed param.
func Binary(name string, value any) *driver.Param {
	return &driver.Param{Name: name, Type: driver.TypeBinary, Value: value}
}

// Filepath creates a file path param.
func Filepath(name, filepath string) *driver.Param {
	return &driver.Param{Name: name, Type: driver.TypeFile, Value: filepath}
}

// Stream creates an input stream param.
func Stream(name string, stream any) *driver.Param {
	return &driver.Param{Name: name, Type: driver.TypeFile, Value: stream, IsInput: true}
}

// TmpFile creates a temp-file param.
func TmpFile(name string, stream any) *driver.Param {
	return &driver.Param{Name: name, Type: driver.TypeFile, Value: stream, IsTmpFile: true}
}

// P creates a param inferred from the value type.
func P(name string, value any) *driver.Param {
	switch v := value.(type) {
	case string:
		return String(name, v)
	case []byte:
		return Binary(name, v)
	case io.Reader, *os.File, *bytes.Buffer:
		return Stream(name, v)
	default:
		return JSON(name, value)
	}
}

// InP creates an anonymous input param.
func InP(value any) *driver.Param {
	return P("", value).AsInput()
}

// OutP creates an output param backed by an Output.
func OutP(name, vtype string, out *driver.Output) *driver.Param {
	return &driver.Param{Name: name, Type: vtype, IsTmpFile: true, Out: out}
}
