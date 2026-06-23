package decode

import "fmt"

// MultiDecoder tries each decoder in order until one succeeds.
type MultiDecoder []Decoder

// Merge combines multiple decoders into one. Returns the single decoder unchanged
// when only one is provided.
func Merge(decoders ...Decoder) Decoder {
	if len(decoders) == 1 {
		return decoders[0]
	}
	return MultiDecoder(decoders)
}

// Decode tries each decoder until one succeeds.
func (d MultiDecoder) Decode(ext string, data []byte, v any) error {
	for _, dec := range d {
		if err := dec.Decode(ext, data, v); err == nil {
			return nil
		}
	}
	return fmt.Errorf("no decoder supports file format: %s", ext)
}
