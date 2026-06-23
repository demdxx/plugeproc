// Package yaml provides a YAML manifest decoder.
package yaml

import (
	"fmt"
	"strings"

	stdyaml "gopkg.in/yaml.v3"

	"github.com/demdxx/plugeproc/decode/fun"
)

// Decoder decodes YAML manifests (.yaml / .yml).
var Decoder = fun.Decoder(func(ext string, data []byte, v any) error {
	ext = strings.ToLower(strings.TrimLeft(ext, "."))
	if ext != "yaml" && ext != "yml" {
		return fmt.Errorf("unsupported file format: %s", ext)
	}
	return stdyaml.Unmarshal(data, v)
})
