package manifest

import (
	"encoding/json"
	"strings"

	"gopkg.in/yaml.v3"
)

// Driver constants (new preferred names).
const (
	DriverShell    = "shell"
	DriverExec     = "exec"
	DriverDocker   = "docker"
	DriverGoplugin = "goplugin"

	// Legacy aliases kept for backward compatibility.
	TypeShell    = DriverShell
	TypeExec     = DriverExec
	TypeDocker   = DriverDocker
	TypeGoplugin = DriverGoplugin
)

// Mode constants (replaces the old "interface" field).
const (
	ModeCall   = "call"
	ModeStream = "stream"

	// Legacy aliases.
	IfaceDefault = "default"
	IfaceStream  = ModeStream
)

// OutputFileType is the output type value that signals the command writes to a
// temp file whose path is injected as a {{out}} (or custom-named) macro argument.
const OutputFileType = "file"

// DefaultOutputMacro is the default macro name for file-type outputs.
const DefaultOutputMacro = "out"

// Manifest describes an external procedure loaded from an .eproc file.
type Manifest struct {
	Name    string `json:"name"             yaml:"name"`
	Version string `json:"version,omitempty" yaml:"version,omitempty"`

	// Preferred fields (new format).
	Driver string `json:"driver,omitempty" yaml:"driver,omitempty"`
	Mode   string `json:"mode,omitempty"   yaml:"mode,omitempty"`

	// Legacy fields — still accepted, silently promoted to Driver/Mode on Normalize().
	Type      string `json:"type,omitempty"      yaml:"type,omitempty"`
	Interface string `json:"interface,omitempty" yaml:"interface,omitempty"`

	// Run is the preferred way to specify what to execute.  It accepts:
	//   - an argv array:       run: [echo, -n, "{{msg}}"]
	//   - a single string:     run: echo -n "{{msg}}"
	//   - a multiline script:  run: |
	//                               echo -n "{{msg}}"
	//                               echo "done"
	//
	// When Run is a string or multiline block, macro substitution is performed
	// without shell quoting so the user's own quotes are respected.
	// run: has higher priority than command: when both are present.
	Run RunSpec `json:"run,omitempty" yaml:"run,omitempty"`

	// Command is the legacy argv-list form.  Still accepted for compatibility.
	Command   CommandArg  `json:"command,omitempty"  yaml:"command,omitempty"`
	Args      []string    `json:"args,omitempty"     yaml:"args,omitempty"`
	Params    []ParamDef  `json:"params,omitempty"   yaml:"params,omitempty"`
	Output    OutputDef   `json:"output,omitempty"   yaml:"output,omitempty"`
	Env       Env         `json:"env,omitempty"      yaml:"env,omitempty"`
	Docker    *DockerConf `json:"docker,omitempty"   yaml:"docker,omitempty"`

	// ScriptMode is set by Normalize() when Run was a string/multiline form.
	// Drivers use this to skip extra shell quoting of macro values.
	ScriptMode bool `json:"-" yaml:"-"`

	Directory string `json:"-" yaml:"-"`
}

// Normalize promotes legacy fields to their new names and fills in defaults.
// It is idempotent and safe to call multiple times.
func (m *Manifest) Normalize() {
	// run: → command (run has higher priority)
	if !m.Run.IsEmpty() {
		m.Command = m.Run.ToCommandArg()
		m.ScriptMode = m.Run.IsScript
	}

	// type → driver
	if m.Driver == "" {
		m.Driver = m.Type
	}
	if m.Driver == "" {
		m.Driver = DriverExec
	}

	// interface → mode
	if m.Mode == "" {
		m.Mode = m.Interface
	}
	if m.Mode == "" || m.Mode == IfaceDefault {
		m.Mode = ModeCall
	}

	// Params: is_input → stdin
	for i := range m.Params {
		p := &m.Params[i]
		if !p.Stdin && p.IsInput {
			p.Stdin = true
		}
	}

	// Output: is_tmp_file → type file
	if m.Output.IsTmpFile && m.Output.Type == "" {
		m.Output.Type = OutputFileType
	}
	// When type is file, ensure the macro name is set (default "out").
	if m.Output.Type == OutputFileType && m.Output.Name == "" {
		m.Output.Name = DefaultOutputMacro
	}
}

// ─── RunSpec ─────────────────────────────────────────────────────────────────

// RunSpec holds the value of a `run:` field.  It records whether the original
// source was an argv array (IsScript=false) or a string/multiline block
// (IsScript=true) so callers can choose an appropriate quoting strategy.
type RunSpec struct {
	parts    []string
	IsScript bool
}

// IsEmpty reports whether the spec is unset.
func (r RunSpec) IsEmpty() bool { return len(r.parts) == 0 }

// ToCommandArg returns the parts as a CommandArg.
func (r RunSpec) ToCommandArg() CommandArg { return CommandArg(r.parts) }

// UnmarshalJSON accepts `"script"`, `["cmd","arg"]`, or a JSON block string.
func (r *RunSpec) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		r.parts = []string{s}
		r.IsScript = true
		return nil
	}
	var multi []string
	if err := json.Unmarshal(data, &multi); err != nil {
		return err
	}
	r.parts = multi
	r.IsScript = false
	return nil
}

// UnmarshalYAML accepts a scalar string, a YAML block literal (|), or a sequence.
func (r *RunSpec) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		r.parts = []string{value.Value}
		r.IsScript = true
		return nil
	}
	var multi []string
	if err := value.Decode(&multi); err != nil {
		return err
	}
	r.parts = multi
	r.IsScript = false
	return nil
}

// ─── ParamDef ────────────────────────────────────────────────────────────────

// ParamDef declares one input parameter of a procedure.
type ParamDef struct {
	Name  string `json:"name"            yaml:"name"`
	Type  string `json:"type"            yaml:"type"`
	Stdin bool   `json:"stdin,omitempty" yaml:"stdin,omitempty"`

	// Legacy — promoted to Stdin by Normalize().
	IsInput   bool `json:"is_input,omitempty"   yaml:"is_input,omitempty"`
	IsTmpFile bool `json:"is_tmp_file,omitempty" yaml:"is_tmp_file,omitempty"`
}

// ─── OutputDef ───────────────────────────────────────────────────────────────

// OutputDef declares the output shape of a procedure.
type OutputDef struct {
	// Name is the macro name used in args when Type is "file" (default "out" → {{out}}).
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	// Type is one of: binary, line, json, file.
	// "file" means the command writes output to a temp file whose path is
	// injected via the Name macro.
	Type string `json:"type,omitempty" yaml:"type,omitempty"`

	// Legacy — promoted to Type "file" by Normalize().
	IsTmpFile bool `json:"is_tmp_file,omitempty" yaml:"is_tmp_file,omitempty"`
}

// ─── DockerConf ──────────────────────────────────────────────────────────────

// DockerConf holds Docker-specific settings for a procedure of type "docker".
type DockerConf struct {
	Image           string `json:"image"                       yaml:"image"`
	PullImage       bool   `json:"pull_image,omitempty"        yaml:"pull_image,omitempty"`
	RetainContainer bool   `json:"retain_container,omitempty"  yaml:"retain_container,omitempty"`
	RemoveAfterDone bool   `json:"remove_after_done,omitempty" yaml:"remove_after_done,omitempty"`
	ContainerName   string `json:"container_name,omitempty"    yaml:"container_name,omitempty"`
}

// ─── Env ─────────────────────────────────────────────────────────────────────

// Env is a map of environment variable key-value pairs.
type Env map[string]string

// Values returns the environment as a plain map.
func (e Env) Values() map[string]string { return e }

// ─── CommandArg ──────────────────────────────────────────────────────────────

// CommandArg is a command that may be a single string or a list of strings in
// JSON/YAML.  It is the legacy form; prefer RunSpec / run: in new manifests.
type CommandArg []string

// String returns the command joined by spaces.
func (c CommandArg) String() string {
	if len(c) == 1 {
		return c[0]
	}
	return strings.Join(c, " ")
}

// IsEmpty reports whether the command is unset.
func (c CommandArg) IsEmpty() bool {
	return len(c) == 0 || (len(c) == 1 && c[0] == "")
}

// UnmarshalJSON accepts both `"cmd"` and `["cmd", "arg"]`.
func (c *CommandArg) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*c = CommandArg{single}
		return nil
	}
	var multi []string
	if err := json.Unmarshal(data, &multi); err != nil {
		return err
	}
	*c = multi
	return nil
}

// UnmarshalYAML accepts both a scalar string and a sequence of strings.
func (c *CommandArg) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		*c = CommandArg{value.Value}
		return nil
	}
	var multi []string
	if err := value.Decode(&multi); err != nil {
		return err
	}
	*c = multi
	return nil
}
