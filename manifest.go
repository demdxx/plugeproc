package plugeproc

import "github.com/demdxx/plugeproc/manifest"

// Type aliases so callers only need to import the root package.
type (
	Manifest   = manifest.Manifest
	ParamDef   = manifest.ParamDef
	OutputDef  = manifest.OutputDef
	DockerConf = manifest.DockerConf
	Env        = manifest.Env
	CommandArg = manifest.CommandArg
)

// Driver constants (preferred names).
const (
	DriverShell    = manifest.DriverShell
	DriverExec     = manifest.DriverExec
	DriverDocker   = manifest.DriverDocker
	DriverGoplugin = manifest.DriverGoplugin
)

// Mode constants.
const (
	ModeCall   = manifest.ModeCall
	ModeStream = manifest.ModeStream
)

// Legacy type/interface constants — kept for any code still referencing them.
const (
	TypeShell    = manifest.TypeShell
	TypeExec     = manifest.TypeExec
	TypeDocker   = manifest.TypeDocker
	TypeGoplugin = manifest.TypeGoplugin
	IfaceDefault = manifest.IfaceDefault
	IfaceStream  = manifest.IfaceStream
)
