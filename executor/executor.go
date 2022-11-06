package executor

type Executor interface {
	Exec(params []*Param, out *Output) error
	Close() error
}
