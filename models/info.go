package models

const (
	ProgTypeShell    = `shell`
	ProgTypeExec     = `exec`
	ProgTypeGoplugin = `goplugin`
	IfaceDefault     = `default`
	IfaceStream      = `stream`
)

type Info struct {
	Name      string   `json:"name"`
	Version   string   `json:"version"`
	Type      string   `json:"type"`
	Interface string   `json:"interface"`
	Command   string   `json:"command"`
	Args      []string `json:"args"`
	Params    []Param  `json:"params"`
	Output    Output   `json:"output"`
	Directory string   `json:"-"`
}
