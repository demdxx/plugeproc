package fs

import (
	"github.com/demdxx/gocast/v2"
	"github.com/demdxx/plugeproc/decode"
	decodejson "github.com/demdxx/plugeproc/decode/json"
)

type options struct {
	procMetaSuffix string
	decoder        decode.Decoder
}

func (o *options) ProcMetaSuffix() string {
	return gocast.Or(o.procMetaSuffix, defaultProcMetaSuffix)
}

func (o *options) Decoder() decode.Decoder {
	if o.decoder == nil {
		o.decoder = decodejson.Decoder
	}
	return o.decoder
}

// Option configures the filesystem loader.
type Option func(*options)

// WithDecoder sets the decoder(s) used to parse manifest files.
func WithDecoder(decoders ...decode.Decoder) Option {
	return func(o *options) {
		o.decoder = decode.Merge(decoders...)
	}
}

// WithProcMetaSuffix overrides the manifest file suffix (default: ".eproc").
func WithProcMetaSuffix(s string) Option {
	return func(o *options) { o.procMetaSuffix = s }
}
