//go:build !windows

package shell

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/demdxx/plugeproc/driver"
	"github.com/demdxx/plugeproc/driver/params"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCallDriverText checks plain-text macro substitution via echo.
func TestCallDriverText(t *testing.T) {
	ctx := context.Background()
	drv := NewCallDriver([]string{"echo", "-n"}, []string{"{{msg}}"})

	for _, msg := range []string{"hello", "world", "foo bar"} {
		var out driver.Output
		require.NoError(t, drv.Exec(ctx, []*driver.Param{params.String("msg", msg)}, &out))
		assert.Equal(t, msg, out.String())
	}
}

// TestCallDriverBinaryInput checks stdin passthrough (cat reads from stdin).
func TestCallDriverBinaryInput(t *testing.T) {
	ctx := context.Background()
	drv := NewCallDriver([]string{"cat"}, nil)

	cases := []string{"hello", "line1\nline2", "binary \x00 data"}
	for _, data := range cases {
		var out driver.Output
		require.NoError(t, drv.Exec(ctx, []*driver.Param{params.InP(data)}, &out))
		assert.Equal(t, data, out.String())
	}
}

// TestCallDriverJSONOutput checks that a JSON param passed as stdin is echoed back
// by cat and can be decoded.
func TestCallDriverJSONOutput(t *testing.T) {
	ctx := context.Background()
	// Use cat: send a Go-encoded JSON struct via stdin and get the raw bytes back.
	drv := NewCallDriver([]string{"cat"}, nil)

	type payload struct {
		Value string `json:"value"`
		N     int    `json:"n"`
	}
	var out driver.Output
	require.NoError(t, drv.Exec(ctx, []*driver.Param{params.InP(payload{"hello", 42})}, &out))

	var result payload
	require.NoError(t, json.Unmarshal(out.Bytes(), &result))
	assert.Equal(t, "hello", result.Value)
	assert.Equal(t, 42, result.N)
}

// TestCallDriverJSONInput checks that a JSON param is forwarded via stdin and echoed back.
func TestCallDriverJSONInput(t *testing.T) {
	ctx := context.Background()
	drv := NewCallDriver([]string{"cat"}, nil)

	input := map[string]any{"key": "value", "n": 1}
	var out driver.Output
	require.NoError(t, drv.Exec(ctx, []*driver.Param{params.InP(input)}, &out))

	var got map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &got))
	assert.Equal(t, "value", got["key"])
}

// TestCallDriverTmpFile checks that a command can write to a temp-file path and the
// output is read back correctly.  tee is used because it writes stdin directly to
// the given file path without requiring any shell quoting tricks.
func TestCallDriverTmpFile(t *testing.T) {
	ctx := context.Background()
	// tee <outfile> — reads stdin, writes to the file and to stdout.
	drv := NewCallDriver([]string{"tee"}, []string{"{{outfile}}"})

	out := &driver.Output{Type: driver.TypeBinary, IsTmpFilepath: true}
	ps := driver.Params{params.InP("filecontent")}.WithOutput("outfile", out)

	require.NoError(t, drv.Exec(ctx, ps, out))

	var buf bytes.Buffer
	require.NoError(t, out.MappingResult(&buf))
	assert.NoError(t, out.Release())
	assert.Equal(t, "filecontent", buf.String())
}

// TestCallDriverMultipleCalls ensures the driver is stateless and can be reused.
func TestCallDriverMultipleCalls(t *testing.T) {
	ctx := context.Background()
	drv := NewCallDriver([]string{"echo", "-n"}, []string{"{{v}}"})

	for i, expected := range []string{"a", "b", "c"} {
		var out driver.Output
		require.NoError(t, drv.Exec(ctx, []*driver.Param{params.String("v", expected)}, &out),
			"call %d", i)
		assert.Equal(t, expected, out.String())
	}
}
