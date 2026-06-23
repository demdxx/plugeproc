package fs

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/demdxx/plugeproc/decode"
	"github.com/demdxx/plugeproc/manifest"
	"github.com/pkg/errors"
)

var errExecutableNotFound = errors.New("executable not found for proc")

const defaultProcMetaSuffix = ".eproc"

// defaultScriptNames are tried in order when no command is specified in the manifest
// and the proc-name script is not found.  "run" mirrors the convention used by
// Docker, npm, and similar tools.
var defaultScriptNames = []string{"run", "exec", "main"}

// Loader discovers procedure manifests by walking a directory tree.
type Loader struct {
	directory      string
	procMetaSuffix string
	decoder        decode.Decoder
}

// New creates a filesystem loader for the given directory.
func New(directory string, opts ...Option) *Loader {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	return &Loader{
		directory:      directory,
		procMetaSuffix: o.ProcMetaSuffix(),
		decoder:        o.Decoder(),
	}
}

// Load walks the directory and returns all discovered manifests.
func (l *Loader) Load() ([]*manifest.Manifest, error) {
	var procs []*manifest.Manifest
	err := filepath.Walk(l.directory, func(path string, info fs.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		base := strings.TrimSuffix(path, filepath.Ext(path))
		if !strings.HasSuffix(base, l.procMetaSuffix) {
			return nil
		}
		m, err := l.loadManifest(path)
		if err != nil {
			return err
		}
		// For exec-type procs without an explicit command, discover the script.
		if m.Command.IsEmpty() && m.Driver == manifest.DriverExec {
			cmd, err := l.findExecutable(path, m.Name)
			if err != nil {
				return err
			}
			m.Command = manifest.CommandArg{cmd}
			m.Driver = manifest.DriverShell
		}
		m.Directory = filepath.Dir(path)
		procs = append(procs, m)
		return nil
	})
	return procs, err
}

func (l *Loader) loadManifest(path string) (*manifest.Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	m := new(manifest.Manifest)
	ext := filepath.Ext(path)
	if err = l.decoder.Decode(ext, data, m); err != nil {
		return nil, err
	}
	if m.Name == "" {
		baseName := strings.TrimRight(filepath.Base(path), ext)
		if baseName == l.procMetaSuffix {
			m.Name = filepath.Base(filepath.Dir(path))
		} else {
			m.Name = strings.TrimSuffix(baseName, l.procMetaSuffix)
		}
	}
	// Promote legacy fields (type→driver, interface→mode, is_input→stdin, etc.)
	m.Normalize()
	return m, nil
}

// findExecutable looks for an executable file in the same directory as the
// manifest.  It first tries files whose base name matches the proc name, then
// falls back to the conventional default names (run, exec, main).
func (l *Loader) findExecutable(manifestPath, name string) (string, error) {
	dir := filepath.Dir(manifestPath)

	// 1. Try {proc-name}.* (current behaviour, highest priority).
	if path, ok := findExecInDir(dir, name); ok {
		return path, nil
	}

	// 2. Try conventional default names.
	for _, def := range defaultScriptNames {
		if path, ok := findExecInDir(dir, def); ok {
			return path, nil
		}
	}

	return "", errors.Wrap(errExecutableNotFound, name)
}

// findExecInDir looks for any executable file whose base name (without
// extension) equals name inside dir.
func findExecInDir(dir, name string) (string, bool) {
	matches, err := filepath.Glob(filepath.Join(dir, name+"*"))
	if err != nil {
		return "", false
	}
	for _, filename := range matches {
		base := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
		if base == name && isExecutable(filename) {
			return filename, true
		}
	}
	return "", false
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Mode()&0o101 != 0
}
