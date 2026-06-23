package driver

import (
	"bytes"
	"io"
	"strings"
)

// Params is a slice of runtime parameter values.
type Params []*Param

// PrepareMacro replaces all {{name}} placeholders in a single string.
func (p Params) PrepareMacro(escSymbol, esc, template string) (string, error) {
	rep, err := p.macroReplacer(escSymbol, esc)
	if err != nil {
		return "", err
	}
	return rep.Replace(rep.Replace(template)), nil
}

// PrepareMacros replaces all {{name}} placeholders in multiple strings.
func (p Params) PrepareMacros(escSymbol, esc string, vals ...string) ([]string, error) {
	rep, err := p.macroReplacer(escSymbol, esc)
	if err != nil {
		return nil, err
	}
	res := make([]string, len(vals))
	for i, v := range vals {
		res[i] = rep.Replace(rep.Replace(v))
	}
	return res, nil
}

func (p Params) macroReplacer(escSymbol, esc string) (*strings.Replacer, error) {
	repArgs := make([]string, 0, len(p)*2)
	for _, param := range p {
		s, err := param.ValueStr()
		if err != nil {
			return nil, err
		}
		if escSymbol != "" {
			s = escSymbol + strings.ReplaceAll(
				strings.ReplaceAll(s, esc, esc+esc),
				escSymbol, esc+escSymbol) + escSymbol
		}
		repArgs = append(repArgs, param.MacroName(), s)
	}
	return strings.NewReplacer(repArgs...), nil
}

// Release frees resources held by all params.
func (p Params) Release() error {
	for _, pr := range p {
		if err := pr.Release(); err != nil {
			return err
		}
	}
	return nil
}

// InputStream concatenates all input params into a single reader.
func (p Params) InputStream(alwaysTail bool) (io.Reader, error) {
	var readers []io.Reader
	for _, pr := range p {
		if pr.IsInput {
			rc, err := pr.Binary()
			if err != nil {
				return nil, err
			}
			readers = append(readers, rc, bytes.NewReader([]byte("\n")))
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

// Param returns the first param with the given name.
func (p Params) Param(name string) *Param {
	for _, pr := range p {
		if pr.Name == name {
			return pr
		}
	}
	return nil
}

// HasInput reports whether any param is marked as stdin input.
func (p Params) HasInput() bool {
	for _, pr := range p {
		if pr.IsInput {
			return true
		}
	}
	return false
}

// WithOutput appends a synthetic output param when the output uses a temp file.
func (p Params) WithOutput(name string, out *Output) Params {
	if name == "" || !out.IsTmpFilepath {
		return p
	}
	return append(p, &Param{
		Name:      name,
		Type:      out.Type,
		IsInput:   false,
		IsTmpFile: true,
		Out:       out,
	})
}
