// Package json provides a JSON manifest decoder.
package json

import (
	stdjson "encoding/json"
	"fmt"
	"strings"

	"github.com/demdxx/plugeproc/decode/fun"
)

// Decoder decodes JSON manifests.
var Decoder = fun.Decoder(func(ext string, data []byte, v any) error {
	if !strings.EqualFold("json", strings.TrimLeft(ext, ".")) {
		return fmt.Errorf("unsupported file format: %s", ext)
	}
	return stdjson.Unmarshal(data, v)
})
