//go:build !windows

package plugeproc

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	dockerclient "github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	decodejson "github.com/demdxx/plugeproc/decode/json"
	decodeyaml "github.com/demdxx/plugeproc/decode/yaml"
	"github.com/demdxx/plugeproc/loader/fs"
)

// ─── shared helpers ──────────────────────────────────────────────────────────

func newTestStore(ctx context.Context) (*Store, error) {
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(file), "tests")
	return NewStoreFromLoader(ctx, fs.New(dir,
		fs.WithDecoder(decodejson.Decoder, decodeyaml.Decoder)))
}

// testStore creates a Store and registers cleanup automatically.
func testStore(t *testing.T) *Store {
	t.Helper()
	store, err := newTestStore(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Release() })
	return store
}

// testCtx returns a context with a generous timeout.
func testCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// requireDocker skips the test when the Docker daemon is not reachable.
func requireDocker(t *testing.T) {
	t.Helper()
	cli, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())
	if err != nil {
		t.Skip("cannot create Docker client:", err)
	}
	defer cli.Close()
	if _, err = cli.Ping(context.Background()); err != nil {
		t.Skip("Docker daemon not reachable:", err)
	}
}

// ─── shell: cat (binary passthrough) ─────────────────────────────────────────

func TestStoreCat(t *testing.T) {
	store := testStore(t)
	ctx := testCtx(t)

	p := store.Get("cat")
	require.NotNil(t, p, "proc 'cat' not found")

	for _, input := range []string{"hello", "world", "multi\nline"} {
		var buf bytes.Buffer
		require.NoError(t, p.Exec(ctx, &buf, input))
		assert.Equal(t, input, buf.String())
	}
}

// ─── shell: proc (JSON output) ────────────────────────────────────────────────

func TestStoreProc(t *testing.T) {
	store := testStore(t)
	ctx := testCtx(t)

	p := store.Get("proc")
	require.NotNil(t, p, "proc 'proc' not found")

	for _, input := range []string{"hello", "world", "json-test"} {
		var result struct {
			Input string `json:"input"`
		}
		require.NoError(t, p.Exec(ctx, &result, input))
		assert.Equal(t, input, result.Input)
	}
}

// TestStoreProcWriter verifies that proc output can be captured into an io.Writer
// and then decoded manually.
func TestStoreProcWriter(t *testing.T) {
	store := testStore(t)
	ctx := testCtx(t)

	var buf bytes.Buffer
	require.NoError(t, store.Exec(ctx, "proc", &buf, "writer-test"))

	var result struct {
		Input string `json:"input"`
	}
	require.NoError(t, json.NewDecoder(&buf).Decode(&result))
	assert.Equal(t, "writer-test", result.Input)
}

// ─── shell: stream (persistent JSON loop) ────────────────────────────────────

func TestStoreStream(t *testing.T) {
	store := testStore(t)
	ctx := testCtx(t)

	p := store.Get("stream")
	require.NotNil(t, p, "proc 'stream' not found")

	var result struct {
		Input  int `json:"input"`
		Output int `json:"output"`
	}
	for i := 0; i < 5; i++ {
		require.NoError(t, p.Exec(ctx, &result, struct {
			V int `json:"v"`
		}{V: i}))
		assert.Equal(t, i, result.Input, "iteration %d", i)
		assert.Equal(t, i*2, result.Output, "iteration %d", i)
	}
}

// ─── shell: infile (file-typed input param → temp file → cat) ───────────────

// TestStoreInFile verifies that a param with type:file writes the caller's value
// to a temp file and passes its path as a macro argument to the script.
func TestStoreInFile(t *testing.T) {
	store := testStore(t)
	ctx := testCtx(t)

	p := store.Get("infile")
	require.NotNil(t, p, "proc 'infile' not found")

	for _, content := range []string{"hello file", "line1\nline2\n", "unicode: привет"} {
		var buf bytes.Buffer
		require.NoError(t, p.Exec(ctx, &buf, content))
		assert.Equal(t, content, buf.String())
	}
}

// ─── shell: copyfile (file input → file output) ───────────────────────────────

// TestStoreCopyFile verifies that a param with type:file (input) combined with
// output type:file results in the content being copied correctly.
func TestStoreCopyFile(t *testing.T) {
	store := testStore(t)
	ctx := testCtx(t)

	p := store.Get("copyfile")
	require.NotNil(t, p, "proc 'copyfile' not found")

	for _, content := range []string{"copy test", "binary \x00\x01\x02 data"} {
		var buf bytes.Buffer
		require.NoError(t, p.Exec(ctx, &buf, content))
		assert.Equal(t, content, buf.String())
	}
}

// ─── shell: xfile (temp-file output) ─────────────────────────────────────────

func TestStoreXFile(t *testing.T) {
	store := testStore(t)
	ctx := testCtx(t)

	require.NotNil(t, store.Get("xfile"), "proc 'xfile' not found")

	for i := 0; i < 5; i++ {
		var buf bytes.Buffer
		require.NoError(t, store.Exec(ctx, "xfile", &buf, "testdata"))
		assert.Equal(t, "testdata", strings.TrimSpace(buf.String()))
	}
}

// ─── store: Get / Register ────────────────────────────────────────────────────

func TestStoreGetMissing(t *testing.T) {
	store := NewStore()
	assert.Nil(t, store.Get("nonexistent"))
	err := store.Exec(context.Background(), "nonexistent", nil)
	assert.ErrorIs(t, err, ErrProcNotFound)
}

func TestStoreRegister(t *testing.T) {
	base := testStore(t)
	ctx := testCtx(t)

	extra := NewStore()
	catProc := base.Get("cat")
	require.NotNil(t, catProc)
	extra.Register(catProc)

	var buf bytes.Buffer
	require.NoError(t, extra.Exec(ctx, "cat", &buf, "registered"))
	assert.Equal(t, "registered", buf.String())
}

// ─── docker: echo text ───────────────────────────────────────────────────────

func TestStoreDockerEcho(t *testing.T) {
	requireDocker(t)
	store := testStore(t)
	ctx := testCtx(t)

	p := store.Get("dockerecho")
	require.NotNil(t, p, "proc 'dockerecho' not found")

	var buf bytes.Buffer
	require.NoError(t, p.Exec(ctx, &buf, "hello"))
	assert.Equal(t, "hello", strings.TrimSpace(buf.String()))
}

// ─── docker: JSON output ─────────────────────────────────────────────────────

func TestStoreDockerJSON(t *testing.T) {
	requireDocker(t)
	store := testStore(t)
	ctx := testCtx(t)

	p := store.Get("dockerjson")
	require.NotNil(t, p, "proc 'dockerjson' not found")

	var result struct {
		Value string `json:"value"`
	}
	require.NoError(t, p.Exec(ctx, &result, "world"))
	assert.Equal(t, "world", result.Value)
}

// ─── shell: runecho (default "run.sh" script name) ───────────────────────────

// TestStoreRunEcho verifies that the loader finds a default "run.*" script when
// no command is specified and no proc-named script exists in the directory.
func TestStoreRunEcho(t *testing.T) {
	store := testStore(t)
	ctx := testCtx(t)

	p := store.Get("runecho")
	require.NotNil(t, p, "proc 'runecho' not found — run.sh fallback may be broken")

	var buf bytes.Buffer
	require.NoError(t, p.Exec(ctx, &buf, "hello from run.sh"))
	assert.Equal(t, "hello from run.sh", buf.String())
}

// ─── shell: echo (run: array form, single named string param) ────────────────

// TestStoreEcho verifies the run:[printf,…,"{{msg}}"] array form.
// Macro values are shell-quoted automatically so spaces are preserved.
func TestStoreEcho(t *testing.T) {
	store := testStore(t)
	ctx := testCtx(t)

	p := store.Get("echo")
	require.NotNil(t, p, "proc 'echo' not found")

	for _, msg := range []string{"hello", "hello world", "unicode: привет"} {
		var buf bytes.Buffer
		require.NoError(t, p.Exec(ctx, &buf, msg))
		assert.Equal(t, msg, buf.String(), "msg=%q", msg)
	}
}

// ─── shell: scriptecho (run: string form, user-controlled quoting) ────────────

// TestStoreScriptEcho verifies that a run:"…" string manifest (IsScript=true)
// passes macro values without extra shell quoting, so the user's own quotes
// ("{{msg}}") handle spacing correctly.
func TestStoreScriptEcho(t *testing.T) {
	store := testStore(t)
	ctx := testCtx(t)

	p := store.Get("scriptecho")
	require.NotNil(t, p, "proc 'scriptecho' not found")

	for _, msg := range []string{"hello", "hello world", "unicode: привет"} {
		var buf bytes.Buffer
		require.NoError(t, p.Exec(ctx, &buf, msg))
		assert.Equal(t, msg, buf.String(), "msg=%q", msg)
	}
}

// ─── shell: multirun (run: | multiline block) ────────────────────────────────

// TestStoreMultiRun verifies that a multi-line YAML run: | block is executed as
// a single bash script.  The {{prefix}} macro is substituted without extra
// quoting (script mode) and stdin is read by the script itself.
func TestStoreMultiRun(t *testing.T) {
	store := testStore(t)
	ctx := testCtx(t)

	p := store.Get("multirun")
	require.NotNil(t, p, "proc 'multirun' not found")

	cases := []struct {
		prefix string
		data   string
		want   string
	}{
		{"tag", "hello", "tag:hello"},
		{"out", "world", "out:world"},
		{"x", "foo bar", "x:foo bar"},
	}
	for _, tc := range cases {
		var buf bytes.Buffer
		require.NoError(t, p.Exec(ctx, &buf, tc.prefix, tc.data))
		assert.Equal(t, tc.want, buf.String(), "prefix=%q data=%q", tc.prefix, tc.data)
	}
}

// ─── shell: greet (run: array form, two named params) ────────────────────────

// TestStoreGreet verifies that a proc with two named string params maps values
// correctly to {{first}} and {{last}} macros by position.
func TestStoreGreet(t *testing.T) {
	store := testStore(t)
	ctx := testCtx(t)

	p := store.Get("greet")
	require.NotNil(t, p, "proc 'greet' not found")

	cases := []struct{ first, last, want string }{
		{"Alice", "Smith", "Alice Smith"},
		{"Bob", "Jones", "Bob Jones"},
	}
	for _, tc := range cases {
		var buf bytes.Buffer
		require.NoError(t, p.Exec(ctx, &buf, tc.first, tc.last))
		assert.Equal(t, tc.want, strings.TrimSpace(buf.String()))
	}
}

// ─── docker: stream (persistent container, mode: stream) ─────────────────────

// TestStoreDockerStream verifies the Docker streaming mode: a single Alpine
// container is kept alive (retain_container: true) and processes multiple
// requests via stdin/stdout using an awk loop.
func TestStoreDockerStream(t *testing.T) {
	requireDocker(t)
	store := testStore(t)
	ctx := testCtx(t)

	p := store.Get("dockerstream")
	require.NotNil(t, p, "proc 'dockerstream' not found")

	for _, word := range []string{"hello", "world", "three"} {
		var buf bytes.Buffer
		require.NoError(t, p.Exec(ctx, &buf, word))
		assert.Equal(t, "ack:"+word, buf.String(), "word=%q", word)
	}
}

// ─── docker: repeat calls (stateless) ────────────────────────────────────────

func TestStoreDockerRepeat(t *testing.T) {
	requireDocker(t)
	store := testStore(t)
	ctx := testCtx(t)

	p := store.Get("dockerecho")
	require.NotNil(t, p)

	for _, msg := range []string{"first", "second", "third"} {
		var buf bytes.Buffer
		require.NoError(t, p.Exec(ctx, &buf, msg))
		assert.Equal(t, msg, strings.TrimSpace(buf.String()))
	}
}
