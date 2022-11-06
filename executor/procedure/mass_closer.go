package procedure

import (
	"io"

	"go.uber.org/multierr"
)

type massCloser []io.Closer

func (c massCloser) Close() (err error) {
	for _, cl := range c {
		err = multierr.Append(err, cl.Close())
	}
	return err
}
