// Package decode provides pluggable manifest decoders.
package decode

// Decoder decodes manifest file content based on its file extension.
type Decoder interface {
	Decode(ext string, data []byte, v any) error
}
