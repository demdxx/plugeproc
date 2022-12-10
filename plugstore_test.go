//go:build !windows
// +build !windows

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

	"github.com/stretchr/testify/assert"

	"github.com/demdxx/plugeproc/loader/fs"
)

func TestPlugstore(t *testing.T) {
	var (
		buf               bytes.Buffer
		_, fileName, _, _ = runtime.Caller(0)
		pwd               = filepath.Dir(fileName)
		ctx, cancel       = context.WithTimeout(context.Background(), time.Second)
		store, err        = NewPlugstoreFromLoader(ctx, fs.New(filepath.Join(pwd, "tests")))
	)
	defer cancel()
	if !assert.NoError(t, err) {
		return
	}
	for _, procName := range []string{"cat", "proc"} {
		proc := store.Get(procName)
		if !assert.NotNil(t, proc) {
			return
		}
		for i := 0; i < 10; i++ {
			buf.Reset()
			err = proc.Exec(&buf, "test")
			if assert.NoError(t, err) {
				if procName == "cat" {
					if !assert.Equal(t, "test", buf.String()) {
						break
					}
				} else {
					var res = struct {
						Input string `json:"input"`
					}{}
					err = json.NewDecoder(&buf).Decode(&res)
					if !assert.NoError(t, err) || !assert.Equal(t, "test", res.Input) {
						break
					}
				}
			} else {
				break
			}
		}
	}
}

func TestPlugstoreStream(t *testing.T) {
	var (
		res struct {
			Input  int `json:"input"`
			Output int `json:"output"`
		}
		_, fileName, _, _ = runtime.Caller(0)
		pwd               = filepath.Dir(fileName)
		ctx, cancel       = context.WithTimeout(context.Background(), time.Second)
		store, err        = NewPlugstoreFromLoader(ctx, fs.New(filepath.Join(pwd, "tests")))
	)
	defer cancel()
	if !assert.NoError(t, err) {
		return
	}
	proc := store.Get("stream")
	if !assert.NotNil(t, proc) {
		return
	}
	for i := 0; i < 10; i++ {
		err = proc.Exec(&res, struct {
			V int `json:"v"`
		}{V: i})
		if !assert.NoError(t, err) || !assert.Equal(t, i*2, res.Output) {
			break
		}
	}
}

func TestPlugstoreXFile(t *testing.T) {
	var (
		buf               bytes.Buffer
		_, fileName, _, _ = runtime.Caller(0)
		pwd               = filepath.Dir(fileName)
		ctx, cancel       = context.WithTimeout(context.Background(), time.Second)
		store, err        = NewPlugstoreFromLoader(ctx, fs.New(filepath.Join(pwd, "tests")))
	)
	defer cancel()
	if !assert.NoError(t, err) {
		return
	}
	proc := store.Get("xfile")
	if !assert.NotNil(t, proc) {
		return
	}
	for i := 0; i < 10; i++ {
		buf.Reset()
		err = proc.Exec(&buf, "test")
		if !assert.NoError(t, err) || !assert.Equal(t, "test", strings.TrimSpace(buf.String())) {
			break
		}
	}
}
