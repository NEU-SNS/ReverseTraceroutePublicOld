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
	restart   bool
	mu        sync.Mutex
}

func (p *Process) Pid() (int, error) {
	defer p.mu.Unlock()
	p.mu.Lock()

	if p.proc != nil {
		glog.V(3).Infoln("Getting Pid for process: %d", p.proc.Pid)
		return p.proc.Pid, nil
	}
	if glog.V(2) {
		glog.Errorln("Attempted to get Pid for unstarted process: %v", p)
	}
	return 0, errors.New("The Process is not yet started")
}

func (p *Process) Start() (int, error) {

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
		glog.Errorf("")
		return 0, err
	}
	p.proc = proc
	return p.proc.Pid, nil
}

func (p *Process) HasProc() bool {
	defer p.mu.Unlock()
	p.mu.Lock()
	return p.proc != nil
}

func (p *Process) Signal(sig os.Signal) error {
	defer p.mu.Unlock()
	p.mu.Lock()
	if p.proc != nil {
		return p.proc.Signal(sig)
	}
	return errors.New("Cannot Signal a process that hasn't started)")
}

func (p *Process) Kill() error {
	defer p.mu.Unlock()
	p.mu.Lock()
	if p.proc != nil {
		return p.proc.Kill()
	}
	return errors.New("Cannot Kill a process that hasn't started)")
}

func (p *Process) Wait() chan error {
	defer p.mu.Unlock()
	p.mu.Lock()
	done := make(chan error, 1)
	go func() {
		defer p.mu.Unlock()
		p.mu.Lock()
		state, err := p.proc.Wait()
		if err == nil {
			done <- err
		}
		p.procState = state
		done <- nil
	}()
	return done
}

/*
func (p *Process) KeepAlive() {
	if !p.restart {
		return
	}
	psChan := make(chan *os.ProcessState, 1)
	errChan := make(chan error, 1)
	go func() {
		state, err := p.proc.Wait()
		if err != nil {
			errChan <- err
			return
		}
		psChan <- state
	}()

	select {

	case ps := <-psChan:
		{
			p.procState = ps
			_, errs := p.Start()
			if errs == nil && p.restart {
				//p.keepAlive()
			}
		}

	case ec := <-errChan:
		{
			log.Println("Error on wait: %v", ec)
			return
		}
	}
}
*/
func (p *Process) GetWaitStatus() *os.ProcessState {
	defer p.mu.Unlock()
	p.mu.Lock()

	if p.procState == nil {
		return nil
	}

	ps := *p.procState
	return &ps

}
