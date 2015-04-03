package proc

import (
	"errors"
	"github.com/golang/glog"
	"os"
	"sync"
)

type Process struct {
	proc      *os.Process
	procAttr  *os.ProcAttr
	procState *os.ProcessState
	prog      string
	argv      []string
	mu        sync.Mutex
	started   bool
}

func New(p string, pA *os.ProcAttr, argv ...string) *Process {
	return &Process{prog: p, procAttr: pA, argv: argv}
}

func (p *Process) Prog() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.prog
}

func (p *Process) Pid() (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.proc != nil {
		glog.V(3).Infoln("Getting Pid for process: %d", p.proc.Pid)
		return p.proc.Pid, nil
	}
	glog.Errorln("Attempted to get Pid for unstarted process: %v", p)
	return 0, errors.New("The Process is not yet started")
}

func (p *Process) Start() (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.started {
		return p.Pid()
	}
	if p.argv == nil {
		p.argv = []string{}
	}
	wd, _ := os.Getwd()
	if p.procAttr == nil {
		p.procAttr = &os.ProcAttr{
			Dir: wd,
			Files: []*os.File{
				os.Stdin,
				os.Stdout,
				os.Stderr,
			},
		}
	}
	if glog.V(1) {
		glog.Infof("Starting process: %s\n", p.prog)
	}
	proc, err := os.StartProcess(p.prog,
		append([]string{p.prog}, p.argv...), p.procAttr)
	if err != nil {
		glog.Errorf("Failed to start proc: %s error: %v", p.prog, err)
		return 0, err
	}
	p.proc = proc
	return p.proc.Pid, nil
}

func (p *Process) HasProc() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.proc != nil
}

func (p *Process) Signal(sig os.Signal) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.proc != nil {
		if glog.V(1) {
			glog.Infof("Signaling PID: %d with signal %v\n", p.proc.Pid, sig)
		}
		return p.proc.Signal(sig)
	}
	return errors.New("Cannot Signal a process that hasn't started)")
}

func (p *Process) Kill() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.proc != nil {
		if glog.V(1) {
			glog.Infof("Killing PID: %d", p.proc.Pid)
		}
		return p.proc.Kill()
	}
	return errors.New("Cannot Kill a process that hasn't started)")
}

func (p *Process) Wait() chan error {
	p.mu.Lock()
	defer p.mu.Unlock()
	done := make(chan error, 1)
	go func() {
		p.mu.Lock()
		defer p.mu.Unlock()
		if glog.V(1) {
			glog.Infof("Waiting on PID: %d", p.proc.Pid)
		}
		state, err := p.proc.Wait()
		if err == nil {
			done <- err
		}
		p.procState = state
		done <- nil
	}()
	return done
}

func (p *Process) GetWaitStatus() *os.ProcessState {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.procState == nil {
		return nil
	}

	ps := *p.procState
	return &ps

}
