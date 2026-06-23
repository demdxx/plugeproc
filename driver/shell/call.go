//go:build !windows

package shell

import (
	"bytes"
	"context"
	"os/exec"
	"strings"

	"github.com/demdxx/plugeproc/driver"
	"github.com/pkg/errors"
)

var ErrCallExecute = errors.New("shell.call")

// CallDriver executes a one-shot shell command via `bash -c`.
type CallDriver struct {
	commandBase   string
	commandPrefix []string
	command       []string
	args          []string
	// scriptMode disables shell quoting of macro values so that user-written
	// bash expressions (run: "echo '{{msg}}'") work as intended.
	scriptMode bool
}

// NewCallDriver creates a driver that runs the command once per Exec call.
// Pass WithScriptMode() when the command is a user-written shell expression
// (from run: "..." or run: | multiline) to disable extra macro quoting.
func NewCallDriver(command, args []string, opts ...Option) *CallDriver {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	return &CallDriver{
		commandBase:   "/usr/bin/env",
		commandPrefix: []string{"bash", "-c"},
		command:       command,
		args:          args,
		scriptMode:    o.scriptMode,
	}
}

func (d *CallDriver) String() string {
	return "shell.call:" + strings.TrimSpace(strings.Join(d.command, " ")+" "+strings.Join(d.args, " "))
}

// Exec runs the command, wires stdin/stdout, and waits for completion.
func (d *CallDriver) Exec(ctx context.Context, params []*driver.Param, out *driver.Output) error {
	q, esc := d.quoting()
	cmdPrepared, err := driver.Params(params).PrepareMacros(q, esc, d.command...)
	if err != nil {
		return err
	}
	argsPrepared, err := driver.Params(params).PrepareMacros(q, esc, d.args...)
	if err != nil {
		return err
	}

	args := make([]string, 0, len(d.commandPrefix)+1)
	args = append(args, d.commandPrefix...)
	args = append(args, strings.Join(cmdPrepared, " ")+" "+strings.Join(argsPrepared, " "))

	cmd := exec.CommandContext(ctx, d.commandBase, args...)
	cmd.Stderr = &bytes.Buffer{}

	if input, err := driver.Params(params).InputStream(false); err != nil {
		return err
	} else {
		cmd.Stdin = input
	}
	if output, err := out.Target(); err != nil {
		return err
	} else {
		cmd.Stdout = output
	}

	if err := cmd.Run(); err != nil {
		text := cmd.Stderr.(*bytes.Buffer).String()
		return errors.Wrap(ErrCallExecute, err.Error()+" : "+text)
	}
	return nil
}

// Close is a no-op for stateless call drivers.
func (d *CallDriver) Close() error { return nil }

// quoting returns the quote/escape symbols for PrepareMacros.
// Script mode: plain substitution (no extra quoting) — the user controls quoting.
// Normal mode: wrap each value in double-quotes for shell safety.
func (d *CallDriver) quoting() (q, esc string) {
	if d.scriptMode {
		return "", ""
	}
	return `"`, `\`
}
