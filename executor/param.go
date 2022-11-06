package executor

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"

	"github.com/demdxx/gocast"
)

// Param input value accessor
type Param struct {
	Name         string
	Type         string
	Value        any
	IsInput      bool
	AsTempFile   bool
	tempFilepath string
}

func (p *Param) MacroName() string {
	return "{{" + p.Name + "}}"
}

func (p *Param) ValueStr() (string, error) {
	switch p.Type {
	case "binary":
		if p.AsTempFile {
			if p.tempFilepath == "" {
				iobj, err := p.Binary()
				if err != nil {
					return "", err
				}
				p.tempFilepath, err = tempFileCreate(iobj)
				if err != nil {
					return "", err
				}
			}
			return p.tempFilepath, nil
		}
		return "binary-type", nil
	case "json":
		data, err := json.Marshal(p.Value)
		if err != nil {
			return "", err
		}
		return string(data), nil
	default:
		return gocast.ToString(p.Value), nil
	}
}

// Binary returns the value as reader object
func (p *Param) Binary() (io.ReadCloser, error) {
	switch v := p.Value.(type) {
	case nil:
		return io.NopCloser(&bytes.Reader{}), nil
	case string:
		return io.NopCloser(strings.NewReader(v)), nil
	case []byte:
		return io.NopCloser(bytes.NewReader(v)), nil
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

// Release related objects
func (p *Param) Release() error {
	if p.tempFilepath != "" {
		if err := os.Remove(p.tempFilepath); err != nil {
			return err
		}
		p.tempFilepath = ""
	}
	return nil
}

// Params list of values
type Params []*Param

// PrepareMacro automaticaly replace MACRO {{params}} on the value as a string
func (p Params) PrepareMacro(template string) (string, error) {
	repArgs := make([]string, 0, len(p))
	for _, param := range p {
		s, err := param.ValueStr()
		if err != nil {
			return "", err
		}
		repArgs = append(repArgs, param.MacroName(), s)
	}
	rep := strings.NewReplacer(repArgs...)
	return rep.Replace(rep.Replace(template)), nil
}

func (p Params) PrepareMacros(vals ...string) (res []string, err error) {
	res = make([]string, 0, len(vals))
	for i, v := range vals {
		if res[i], err = p.PrepareMacro(v); err != nil {
			return nil, err
		}
	}
	return res, nil
}

// Release all related objects of the params
func (p Params) Release() error {
	for _, pr := range p {
		if err := pr.Release(); err != nil {
			return err
		}
	}
	return nil
}

func (p Params) InputStream(alwaysTail bool) (io.Reader, error) {
	var readers []io.Reader
	for _, pr := range p {
		if pr.IsInput {
			iobj, err := pr.Binary()
			if err != nil {
				return nil, err
			}
			readers = append(readers, iobj, bytes.NewReader([]byte("\n")))
		}
	}
	if len(readers) > 0 {
		if !alwaysTail && len(readers) == 2 {
			return readers[0], nil
		}
		return io.MultiReader(readers...), nil
	}
	return &bytes.Buffer{}, nil
}
