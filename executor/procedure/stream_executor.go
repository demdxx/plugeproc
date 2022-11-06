package procedure

import (
	"bufio"
	"context"
	"encoding/binary"
	"io"
	"os/exec"
	"sync"

	"github.com/demdxx/plugeproc/executor"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var ErrInvalidResponse = errors.New(`invalid response`)

type StreamExecutor struct {
	mx  sync.Mutex
	ctx context.Context

	command string
	args    []string
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  *bufio.Reader
	stderr  io.ReadCloser
	closer  massCloser
}

func NewStreamExecutor(ctx context.Context, command string, args []string) *StreamExecutor {
	return &StreamExecutor{ctx: ctx, command: command, args: args}
}

func (se *StreamExecutor) Exec(params []*executor.Param, out *executor.Output) error {
	if se.cmd == nil {
		if err := se.establish(); err != nil {
			return err
		}
	}
	// Prepare stream request
	input, err := executor.Params(params).InputStream(true)
	if err != nil {
		_ = se.Close()
		return err
	}
	// Send request into the extension
	if _, err = io.Copy(se.stdin, input); err != nil {
		_ = se.Close()
		return err
	}
	// Receive response
	// NOTE: In stream mode we can`t
	// NOTE: read response without knowing the bourthers
	// NOTE: in other case it will way response infinitely
	target, err := out.Target()
	if err != nil {
		_ = se.Close()
		return err
	}

	if out.Type == "binary" {
		bsize := make([]byte, 4)
		_, err = se.stdout.Read(bsize)
		if err == nil {
			rsize := binary.LittleEndian.Uint32(bsize)
			_, err = io.CopyN(target, se.stdout, int64(rsize))
		}
		if err != nil {
			_ = se.Close() // Kill the process to reopen again for the next request
			return err
		}
	} else {
		line, _, err := se.stdout.ReadLine()
		if err != nil {
			_ = se.Close() // Kill the process to reopen again for the next request
			return err
		}
		if _, err = target.Write(line); err != nil {
			return err
		}
	}
	return err
}

func (se *StreamExecutor) establish() error {
	if err := se.Close(); err != nil {
		return err
	}
	se.mx.Lock()
	defer se.mx.Unlock()

	cmd := exec.CommandContext(se.ctx, "/usr/bin/env",
		append([]string{"bash", "-c", se.command}, se.args...)...)
	// Prepare streams
	in, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	serr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	// Run command init
	if err := cmd.Start(); err != nil {
		zap.L().Error("start error", zap.String("command", se.command), zap.Error(err))
	}
	zap.L().Info("exec command", zap.String("command", se.command))
	// Stderr loop
	go func() {
		sreader := bufio.NewReader(serr)
		for {
			if line, _, err := sreader.ReadLine(); err != nil {
				zap.L().Error("error", zap.String("command", se.command), zap.Error(err))
				break
			} else {
				zap.L().Info("stderr> " + string(line))
			}
		}
	}()
	// Execution processing
	go func() {
		if err := cmd.Wait(); err != nil {
			zap.L().Error("execute error", zap.String("command", se.command), zap.Error(err))
		}
	}()
	// Save std stream pointers
	se.stdin = in
	se.stdout = bufio.NewReader(out)
	se.stderr = serr
	se.closer = massCloser{in, out, serr}
	return nil
}

func (se *StreamExecutor) Close() error {
	se.mx.Lock()
	defer se.mx.Unlock()
	if se.cmd == nil {
		return nil
	}
	_ = se.closer.Close()
	se.closer = se.closer[:0]
	err := se.cmd.Process.Kill()
	se.cmd = nil
	return err
}
