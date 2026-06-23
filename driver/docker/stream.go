package docker

import (
	"bufio"
	"context"
	"io"

	"github.com/demdxx/plugeproc/driver"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// StreamDriver keeps a single Docker container alive and exchanges data through its
// stdin/stdout for every Exec call.  The container image must run an interactive
// loop that reads one request per line and writes one response per line.
type StreamDriver struct {
	cli *client.Client

	containerName    string
	containerConfig  *container.Config
	hostConfig       *container.HostConfig
	networkingConfig *network.NetworkingConfig
	platformConfig   *ocispec.Platform
	pullImage        bool

	containerID string
	conn        *types.HijackedResponse
	stdout      *bufio.Reader
}

// NewStreamDriver creates a Docker stream driver.
func NewStreamDriver(cli *client.Client, _ []string, opts ...Option) *StreamDriver {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	return &StreamDriver{
		cli:              cli,
		containerName:    o.containerName,
		containerConfig:  o.containerConfig,
		hostConfig:       o.hostConfig,
		networkingConfig: o.networkingConfig,
		platformConfig:   o.platformConfig,
		pullImage:        o.pullImage,
	}
}

// Exec sends params to the container stdin and reads one framed response from stdout.
func (d *StreamDriver) Exec(ctx context.Context, params []*driver.Param, out *driver.Output) error {
	if d.conn == nil {
		if err := d.establish(ctx); err != nil {
			return err
		}
	}
	input, err := driver.Params(params).InputStream(true)
	if err != nil {
		_ = d.Close()
		return err
	}
	if _, err = io.Copy(d.conn.Conn, input); err != nil {
		_ = d.Close()
		return err
	}
	if err = out.ProcessStreamResponse(d.stdout); err != nil {
		_ = d.Close()
	}
	return err
}

// establish creates and starts the container, then attaches stdin/stdout.
func (d *StreamDriver) establish(ctx context.Context) error {
	if d.pullImage {
		d.pullImage = false
		if err := pullDockerImage(ctx, d.cli, d.containerConfig.Image); err != nil {
			return err
		}
	}

	cfg := *d.containerConfig
	cfg.AttachStdin = true
	cfg.AttachStdout = true
	cfg.OpenStdin = true

	resp, err := d.cli.ContainerCreate(ctx, &cfg, d.hostConfig, d.networkingConfig, d.platformConfig, d.containerName)
	if err != nil {
		return err
	}
	d.containerID = resp.ID

	// Attach before starting so we don't miss any output.
	hres, err := d.cli.ContainerAttach(ctx, resp.ID,
		container.AttachOptions{Stdin: true, Stdout: true, Stderr: false, Stream: true})
	if err != nil {
		_ = d.removeContainer()
		return err
	}
	d.conn = &hres

	// Docker attach uses a multiplexed stream with 8-byte frame headers.
	// Pipe it through stdcopy to strip those headers before line-reading.
	pr, pw := io.Pipe()
	go func() {
		_, _ = stdcopy.StdCopy(pw, io.Discard, hres.Reader)
		_ = pw.Close()
	}()
	d.stdout = bufio.NewReader(pr)

	if err = d.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		_ = d.Close()
		return err
	}
	return nil
}

// Close stops and removes the container.
func (d *StreamDriver) Close() (err error) {
	if d.conn != nil {
		_ = d.conn.Conn.Close()
		d.conn = nil
	}
	return d.removeContainer()
}

func (d *StreamDriver) removeContainer() error {
	if d.containerID == "" {
		return nil
	}
	err := d.cli.ContainerRemove(context.Background(), d.containerID,
		container.RemoveOptions{Force: true})
	if err == nil {
		d.containerID = ""
	}
	return err
}

// pullDockerImage pulls the image and discards the progress stream.
func pullDockerImage(ctx context.Context, cli *client.Client, imageName string) error {
	rc, err := cli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return err
	}
	_, _ = io.Copy(io.Discard, rc)
	return rc.Close()
}
