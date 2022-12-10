package plugeproc

import (
	"context"

	"github.com/pkg/errors"
	"go.uber.org/multierr"

	"github.com/demdxx/gocast/v2"
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
		cparams executor.Params
		out     = &executor.Output{
			Type:           pl.info.Output.Type,
			AsTempFilepath: pl.info.Output.AsTempFile,
		}
	)
	if len(params) != len(pl.info.Params) {
		return errors.Wrap(ErrInvalidCountOfParams, gocast.Str(len(pl.info.Params)))
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
	// In some cases we have to define the target filepath as output
	if pl.info.Output.Name != "" && pl.info.Output.AsTempFile {
		cparams = append(cparams, &executor.Param{
			Name:       pl.info.Output.Name,
			Type:       pl.info.Output.Type,
			Value:      nil,
			IsInput:    false,
			AsTempFile: true,
			Out:        out,
		})
	}
	if err = pl.executor.Exec(cparams, out); err != nil {
		return err
	}
	err = out.MappingResult(target)
	return multierr.Append(err, out.Release())
}

func (pl *plugeproc) Close() error {
	return pl.executor.Close()
}
