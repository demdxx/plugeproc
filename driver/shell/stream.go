//go:build !windows

package shell

import (
	"bufio"
	"context"
	"io"
	"os/exec"
	"strings"
	"sync"

	"github.com/demdxx/plugeproc/driver"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var ErrInvalidResponse = errors.New("shell.stream: invalid response")

// StreamDriver keeps a persistent shell process alive across multiple Exec calls.
type StreamDriver struct {
	mx sync.Mutex

	commandBase   string
	commandPrefix []string
	command       []string
	args          []string
	scriptMode    bool

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	stderr io.ReadCloser
	closer massCloser
}

// NewStreamDriver creates a driver backed by a long-lived bash process.
// Pass WithScriptMode() when the command is a user-written shell expression.
func NewStreamDriver(command, args []string, opts ...Option) *StreamDriver {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	return &StreamDriver{
		commandBase:   "/usr/bin/env",
		commandPrefix: []string{"bash", "-c"},
		command:       command,
		args:          args,
		scriptMode:    o.scriptMode,
	}
}

func (d *StreamDriver) String() string {
	return "shell.stream:" + strings.TrimSpace(strings.Join(d.command, " ")+" "+strings.Join(d.args, " "))
}

// Exec sends params to the persistent process and reads back one framed response.
func (d *StreamDriver) Exec(ctx context.Context, params []*driver.Param, out *driver.Output) error {
	if d.cmd == nil {
		if err := d.establish(ctx); err != nil {
			return err
		}
	}
	input, err := driver.Params(params).InputStream(true)
	if err != nil {
		_ = d.Close()
		return err
	}
	if _, err = io.Copy(d.stdin, input); err != nil {
		_ = d.Close()
		return err
	}
	if err = out.ProcessStreamResponse(d.stdout); err != nil {
		_ = d.Close()
	}
	return err
}

// establish spawns the persistent subprocess.
func (d *StreamDriver) establish(ctx context.Context) error {
	if err := d.Close(); err != nil {
		return err
	}
	d.mx.Lock()
	defer d.mx.Unlock()

	args := make([]string, 0, len(d.commandPrefix)+len(d.command)+len(d.args))
	args = append(args, d.commandPrefix...)
	args = append(args, d.command...)
	args = append(args, d.args...)

	cmd := exec.CommandContext(ctx, d.commandBase, args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		zap.L().Error("stream driver start error", zap.Strings("command", d.command), zap.Error(err))
		return err
	}
	zap.L().Info("stream driver started", zap.Strings("command", d.command))

	go func() {
		reader := bufio.NewReader(stderr)
		for {
			line, _, err := reader.ReadLine()
			if err != nil {
				zap.L().Error("stream driver stderr closed", zap.Strings("command", d.command), zap.Error(err))
				break
			}
			zap.L().Warn("stream driver stderr", zap.String("line", string(line)))
		}
	}()
	go func() {
		if err := cmd.Wait(); err != nil {
			zap.L().Error("stream driver exited", zap.Strings("command", d.command), zap.Error(err))
		}
	}()

	d.cmd = cmd
	d.stdin = stdin
	d.stdout = bufio.NewReader(stdout)
	d.stderr = stderr
	d.closer = massCloser{stdin, stdout, stderr}
	return nil
}

// Close terminates the persistent subprocess.
func (d *StreamDriver) Close() error {
	d.mx.Lock()
	defer d.mx.Unlock()
	if d.cmd == nil {
		return nil
	}
	_ = d.closer.Close()
	d.closer = d.closer[:0]
	err := d.cmd.Process.Kill()
	d.cmd = nil
	return err
}
