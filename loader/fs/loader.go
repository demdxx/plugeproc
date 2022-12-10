package fs

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/demdxx/plugeproc/models"
)

var errInvalidExternalProc = errors.New(`executable proc not found`)

type Loader struct {
	directory     string
	procMetaSufix string
}

func New(directory string, procMetaSufixs ...string) *Loader {
	procMetaSufix := ".eproc.json"
	if len(procMetaSufixs) > 0 && procMetaSufixs[0] != "" {
		procMetaSufix = procMetaSufixs[0]
	}
	return &Loader{
		directory:     directory,
		procMetaSufix: procMetaSufix,
	}
}

func (l *Loader) Load() ([]*models.Info, error) {
	procs := make([]*models.Info, 0, 10)
	err := filepath.Walk(l.directory, func(path string, info fs.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() || !strings.HasSuffix(path, l.procMetaSufix) {
			return nil
		}
		procInfo := new(models.Info)
		if data, err := os.ReadFile(path); err != nil {
			return err
		} else if err = json.Unmarshal(data, procInfo); err != nil {
			return err
		}
		if procInfo.Name == "" {
			if info.Name() == l.procMetaSufix {
				procInfo.Name = filepath.Base(filepath.Dir(path))
			} else {
				procInfo.Name = strings.TrimSuffix(filepath.Base(path), l.procMetaSufix)
			}
		}
		if procInfo.Command == "" {
			if procInfo.Type == "exec" || procInfo.Type == "proc" {
				files, err := filepath.Glob(
					filepath.Join(filepath.Dir(path), procInfo.Name+"*"))
				if err != nil {
					return err
				}
				for _, filename := range files {
					curfilename := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
					// command = {{name}}.{{ext}}
					if curfilename == procInfo.Name && isExecutable(filename) {
						procInfo.Command = filename
					}
				}
				if procInfo.Command == "" {
					return errInvalidExternalProc
				}
				procInfo.Type = models.ProgTypeShell
			}
		}
		procInfo.Directory = filepath.Dir(path)
		procs = append(procs, procInfo)
		return nil
	})
	return procs, err
}

func isExecutable(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) || info.IsDir() {
		return false
	}
	return info.Mode()&0101 != 0
}
