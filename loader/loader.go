package loader

import "github.com/demdxx/plugeproc/manifest"

// Loader discovers and parses procedure manifests from some source.
type Loader interface {
	Load() ([]*manifest.Manifest, error)
}
