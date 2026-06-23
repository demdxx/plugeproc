package plugeproc

import (
	"context"

	"github.com/demdxx/plugeproc/loader"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
)

// Store is a registry of named procedures.
type Store struct {
	procs []Proc
}

// NewStoreFromLoader builds a Store by loading all manifests from the given loader.
func NewStoreFromLoader(ctx context.Context, l loader.Loader) (*Store, error) {
	manifests, err := l.Load()
	if err != nil {
		return nil, err
	}
	procs := make([]Proc, 0, len(manifests))
	for _, m := range manifests {
		p, err := New(m)
		if err != nil {
			return nil, err
		}
		procs = append(procs, p)
	}
	return NewStore(procs...), nil
}

// NewStore creates a Store from an explicit list of procs.
func NewStore(procs ...Proc) *Store {
	return &Store{procs: procs}
}

// Register adds procs to the store and returns the store for chaining.
func (s *Store) Register(procs ...Proc) *Store {
	s.procs = append(s.procs, procs...)
	return s
}

// Get returns the named proc, or nil if not found.
func (s *Store) Get(name string) Proc {
	for _, p := range s.procs {
		if p.Name() == name {
			return p
		}
	}
	return nil
}

// Exec finds and executes a proc by name.
func (s *Store) Exec(ctx context.Context, name string, target any, params ...any) error {
	p := s.Get(name)
	if p == nil {
		return errors.Wrap(ErrProcNotFound, name)
	}
	return p.Exec(ctx, target, params...)
}

// Release closes all procs in the store.
func (s *Store) Release() (err error) {
	for _, p := range s.procs {
		err = multierr.Append(err, p.Release())
	}
	return err
}
