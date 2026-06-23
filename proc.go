package plugeproc

import (
	"context"

	"github.com/demdxx/gocast/v2"
	"github.com/demdxx/plugeproc/driver"
	dockerdriver "github.com/demdxx/plugeproc/driver/docker"
	"github.com/demdxx/plugeproc/driver/shell"
	"github.com/demdxx/plugeproc/manifest"
	"github.com/docker/docker/api/types/container"
	dockerclient "github.com/docker/docker/client"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
)

// Proc is the interface for an external procedure executor.
type Proc interface {
	Name() string
	Exec(ctx context.Context, target any, params ...any) error
	Release() error
}

type proc struct {
	m      *manifest.Manifest
	driver driver.Driver
}

// New creates a Proc from a manifest.
func New(m *manifest.Manifest) (Proc, error) {
	d, err := newDriver(m)
	if err != nil {
		return nil, err
	}
	return &proc{m: m, driver: d}, nil
}

// Name returns the procedure name from the manifest.
func (p *proc) Name() string { return p.m.Name }

// Exec validates params, runs the driver, and maps the output to target.
func (p *proc) Exec(ctx context.Context, target any, params ...any) (err error) {
	if len(params) != len(p.m.Params) {
		return errors.Wrap(ErrInvalidCountOfParams, gocast.Str(len(p.m.Params)))
	}

	var dparams driver.Params
	defer func() { _ = dparams.Release() }()

	for i, pd := range p.m.Params {
		// type: file on an input param means "write the value to a temp file and
		// pass the path as the {{name}} macro".  The legacy IsTmpFile flag carries
		// the same semantic and is left unchanged for backward compat.
		isTmpFile := pd.IsTmpFile || pd.Type == manifest.OutputFileType
		dparams = append(dparams, &driver.Param{
			Name:      pd.Name,
			Type:      pd.Type,
			Value:     params[i],
			IsInput:   pd.Stdin,
			IsTmpFile: isTmpFile,
		})
	}

	out := &driver.Output{
		Type:          outputDriverType(p.m.Output),
		IsTmpFilepath: p.m.Output.Type == manifest.OutputFileType || p.m.Output.IsTmpFile,
	}
	if err = p.driver.Exec(ctx, dparams.WithOutput(p.m.Output.Name, out), out); err != nil {
		return err
	}
	err = out.MappingResult(target)
	return multierr.Append(err, out.Release())
}

// outputDriverType maps the manifest output type to the driver-level type string.
// When the manifest says "file", the actual data format is binary (the driver reads
// the temp file as raw bytes).
func outputDriverType(o manifest.OutputDef) string {
	if o.Type == manifest.OutputFileType {
		return driver.TypeBinary
	}
	return o.Type
}

// Release closes the underlying driver.
func (p *proc) Release() error { return p.driver.Close() }

// newDriver selects and constructs the execution driver for the manifest type.
func newDriver(m *manifest.Manifest) (driver.Driver, error) {
	switch m.Driver {
	case manifest.DriverShell, manifest.DriverExec:
		var shellOpts []shell.Option
		if m.ScriptMode {
			shellOpts = append(shellOpts, shell.WithScriptMode())
		}
		if m.Mode == manifest.ModeStream {
			return shell.NewStreamDriver(m.Command, m.Args, shellOpts...), nil
		}
		return shell.NewCallDriver(m.Command, m.Args, shellOpts...), nil

	case manifest.DriverDocker:
		if m.Docker == nil {
			return nil, errors.New("docker manifest section is required for driver=docker")
		}
		cli, err := dockerclient.NewClientWithOpts(
			dockerclient.FromEnv,
			dockerclient.WithAPIVersionNegotiation(),
		)
		if err != nil {
			return nil, errors.Wrap(err, "docker client")
		}
		opts := buildDockerOptions(m.Docker)
		if m.Mode == manifest.ModeStream {
			// Stream driver creates a long-lived container; its command must be
			// written into ContainerConfig.Cmd because NewStreamDriver ignores
			// the bare command slice.
			cmd := []string(m.Command)
			opts = append(opts, dockerdriver.WithContainerConfigModifier(func(cfg *container.Config) {
				cfg.Cmd = cmd
			}))
			return dockerdriver.NewStreamDriver(cli, nil, opts...), nil
		}
		return dockerdriver.NewCallDriver(cli, m.Command, opts...), nil

	default:
		return nil, errors.Wrap(ErrUnsupportedDriver, m.Driver)
	}
}

func buildDockerOptions(cfg *manifest.DockerConf) []dockerdriver.Option {
	opts := []dockerdriver.Option{
		dockerdriver.WithImage(cfg.Image, cfg.PullImage),
		dockerdriver.WithRemoveAfterDone(cfg.RemoveAfterDone),
		dockerdriver.WithRetainContainer(cfg.RetainContainer),
	}
	if cfg.ContainerName != "" {
		opts = append(opts, dockerdriver.WithContainerName(cfg.ContainerName))
	}
	return opts
}
