package docker

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/demdxx/plugeproc/driver"
	"github.com/demdxx/plugeproc/driver/params"
	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// streamScript is a simple shell script that reads one JSON line and echoes one back.
const streamScript = `while IFS= read -r line; do [ -z "$line" ] && continue; echo "$line"; done`

// TestStreamDriverText sends text lines to a persistent container and reads them back.
func TestStreamDriverText(t *testing.T) {
	cli := newDockerClient(t)
	ctx := context.Background()

	drv := NewStreamDriver(cli, nil,
		WithContainerConfig(&container.Config{
			Image:     "alpine:latest",
			Cmd:       []string{"sh", "-c", streamScript},
			OpenStdin: true,
			StdinOnce: false,
			Tty:       false,
		}),
		WithPullImage(true),
	)
	defer drv.Close()

	for _, msg := range []string{"hello", "world"} {
		var out driver.Output
		require.NoError(t, drv.Exec(ctx, []*driver.Param{params.InP(msg)}, &out))
		assert.Equal(t, msg, out.String())
	}
}

// TestStreamDriverJSON sends JSON via a persistent container and decodes the response.
func TestStreamDriverJSON(t *testing.T) {
	cli := newDockerClient(t)
	ctx := context.Background()

	const script = `while IFS= read -r line; do
  [ -z "$line" ] && continue
  v=$(echo "$line" | awk -F'"v":' '{print $2}' | tr -d '} ')
  printf '{"input":%s,"output":%d}\n' "$v" "$((v*2))"
done`

	drv := NewStreamDriver(cli, nil,
		WithContainerConfig(&container.Config{
			Image:     "alpine:latest",
			Cmd:       []string{"sh", "-c", script},
			OpenStdin: true,
			StdinOnce: false,
			Tty:       false,
		}),
		WithPullImage(true),
	)
	defer drv.Close()

	for _, c := range []struct{ in, want int }{{1, 2}, {3, 6}, {5, 10}} {
		var out driver.Output
		require.NoError(t, drv.Exec(ctx,
			[]*driver.Param{params.InP(map[string]int{"v": c.in})}, &out))
		var result struct {
			Input  int `json:"input"`
			Output int `json:"output"`
		}
		require.NoError(t, json.Unmarshal(out.Bytes(), &result))
		assert.Equal(t, c.in, result.Input)
		assert.Equal(t, c.want, result.Output)
	}
}
