package driver

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
)

type extBuffer struct {
	bytes.Buffer
}

func (b *extBuffer) Close() error { return nil }

// Output captures the result written by an executed procedure.
type Output struct {
	Type          string
	Value         io.ReadWriteCloser
	IsTmpFilepath bool
	tempFilepath  string
}

// String returns the captured output as a string.
func (out *Output) String() string {
	return string(out.Bytes())
}

// Bytes drains and returns the captured output.
func (out *Output) Bytes() []byte {
	if out.Value == nil {
		return nil
	}
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, out.Value)
	return buf.Bytes()
}

// TargetFilepath returns (or creates) the temp file path for file-based output.
func (out *Output) TargetFilepath() (_ string, err error) {
	if out.tempFilepath == "" {
		if out.tempFilepath, err = tempFilepath(); err != nil {
			return "", err
		}
	}
	return out.tempFilepath, err
}

// Target returns the writer where the procedure should write its output.
func (out *Output) Target() (_ io.Writer, err error) {
	if !out.IsTmpFilepath && out.Type == TypeFile {
		out.Value, err = tempFrom(nil)
	} else {
		out.Value = &extBuffer{}
	}
	return out.Value, err
}

// ProcessStreamResponse reads one framed response from a persistent stream process.
func (out *Output) ProcessStreamResponse(stream io.Reader) error {
	target, err := out.Target()
	if err != nil {
		return err
	}

	if out.Type == TypeBinary {
		bsize := make([]byte, 4)
		if _, err = stream.Read(bsize); err != nil {
			return err
		}
		rsize := binary.LittleEndian.Uint32(bsize)
		_, err = io.CopyN(target, stream, int64(rsize))
		return err
	}

	var line []byte
	switch r := stream.(type) {
	case *bufio.Reader:
		line, _, err = r.ReadLine()
	default:
		line, _, err = bufio.NewReader(stream).ReadLine()
	}
	if err != nil {
		return err
	}
	_, err = target.Write(line)
	return err
}

// JSON decodes the captured output into target.
func (out *Output) JSON(target any) error {
	return json.NewDecoder(out.Value).Decode(target)
}

// MappingResult copies or decodes the captured output into the caller's target value.
func (out *Output) MappingResult(target any) (err error) {
	if out.IsTmpFilepath && out.tempFilepath != "" {
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

// Release closes and clears the output value.
func (out *Output) Release() (err error) {
	if out.Value != nil {
		if c, _ := out.Value.(interface{ Close() error }); c != nil {
			err = c.Close()
		}
		out.Value = nil
	}
	return err
}
