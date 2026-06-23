//go:build !windows

package shell

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/demdxx/plugeproc/driver"
	"github.com/demdxx/plugeproc/driver/params"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStreamDriverJSON runs a bash loop that reads one JSON line per request
// and replies with a transformed JSON line.
func TestStreamDriverJSON(t *testing.T) {
	ctx := context.Background()
	drv := NewStreamDriver([]string{
		`while IFS='$\n' read -r line; do` + "\n" +
			`  [ -z "$line" ] && continue` + "\n" +
			`  V=$(printf '%s' "$line" | awk -F'"v":' '{print $2}' | awk '{print $1}' | tr -d '}')` + "\n" +
			`  echo "{\"input\":$V,\"output\":$((V*2))}"` + "\n" +
			`done`,
	}, nil)
	defer func() { _ = drv.Close() }()

	cases := []struct{ in, wantOutput int }{{1, 2}, {3, 6}, {5, 10}}
	for _, c := range cases {
		var out driver.Output
		require.NoError(t, drv.Exec(ctx,
			[]*driver.Param{params.InP(map[string]int{"v": c.in})}, &out))
		var result struct {
			Input  int `json:"input"`
			Output int `json:"output"`
		}
		require.NoError(t, json.Unmarshal(out.Bytes(), &result))
		assert.Equal(t, c.in, result.Input)
		assert.Equal(t, c.wantOutput, result.Output)
	}
}

// TestStreamDriverText runs a bash loop that echoes each line back.
func TestStreamDriverText(t *testing.T) {
	ctx := context.Background()
	drv := NewStreamDriver([]string{
		`while IFS='$\n' read -r line; do` + "\n" +
			`  [ -z "$line" ] && continue` + "\n" +
			`  echo "$line"` + "\n" +
			`done`,
	}, nil)
	defer func() { _ = drv.Close() }()

	for _, msg := range []string{"hello", "world", "stream test"} {
		var out driver.Output
		require.NoError(t, drv.Exec(ctx, []*driver.Param{params.InP(msg)}, &out))
		assert.Equal(t, msg, out.String())
	}
}

// TestStreamDriverMultipleRequests sends many requests over the same stream process.
func TestStreamDriverMultipleRequests(t *testing.T) {
	ctx := context.Background()
	drv := NewStreamDriver([]string{
		`while IFS='$\n' read -r line; do` + "\n" +
			`  [ -z "$line" ] && continue` + "\n" +
			`  echo "got:$line"` + "\n" +
			`done`,
	}, nil)
	defer func() { _ = drv.Close() }()

	for i := 0; i < 5; i++ {
		var out driver.Output
		require.NoError(t, drv.Exec(ctx, []*driver.Param{params.InP("ping")}, &out),
			"request %d", i)
		assert.Equal(t, "got:ping", out.String())
	}
}
