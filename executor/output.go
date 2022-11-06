package executor

import (
	"bytes"
	"encoding/json"
	"io"
)

type extBuffer struct {
	bytes.Buffer
}

func (b *extBuffer) Close() error { return nil }

type Output struct {
	Type  string
	Value io.ReadWriteCloser
}

func (out *Output) Target() (_ io.Writer, err error) {
	if out.Type == "file" {
		out.Value, err = tempFrom(nil)
	} else {
		out.Value = &extBuffer{}
	}
	return out.Value, err
}

func (out *Output) JSON(target any) error {
	return json.NewDecoder(out.Value).Decode(target)
}

func (out *Output) Release() (err error) {
	if out.Value != nil {
		if c, _ := out.Value.(io.Closer); c != nil {
			err = c.Close()
		}
		out.Value = nil
	}
	return err
}
