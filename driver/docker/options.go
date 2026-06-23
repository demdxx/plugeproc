package docker

import (
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type options struct {
	pullImage        bool
	retainContainer  bool
	removeAfterDone  bool
	containerName    string
	containerConfig  *container.Config
	hostConfig       *container.HostConfig
	networkingConfig *network.NetworkingConfig
	platformConfig   *ocispec.Platform
}

// Option configures a Docker driver.
type Option func(*options)

// WithContainerName sets the container name used to find or create the container.
func WithContainerName(name string) Option {
	return func(o *options) { o.containerName = name }
}

// WithPullImage controls whether the image is pulled before use.
func WithPullImage(pull bool) Option {
	return func(o *options) { o.pullImage = pull }
}

// WithImage sets the image (and optionally enables pulling).
func WithImage(image string, pullImage ...bool) Option {
	return func(o *options) {
		if o.containerConfig == nil {
			o.containerConfig = &container.Config{}
		}
		o.containerConfig.Image = image
		if len(pullImage) > 0 {
			o.pullImage = pullImage[0]
		}
	}
}

// WithRemoveAfterDone removes the container after execution.
func WithRemoveAfterDone(remove bool) Option {
	return func(o *options) { o.removeAfterDone = remove }
}

// WithRetainContainer reuses the container across multiple Exec calls.
func WithRetainContainer(retain bool) Option {
	return func(o *options) { o.retainContainer = retain }
}

// WithSimpleContainerConfig sets the image and the container entry command.
func WithSimpleContainerConfig(image string, cmd []string, pullImage ...bool) Option {
	return func(o *options) {
		o.containerConfig = &container.Config{Image: image, Cmd: cmd, Tty: false}
		if len(pullImage) > 0 {
			o.pullImage = pullImage[0]
		}
	}
}

// WithContainerConfig provides a full container.Config.
func WithContainerConfig(cfg *container.Config) Option {
	return func(o *options) { o.containerConfig = cfg }
}

// WithContainerConfigModifier allows in-place mutation of the container config.
func WithContainerConfigModifier(fn func(*container.Config)) Option {
	return func(o *options) {
		if o.containerConfig == nil {
			o.containerConfig = &container.Config{}
		}
		fn(o.containerConfig)
	}
}

// WithHostConfig provides a container.HostConfig.
func WithHostConfig(cfg *container.HostConfig) Option {
	return func(o *options) { o.hostConfig = cfg }
}

// WithHostConfigModifier allows in-place mutation of the host config.
func WithHostConfigModifier(fn func(*container.HostConfig)) Option {
	return func(o *options) {
		if o.hostConfig == nil {
			o.hostConfig = &container.HostConfig{}
		}
		fn(o.hostConfig)
	}
}

// WithNetworkingConfig provides a network.NetworkingConfig.
func WithNetworkingConfig(cfg *network.NetworkingConfig) Option {
	return func(o *options) { o.networkingConfig = cfg }
}

// WithPlatformConfig provides an OCI platform spec.
func WithPlatformConfig(cfg *ocispec.Platform) Option {
	return func(o *options) { o.platformConfig = cfg }
}
