package docker

import (
	"context"

	"github.com/demdxx/plugeproc/driver"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
)

// CallDriver executes a command inside a Docker container for each Exec call.
type CallDriver struct {
	cli             *client.Client
	pullImage       bool
	retainContainer bool
	removeAfterDone bool
	command         []string
	streamType      bool

	containerName    string
	containerConfig  *container.Config
	hostConfig       *container.HostConfig
	networkingConfig *network.NetworkingConfig
	platformConfig   *ocispec.Platform

	containerID string
	conn        *containerConnect
}

// NewCallDriver creates a one-shot Docker driver.
func NewCallDriver(cli *client.Client, command []string, opts ...Option) *CallDriver {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	return &CallDriver{
		cli:              cli,
		command:          command,
		pullImage:        o.pullImage,
		retainContainer:  o.retainContainer,
		removeAfterDone:  o.removeAfterDone,
		containerName:    o.containerName,
		containerConfig:  o.containerConfig,
		hostConfig:       o.hostConfig,
		networkingConfig: o.networkingConfig,
		platformConfig:   o.platformConfig,
	}
}

// Exec runs the command in a container and captures its output.
func (d *CallDriver) Exec(ctx context.Context, params []*driver.Param, out *driver.Output) error {
	conn, err := d.establish(ctx, params)
	if err != nil {
		return err
	}
	// When the container is retained we must not release it after each call;
	// cleanup happens in Close().
	if !d.retainContainer {
		defer func() { _ = conn.Release() }()
	}
	return conn.exec(ctx, d.command, params, out)
}

func (d *CallDriver) establish(ctx context.Context, params driver.Params) (_ *containerConnect, err error) {
	if d.retainContainer && d.containerID != "" &&
		(d.conn != nil && (!d.isGlobalCommand() || d.conn.generalExec.IsValid())) {
		return d.conn, nil
	}

	if d.pullImage {
		d.pullImage = false
		if err := pullDockerImage(ctx, d.cli, d.containerConfig.Image); err != nil {
			return nil, errors.Wrap(ErrImagePull, err.Error())
		}
	}

	containerConfig := *d.containerConfig
	containerConfig.Tty = false

	d.conn = &containerConnect{
		cli:             d.cli,
		removeAfterDone: d.removeAfterDone,
		streamType:      d.streamType,
	}

	if d.isGlobalCommand() {
		// One-shot execution: the command IS the container's main process.
		// We must attach BEFORE starting so we never miss output from fast-exit
		// commands (e.g. `echo hello` on Linux exits in microseconds).
		if len(d.command) > 0 {
			containerConfig.Cmd = d.command
		}
		d.containerID, err = createContainer(ctx, d.cli, d.containerName,
			params, &containerConfig, d.hostConfig, d.networkingConfig, d.platformConfig)
		if err != nil {
			return nil, err
		}
		d.conn.containerID = d.containerID

		hres, err := d.cli.ContainerAttach(ctx, d.containerID,
			container.AttachOptions{Stdin: true, Stdout: true, Stderr: true, Stream: true})
		if err != nil {
			_ = d.Close()
			return nil, errors.Wrap(ErrContainerAttach, err.Error())
		}
		// Both output and generalExec point to the same attach response:
		// SendParams writes to Conn (stdin), ReadOutput reads from Reader (stdout).
		d.conn.output = wrapExec(hres)
		d.conn.generalExec = wrapExec(hres)

		if err = d.cli.ContainerStart(ctx, d.containerID, container.StartOptions{}); err != nil {
			_ = d.Close()
			return nil, errors.Wrap(ErrContainerStart, err.Error())
		}
	} else {
		// Sidecar execution: a long-running container is started first, then
		// commands are exec'd inside it via ContainerExecCreate/Attach.
		_, d.containerID, err = getOrCreateContainer(ctx, d.cli, d.containerName,
			params, &containerConfig, d.hostConfig, d.networkingConfig, d.platformConfig)
		if err != nil {
			return nil, err
		}
		d.conn.containerID = d.containerID
		// generalExec is left nil; conn.exec will call execWrapper per Exec call.
	}

	return d.conn, nil
}

func (d *CallDriver) isGlobalCommand() bool {
	return (len(d.command) == 0 || len(d.containerConfig.Cmd) == 0) &&
		(!d.retainContainer || d.streamType)
}

// Close removes the container if needed.
func (d *CallDriver) Close() (err error) {
	if d.containerID == "" {
		return nil
	}
	err = d.conn.Release()
	d.containerID = ""
	return err
}
