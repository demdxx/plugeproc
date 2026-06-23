package docker

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/demdxx/plugeproc/driver"
	"github.com/demdxx/plugeproc/driver/params"
	dockerclient "github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newDockerClient returns a Docker client or skips the test when the daemon is unreachable.
func newDockerClient(t *testing.T) *dockerclient.Client {
	t.Helper()
	cli, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())
	if err != nil {
		t.Skip("cannot create Docker client:", err)
	}
	if _, err = cli.Ping(context.Background()); err != nil {
		_ = cli.Close()
		t.Skip("Docker daemon not reachable:", err)
	}
	t.Cleanup(func() { _ = cli.Close() })
	return cli
}

// alpine options shared across tests.
func alpineOpts(remove bool) []Option {
	return []Option{WithImage("alpine:latest"), WithPullImage(true), WithRemoveAfterDone(remove)}
}

// sidecarOpts starts a long-running container so commands can be exec'd inside it.
func sidecarOpts(remove bool) []Option {
	return []Option{
		WithSimpleContainerConfig("alpine:latest", []string{"tail", "-f", "/dev/null"}),
		WithPullImage(true), WithRemoveAfterDone(remove),
	}
}

// TestCallDriverText verifies plain-text macro substitution via echo.
func TestCallDriverText(t *testing.T) {
	cli := newDockerClient(t)
	ctx := context.Background()

	for _, msg := range []string{"hello", "world"} {
		drv := NewCallDriver(cli, []string{"echo", "-n", "{{msg}}"}, alpineOpts(true)...)
		var out driver.Output
		require.NoError(t, drv.Exec(ctx, []*driver.Param{params.String("msg", msg)}, &out))
		assert.Equal(t, msg, out.String())
		assert.NoError(t, drv.Close())
	}
}

// TestCallDriverBinaryInput verifies stdin passthrough via cat inside a sidecar container.
func TestCallDriverBinaryInput(t *testing.T) {
	cli := newDockerClient(t)
	ctx := context.Background()

	drv := NewCallDriver(cli, []string{"cat"}, sidecarOpts(true)...)
	defer func() { _ = drv.Close() }()

	for _, data := range []string{"hello", "second call"} {
		var out driver.Output
		require.NoError(t, drv.Exec(ctx, []*driver.Param{params.InP(data)}, &out))
		assert.Equal(t, data, out.String())
	}
}

// TestCallDriverJSONOutput verifies that JSON produced by a container command can be decoded.
func TestCallDriverJSONOutput(t *testing.T) {
	cli := newDockerClient(t)
	ctx := context.Background()

	drv := NewCallDriver(cli,
		[]string{"sh", "-c", `printf '{"value":"%s","n":7}' {{msg}}`},
		alpineOpts(true)...)
	defer func() { _ = drv.Close() }()

	var result struct {
		Value string `json:"value"`
		N     int    `json:"n"`
	}
	var out driver.Output
	require.NoError(t, drv.Exec(ctx, []*driver.Param{params.String("msg", "hi")}, &out))
	require.NoError(t, json.Unmarshal(out.Bytes(), &result))
	assert.Equal(t, "hi", result.Value)
	assert.Equal(t, 7, result.N)
}

// TestCallDriverJSONInput verifies that a JSON param is forwarded via stdin and echoed back.
func TestCallDriverJSONInput(t *testing.T) {
	cli := newDockerClient(t)
	ctx := context.Background()

	drv := NewCallDriver(cli, []string{"cat"}, sidecarOpts(true)...)
	defer func() { _ = drv.Close() }()

	type payload struct {
		Key string `json:"key"`
		N   int    `json:"n"`
	}
	var out driver.Output
	require.NoError(t, drv.Exec(ctx, []*driver.Param{params.InP(payload{"hello", 42})}, &out))

	var got payload
	require.NoError(t, json.Unmarshal(out.Bytes(), &got))
	assert.Equal(t, "hello", got.Key)
	assert.Equal(t, 42, got.N)
}

// TestCallDriverRetainedContainer verifies that WithRetainContainer reuses the container
// across multiple Exec calls.
func TestCallDriverRetainedContainer(t *testing.T) {
	cli := newDockerClient(t)
	ctx := context.Background()

	drv := NewCallDriver(cli, []string{"echo", "-n", "{{v}}"},
		WithSimpleContainerConfig("alpine:latest", []string{"tail", "-f", "/dev/null"}),
		WithPullImage(true), WithRemoveAfterDone(true), WithRetainContainer(true))
	defer func() { _ = drv.Close() }()

	for _, v := range []string{"first", "second", "third"} {
		var out driver.Output
		require.NoError(t, drv.Exec(ctx, []*driver.Param{params.String("v", v)}, &out))
		assert.Equal(t, v, out.String())
	}
}
