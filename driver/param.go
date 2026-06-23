package driver

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"

	"github.com/demdxx/gocast/v2"
)

const (
	TypeString = "string"
	TypeJSON   = "json"
	TypeBinary = "binary"
	TypeFile   = "file"
)

// Param is the runtime value of a single procedure parameter.
type Param struct {
	Name         string
	Type         string
	Value        any
	IsInput      bool
	IsTmpFile    bool
	Out          *Output
	tempFilepath string
}

// MacroName returns the template placeholder for this param, e.g. "{{name}}".
func (p *Param) MacroName() string {
	return "{{" + p.Name + "}}"
}

// ValueStr returns the string representation used for macro substitution.
func (p *Param) ValueStr() (string, error) {
	switch {
	case p.Type == TypeBinary || (p.Type == TypeFile && p.IsTmpFile):
		if !p.IsTmpFile {
			return "binary-type", nil
		}
		if p.tempFilepath == "" {
			if p.Out == nil {
				rc, err := p.Binary()
				if err != nil {
					return "", err
				}
				if p.tempFilepath, err = tempFileCreate(rc); err != nil {
					return "", err
				}
			} else {
				var err error
				if p.tempFilepath, err = p.Out.TargetFilepath(); err != nil {
					return "", err
				}
			}
		}
		return p.tempFilepath, nil
	case p.Type == TypeJSON:
		switch v := p.Value.(type) {
		case json.RawMessage:
			return string(v), nil
		default:
			data, err := json.Marshal(p.Value)
			if err != nil {
				return "", err
			}
			return string(data), nil
		}
	default:
		return gocast.Str(p.Value), nil
	}
}

// AsInput marks the param as an stdin input.
func (p *Param) AsInput() *Param {
	p.IsInput = true
	return p
}

// Binary returns the param value as a ReadCloser for stdin piping.
func (p *Param) Binary() (io.ReadCloser, error) {
	switch v := p.Value.(type) {
	case nil:
		return io.NopCloser(&bytes.Reader{}), nil
	case string:
		return io.NopCloser(strings.NewReader(v)), nil
	case []byte:
		return io.NopCloser(bytes.NewReader(v)), nil
	case *bytes.Buffer:
		return io.NopCloser(v), nil
	case *os.File:
		return v, nil
	case io.ReadCloser:
		return v, nil
	case io.Reader:
		return io.NopCloser(v), nil
	default:
		data, err := json.Marshal(p.Value)
		if err != nil {
			return nil, err
		}
		return io.NopCloser(bytes.NewReader(data)), nil
	}
}

// Release cleans up any temporary file created for this param.
func (p *Param) Release() error {
	if p.IsTmpFile && p.tempFilepath != "" && p.Out == nil {
		if err := os.Remove(p.tempFilepath); err != nil {
			return err
		}
		p.tempFilepath = ""
	}
	return nil
}
