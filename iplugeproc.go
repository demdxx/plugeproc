package plugeproc

import (
	"context"
	"io"

	"github.com/demdxx/plugeproc/executor"
	"github.com/demdxx/plugeproc/executor/procedure"
	"github.com/demdxx/plugeproc/models"
	"github.com/pkg/errors"
)

var ErrUnsupportedExecutor = errors.New(`unsupported executor`)

const (
	ProgTypeShell    = models.ProgTypeShell
	ProgTypeExec     = models.ProgTypeExec
	ProgTypeGoplugin = models.ProgTypeGoplugin
	IfaceDefault     = models.IfaceDefault
	IfaceStream      = models.IfaceStream
)

type (
	Param  = models.Param
	Output = models.Output
	Info   = models.Info
	_Info  Info
)

type IPlugeprog interface {
	io.Closer
	Name() string
	Info() *Info
	Exec(target any, params ...any) error
}

func (info *_Info) newExecutor(ctx context.Context) (executor.Executor, error) {
	switch info.Type {
	case ProgTypeShell, ProgTypeExec:
		if info.Interface == IfaceStream {
			return &wrapper{execDriver: procedure.NewStreamExecutor(ctx, info.Command, info.Args)}, nil
		}
		return &wrapper{execDriver: procedure.NewCallExecutor(ctx, info.Command, info.Args)}, nil
	default:
		return nil, errors.Wrap(ErrUnsupportedExecutor, info.Type)
	}
}
