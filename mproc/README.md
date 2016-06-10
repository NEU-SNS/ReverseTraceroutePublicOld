# mproc
--
    import "github.com/NEU-SNS/ReverseTraceroute/mproc"

Package mproc is a simple process manager

## Usage

#### type FailFunc

```go
type FailFunc func(err error, ps *os.ProcessState, p *proc.Process) bool
```

FailFunc is a function that is called when a process dies

#### type MProc

```go
type MProc interface {
	ManageProcess(p *proc.Process, ka bool, retry uint, f FailFunc) (uint32, error)
	KillAll()
	IntAll()
	EndKeepAlive(id uint32) error
	SignalProc(id uint32, sig os.Signal) error
	WaitProc(id uint32) chan error
	GetProc(id uint32) *proc.Process
	KillProc(id uint32) error
}
```

MProc is a basic process manager

#### func  New

```go
func New() MProc
```
New creates a new MProc
