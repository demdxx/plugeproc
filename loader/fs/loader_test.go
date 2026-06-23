package fs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/demdxx/plugeproc/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeFile creates a file with the given contents and, optionally, makes it executable.
func writeFile(t *testing.T, path, content string, executable bool) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	if executable {
		require.NoError(t, os.Chmod(path, 0o755))
	}
}

// TestFindExecutableByProcName checks that a script matching the proc name is found first.
func TestFindExecutableByProcName(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "myprog.sh"), "#!/bin/sh\ncat", true)
	writeFile(t, filepath.Join(dir, "run.sh"), "#!/bin/sh\necho run", true)

	l := New(dir)
	got, err := l.findExecutable(filepath.Join(dir, ".eproc.json"), "myprog")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "myprog.sh"), got)
}

// TestFindExecutableDefaultRun checks that run.sh is used as a fallback when no
// proc-named script exists.
func TestFindExecutableDefaultRun(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "run.sh"), "#!/bin/sh\ncat", true)

	l := New(dir)
	got, err := l.findExecutable(filepath.Join(dir, ".eproc.json"), "noname")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "run.sh"), got)
}

// TestFindExecutableDefaultExec checks that exec.sh is found when run.sh is absent.
func TestFindExecutableDefaultExec(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "exec.sh"), "#!/bin/sh\ncat", true)

	l := New(dir)
	got, err := l.findExecutable(filepath.Join(dir, ".eproc.json"), "noname")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "exec.sh"), got)
}

// TestFindExecutableNotFound verifies the error when nothing matches.
func TestFindExecutableNotFound(t *testing.T) {
	dir := t.TempDir()
	l := New(dir)
	_, err := l.findExecutable(filepath.Join(dir, ".eproc.json"), "noname")
	assert.Error(t, err)
}

// TestLoadNormalizeNewFormat verifies that manifests using the new field names load correctly.
// The manifest explicitly sets its name so the loader doesn't derive it from the temp dir.
func TestLoadNormalizeNewFormat(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "myprog.sh"), "#!/bin/sh\ncat", true)
	writeFile(t, filepath.Join(dir, ".eproc.json"),
		`{"name":"myprog","driver":"exec","params":[{"name":"data","type":"binary","stdin":true}],"output":{"type":"binary"}}`,
		false)

	l := New(dir)
	procs, err := l.Load()
	require.NoError(t, err)
	require.Len(t, procs, 1)

	m := procs[0]
	assert.Equal(t, "myprog", m.Name)
	assert.Equal(t, manifest.DriverShell, m.Driver) // exec → discovers script → promoted to shell
	assert.Equal(t, manifest.ModeCall, m.Mode)
	require.Len(t, m.Params, 1)
	assert.True(t, m.Params[0].Stdin)
}

// TestLoadNormalizeLegacyFormat verifies that old-style manifests (type/interface/is_input)
// are transparently promoted to the new field names.
func TestLoadNormalizeLegacyFormat(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".eproc.json"),
		`{"name":"oldproc","type":"shell","interface":"stream","command":"/bin/cat","params":[{"name":"data","type":"binary","is_input":true}],"output":{"type":"binary"}}`,
		false)

	l := New(dir)
	procs, err := l.Load()
	require.NoError(t, err)
	require.Len(t, procs, 1)

	m := procs[0]
	assert.Equal(t, manifest.DriverShell, m.Driver)
	assert.Equal(t, manifest.ModeStream, m.Mode)
	require.Len(t, m.Params, 1)
	assert.True(t, m.Params[0].Stdin, "is_input should be promoted to stdin")
}

// TestLoadNormalizeFileOutput verifies that is_tmp_file is preserved and the default
// macro name "out" is applied when output.name is empty.
func TestLoadNormalizeFileOutput(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".eproc.json"),
		`{"name":"writer","driver":"shell","command":"/bin/sh","args":["-c","echo $1 > $2"],"params":[{"name":"data","type":"string"}],"output":{"type":"file"}}`,
		false)

	l := New(dir)
	procs, err := l.Load()
	require.NoError(t, err)
	require.Len(t, procs, 1)

	m := procs[0]
	assert.Equal(t, manifest.OutputFileType, m.Output.Type)
	assert.Equal(t, manifest.DefaultOutputMacro, m.Output.Name) // "out" filled by Normalize
}

// TestLoadNormalizeLegacyIsTmpFile verifies old is_tmp_file: true is promoted.
func TestLoadNormalizeLegacyIsTmpFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".eproc.json"),
		`{"name":"writer","driver":"shell","command":"/bin/sh","args":["-c","echo"],"params":[{"name":"data","type":"string"}],"output":{"name":"outfile","is_tmp_file":true}}`,
		false)

	l := New(dir)
	procs, err := l.Load()
	require.NoError(t, err)
	require.Len(t, procs, 1)

	m := procs[0]
	// is_tmp_file promotes to type "file"
	assert.Equal(t, manifest.OutputFileType, m.Output.Type)
	assert.Equal(t, "outfile", m.Output.Name) // explicit name preserved
}
