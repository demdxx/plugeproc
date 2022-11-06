package plugeproc

import (
	"context"

	"github.com/demdxx/plugeproc/loader"
)

// Plugstore provides access to the list of external proc
type Plugstore struct {
	eplugs []IPlugeprog
}

// NewPlugstoreFromLoader interface accessor
func NewPlugstoreFromLoader(ctx context.Context, loader loader.Loader) (*Plugstore, error) {
	einfo, err := loader.Load()
	if err != nil {
		return nil, err
	}
	eplugs := make([]IPlugeprog, 0, len(einfo))
	for _, info := range einfo {
		eplug, err := New(ctx, info)
		if err != nil {
			return nil, err
		}
		eplugs = append(eplugs, eplug)
	}
	return NewPlugstore(eplugs), nil
}

// NewPlugstore from list of external procs
func NewPlugstore(eplugs []IPlugeprog) *Plugstore {
	return &Plugstore{eplugs: eplugs}
}

// Get external proc by name
func (ps *Plugstore) Get(name string) IPlugeprog {
	for _, plug := range ps.eplugs {
		if plug.Name() == name {
			return plug
		}
	}
	return nil
}
