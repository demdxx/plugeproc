package executor

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

// Param input value accessor
type Param struct {
	Name         string
	Type         string
	Value        any
	IsInput      bool
	AsTempFile   bool
	Out          *Output
	tempFilepath string
}

func (p *Param) MacroName() string {
	return "{{" + p.Name + "}}"
}

func (p *Param) ValueStr() (string, error) {
	switch p.Type {
	case TypeBinary:
		if p.AsTempFile {
			if p.tempFilepath == "" && p.AsTempFile {
				if p.Out == nil {
					iobj, err := p.Binary()
					if err != nil {
						return "", err
					}
					if p.tempFilepath, err = tempFileCreate(iobj); err != nil {
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
		}
		return "binary-type", nil
	case TypeJSON:
		data, err := json.Marshal(p.Value)
		if err != nil {
			return "", err
		}
		return string(data), nil
	default:
		return gocast.Str(p.Value), nil
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
	if p.AsTempFile && p.tempFilepath != "" && p.Out == nil {
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
func (p Params) PrepareMacro(escSimbol, esc, template string) (string, error) {
	rep, err := p.macroReplacer(escSimbol, esc)
	if err != nil {
		return "", err
	}
	return rep.Replace(rep.Replace(template)), nil
}

// PrepareMacros automaticaly replace MACRO {{params}} on the value as a string
func (p Params) PrepareMacros(escSimbol, esc string, vals ...string) (res []string, err error) {
	rep, err := p.macroReplacer(escSimbol, esc)
	if err != nil {
		return nil, err
	}
	res = make([]string, len(vals))
	for i, v := range vals {
		res[i] = rep.Replace(rep.Replace(v))
	}
	return res, nil
}

func (p Params) macroReplacer(escSimbol, esc string) (*strings.Replacer, error) {
	repArgs := make([]string, 0, len(p))
	for _, param := range p {
		s, err := param.ValueStr()
		if err != nil {
			return nil, err
		}
		if escSimbol != "" {
			s = escSimbol + strings.ReplaceAll(
				strings.ReplaceAll(s, esc, esc+esc),
				escSimbol, esc+escSimbol) + escSimbol
		}
		repArgs = append(repArgs, param.MacroName(), s)
	}
	return strings.NewReplacer(repArgs...), nil
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
