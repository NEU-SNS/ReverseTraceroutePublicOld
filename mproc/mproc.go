package mproc

import (
	"errors"
	"fmt"
	"github.com/NEU-SNS/mproc/proc"
	"os"
)

type mProc struct {
	managedProcs map[int]*proc.Process
}

func New() *mProc {
	return &mProc{managedProcs: make(map[int]*proc.Process, 10)}

}

func (mp *mProc) ManageProcess(p *proc.Process) *proc.Process {
	if p.HasProc() {
		mp.addToStarted(p)
	} else {
		p.Start()
	}
	return p
}

func (mp *mProc) addToStarted(p *proc.Process) {
	pid, err := p.Pid()
	if err == nil {
		mp.managedProcs[pid] = p
		return
	}
}

func (mp *mProc) SignalProc(pid int, sig os.Signal) error {
	pro := mp.managedProcs[pid]
	if pro == nil {
		return errors.New(
			fmt.Sprintf("Error: Process with PID: %d does not exist", pid))
	}
	return pro.Signal(sig)
}

func (mp *mProc) WaitProc(pid int) chan error {
	proc := mp.GetProc(pid)
	return proc.Wait()
}

func (mp *mProc) GetProc(pid int) *proc.Process {
	return mp.managedProcs[pid]
}

func (mp *mProc) KillProc(pid int) error {
	proc := mp.managedProcs[pid]
	if proc == nil {
		return errors.New(
			fmt.Sprintf("Error: Process with PID: %d does not exist", pid))
	}
	return proc.Kill()
}
