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
	Type           string
	Value          io.ReadWriteCloser
	AsTempFilepath bool
	tempFilepath   string
}

func (out *Output) TargetFilepath() (_ string, err error) {
	if out.tempFilepath == "" {
		if out.tempFilepath, err = tempFilepath(); err != nil {
			return "", err
		}
	}
	return out.tempFilepath, err
}

func (out *Output) Target() (_ io.Writer, err error) {
	if !out.AsTempFilepath && out.Type == TypeFile {
		out.Value, err = tempFrom(nil)
	} else {
		out.Value = &extBuffer{}
	}
	return out.Value, err
}

func (out *Output) JSON(target any) error {
	return json.NewDecoder(out.Value).Decode(target)
}

func (out *Output) MappingResult(target any) (err error) {
	if out.AsTempFilepath && out.tempFilepath != "" {
		if out.Value, err = tempOpen(out.tempFilepath); err != nil {
			out.Value = nil
			return err
		}
	}
	switch t := target.(type) {
	case *io.Reader:
		*t = out.Value
		out.Value = nil
	case *io.ReadWriter:
		*t = out.Value
		out.Value = nil
	case *io.ReadCloser:
		*t = out.Value
		out.Value = nil
	case *io.ReadWriteCloser:
		*t = out.Value
		out.Value = nil
	case io.Writer:
		_, err = io.Copy(t, out.Value)
	default:
		err = out.JSON(target)
	}
	return err
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
