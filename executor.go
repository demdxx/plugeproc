package plugeproc

import (
	"github.com/demdxx/plugeproc/executor"
)

type wrapper struct {
	execDriver executor.Executor
}

func (exe *wrapper) Exec(params []*executor.Param, out *executor.Output) error {
	if err := exe.execDriver.Exec(params, out); err != nil {
		_ = executor.Params(params).Release()
		return err
	}
	return executor.Params(params).Release()
}

func (exe *wrapper) Close() error {
	return exe.execDriver.Close()
}
