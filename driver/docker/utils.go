package docker

import (
	"context"
	"slices"

	"github.com/demdxx/plugeproc/driver"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
)

func getOrCreateContainer(
	ctx context.Context,
	cli *client.Client,
	name string,
	params driver.Params,
	config *container.Config,
	hostConfig *container.HostConfig,
	networkingConfig *network.NetworkingConfig,
	platformConfig *ocispec.Platform,
) (started bool, id string, err error) {
	if name != "" {
		if id, err = getContainerByName(ctx, cli, name); err != nil || id != "" {
			return true, id, err
		}
	}
	return runContainer(ctx, cli, name, params, config, hostConfig, networkingConfig, platformConfig)
}

func runContainer(
	ctx context.Context,
	cli *client.Client,
	name string,
	params driver.Params,
	config *container.Config,
	hostConfig *container.HostConfig,
	networkingConfig *network.NetworkingConfig,
	platformConfig *ocispec.Platform,
) (started bool, id string, err error) {
	id, err = createContainer(ctx, cli, name, params, config, hostConfig, networkingConfig, platformConfig)
	if err != nil {
		return false, "", err
	}
	if err = cli.ContainerStart(ctx, id, container.StartOptions{}); err != nil {
		_ = cli.ContainerRemove(ctx, id, container.RemoveOptions{Force: true})
		return false, "", errors.Wrap(ErrContainerStart, err.Error())
	}
	return false, id, nil
}

// createContainer creates (but does not start) a container after applying macro
// substitution to Cmd and Entrypoint.  Callers that need to attach before
// starting should use this function followed by ContainerAttach / ContainerStart.
func createContainer(
	ctx context.Context,
	cli *client.Client,
	name string,
	params driver.Params,
	config *container.Config,
	hostConfig *container.HostConfig,
	networkingConfig *network.NetworkingConfig,
	platformConfig *ocispec.Platform,
) (id string, err error) {
	containerConfig := *config

	// Docker accepts discrete argv elements — plain macro substitution without shell quoting.
	containerConfig.Cmd, err = params.PrepareMacros(``, ``, containerConfig.Cmd...)
	if err != nil {
		return "", err
	}
	containerConfig.Entrypoint, err = params.PrepareMacros(``, ``, containerConfig.Entrypoint...)
	if err != nil {
		return "", err
	}

	resp, err := cli.ContainerCreate(ctx, &containerConfig, hostConfig, networkingConfig, platformConfig, name)
	if err != nil {
		return "", errors.Wrap(ErrContainerCreate, err.Error())
	}
	return resp.ID, nil
}

func getContainerByName(ctx context.Context, cli *client.Client, name string) (string, error) {
	list, err := cli.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		return "", err
	}
	for _, c := range list {
		if slices.Contains(c.Names, name) {
			return c.ID, nil
		}
	}
	return "", nil
}
