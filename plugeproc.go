package plugeproc

import (
	"context"
	"io"

	"github.com/pkg/errors"

	"github.com/demdxx/gocast"
	"github.com/demdxx/plugeproc/executor"
)

var ErrInvalidCountOfParams = errors.New(`invalid number of params, required`)

type plugeproc struct {
	info     *Info
	executor executor.Executor
}

func New(ctx context.Context, info *Info) (IPlugeprog, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	exec, err := (*_Info)(info).newExecutor(ctx)
	if err != nil {
		return nil, err
	}
	return &plugeproc{info: info, executor: exec}, nil
}

func (pl *plugeproc) Name() string {
	return pl.info.Name
}

func (pl *plugeproc) Info() *Info {
	return pl.info
}

func (pl *plugeproc) Exec(target any, params ...any) (err error) {
	var (
		out     = &executor.Output{Type: pl.info.Output.Type}
		cparams executor.Params
	)
	if len(params) != len(pl.info.Params) {
		return errors.Wrap(ErrInvalidCountOfParams, gocast.ToString(len(pl.info.Params)))
	}
	defer func() { _ = cparams.Release() }()
	for i, p := range pl.info.Params {
		cparams = append(cparams, &executor.Param{
			Name:       p.Name,
			Type:       p.Type,
			Value:      params[i],
			IsInput:    p.IsInput,
			AsTempFile: p.AsTempFile,
		})
	}
	if err = pl.executor.Exec(cparams, out); err != nil {
		return err
	}
	switch t := target.(type) {
	case *io.Reader:
		*t = out.Value
	case *io.ReadWriter:
		*t = out.Value
	case *io.ReadCloser:
		*t = out.Value
	case *io.ReadWriteCloser:
		*t = out.Value
	case io.Writer:
		_, err = io.Copy(t, out.Value)
	default:
		err = out.JSON(target)
	}
	_ = out.Release()
	return err
}

func (pl *plugeproc) Close() error {
	return pl.executor.Close()
}
