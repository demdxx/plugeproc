package docker

import (
	"bytes"
	"context"
	"io"

	"github.com/demdxx/plugeproc/driver"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/pkg/errors"
)

type containerExec struct {
	process types.HijackedResponse // types.HijackedResponse remains in api/types across versions
}

func wrapExec(process types.HijackedResponse) *containerExec {
	return &containerExec{process: process}
}

func (c *containerExec) SendParams(params driver.Params) error {
	if !params.HasInput() {
		return nil
	}
	input, err := params.InputStream(false)
	if err != nil {
		return err
	}
	if _, err = io.Copy(c.process.Conn, input); err != nil {
		return err
	}
	// Signal EOF to the container process so it stops waiting for more input.
	return c.process.CloseWrite()
}

func (c *containerExec) ReadOutput(out *driver.Output) error {
	outWriter, err := out.Target()
	if err != nil {
		return err
	}
	var stdErr bytes.Buffer
	if _, err = stdcopy.StdCopy(outWriter, &stdErr, c.process.Reader); err != nil {
		return err
	}
	if stdErr.Len() > 0 {
		return errors.Wrap(ErrCallExecute, stdErr.String())
	}
	return nil
}

func (c *containerExec) IsValid() bool {
	return c != nil && c.process.Conn != nil
}

func (c *containerExec) Release() error {
	if c == nil || c.process.Conn == nil {
		return nil
	}
	if err := c.process.Conn.Close(); err != nil {
		return err
	}
	c.process.Conn = nil
	return nil
}

type containerConnect struct {
	cli             *client.Client
	streamType      bool
	removeAfterDone bool
	containerID     string
	output          *containerExec
	generalExec     *containerExec
}

func (c *containerConnect) exec(ctx context.Context, command []string, params driver.Params, out *driver.Output) (err error) {
	var cExec *containerExec

	if c.generalExec.IsValid() {
		cExec = c.generalExec
	} else if cExec, err = c.execWrapper(ctx, command, params); err != nil {
		return err
	}

	if c.streamType {
		c.generalExec = cExec
	} else {
		defer func() { _ = cExec.Release() }()
	}

	// When cExec was created by execWrapper its own connection carries stdout;
	// when it is the generalExec (container attach) we read from c.output instead.
	readSrc := cExec
	if c.generalExec.IsValid() && c.generalExec == cExec {
		readSrc = c.output
	}
	if err = cExec.SendParams(params); err == nil {
		err = readSrc.ReadOutput(out)
	}
	if err != nil && c.streamType {
		_ = cExec.Release()
	}
	return err
}

func (c *containerConnect) execWrapper(ctx context.Context, command []string, params driver.Params) (*containerExec, error) {
	// Docker exec takes discrete argv elements — use plain substitution without shell quoting.
	prepared, err := params.PrepareMacros(``, ``, command...)
	if err != nil {
		return nil, err
	}

	// types.ExecConfig → container.ExecOptions in Docker SDK v28+
	execID, err := c.cli.ContainerExecCreate(ctx, c.containerID,
		container.ExecOptions{Cmd: prepared, AttachStdout: true, AttachStderr: true, AttachStdin: true})
	if err != nil {
		return nil, errors.Wrap(ErrContainerExecCreate, err.Error())
	}

	// types.ExecStartCheck → container.ExecAttachOptions (alias of ExecStartOptions) in Docker SDK v28+
	resp, err := c.cli.ContainerExecAttach(ctx, execID.ID, container.ExecAttachOptions{})
	if err != nil {
		return nil, errors.Wrap(ErrContainerExecAttach, err.Error())
	}
	return wrapExec(resp), nil
}

func (c *containerConnect) Release() (err error) {
	if c == nil || c.streamType {
		return nil
	}
	if err = c.generalExec.Release(); err != nil {
		return err
	}
	if c.removeAfterDone && c.containerID != "" {
		err = c.cli.ContainerRemove(context.Background(), c.containerID, container.RemoveOptions{Force: true})
		c.containerID = ""
	}
	return err
}
