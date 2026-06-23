package docker

import "github.com/pkg/errors"

var (
	ErrImagePull           = errors.New("docker.image.pull")
	ErrContainerCreate     = errors.New("docker.container.create")
	ErrContainerStart      = errors.New("docker.container.start")
	ErrContainerAttach     = errors.New("docker.container.attach")
	ErrContainerExecCreate = errors.New("docker.container.exec.create")
	ErrContainerExecAttach = errors.New("docker.container.exec.attach")
	ErrCallExecute         = errors.New("docker.call")
)
