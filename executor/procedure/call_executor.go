package procedure

import (
	"bytes"
	"context"
	"os/exec"
	"strings"

	"github.com/demdxx/plugeproc/executor"
	"github.com/pkg/errors"
)

// Error list...
var (
	ErrCallExecute       = errors.New("call")
	ErrCantCreateTmpFile = errors.New("invalid tmp file creation")
)

type CallExecutor struct {
	ctx context.Context

	command string
	args    []string
}

func NewCallExecutor(ctx context.Context, command string, args []string) *CallExecutor {
	if ctx == nil {
		ctx = context.Background()
	}
	return &CallExecutor{
		ctx:     ctx,
		command: command,
		args:    args,
	}
}

func (ce *CallExecutor) Exec(params []*executor.Param, out *executor.Output) error {
	// Init new command executor
	cmdStr, err := executor.Params(params).PrepareMacro(`"`, `\`, ce.command)
	if err != nil {
		return err
	}
	cmdArgs, err := executor.Params(params).PrepareMacros(`"`, `\`, ce.args...)
	if err != nil {
		return err
	}
	newCmd := cmdStr + " " + strings.Join(cmdArgs, " ")
	cmd := exec.CommandContext(ce.ctx, "/usr/bin/env", "bash", "-c", newCmd)
	cmd.Stderr = &bytes.Buffer{}
	if input, err := executor.Params(params).InputStream(false); err != nil {
		return err
	} else {
		cmd.Stdin = input
	}
	if output, err := out.Target(); err != nil {
		return err
	} else {
		cmd.Stdout = output
	}

	// Execute shell command
	if err := cmd.Run(); err != nil {
		text := cmd.Stderr.(*bytes.Buffer).String()
		return errors.Wrap(ErrCallExecute, err.Error()+" : "+text)
	}
	return nil
}

func (ce *CallExecutor) Close() error {
	return nil
}
