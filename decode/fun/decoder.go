// Package fun provides a function adapter for the decode.Decoder interface.
package fun

// Decoder wraps a plain function as a decode.Decoder.
type Decoder func(ext string, data []byte, v any) error

// Decode calls the underlying function.
func (d Decoder) Decode(ext string, data []byte, v any) error {
	return d(ext, data, v)
}
