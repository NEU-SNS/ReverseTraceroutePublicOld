package mproc

import (
	"errors"
	"fmt"
	"github.com/NEU-SNS/ReverseTraceroute/mproc/proc"
	"github.com/golang/glog"
	"os"
	"sync"
)

type mProc struct {
	mu           sync.Mutex
	managedProcs map[int]*managedP
}

type managedP struct {
	p         *proc.Process
	mu        sync.Mutex
	keepAlive bool
}

//New: Return a pointer to a newly created mProc.
func New() *mProc {
	return &mProc{managedProcs: make(map[int]*managedP, 10)}

}

func create(p *proc.Process, keepAlive bool) *managedP {
	return &managedP{p: p, keepAlive: keepAlive}
}

var id = 0

//ManageProcess: Add a process to the manager and start it.
//The function returns the id, error
func (mp *mProc) ManageProcess(p *proc.Process, ka bool) (int, error) {
	if p == nil {
		return 0, errors.New("ManageProcess Argument nil: p")
	}
	defer mp.mu.Unlock()
	mp.mu.Lock()

	_, err := p.Start()
	if err != nil {
		return 0, err
	}
	manp := create(p, ka)
	mp.managedProcs[id] = manp
	if ka {
		mp.keepAlive(id)
	}
	rid := id
	id = id + 1
	return rid, err
}

func (mp *mProc) keepAlive(id int) {
	go func() {
		p := mp.getMp(id)
		err := <-p.p.Wait()
		pid, e := p.p.Pid()
		if e != nil {
			glog.Errorf("mProc Failed to get PID")
			return
		}
		if err == nil {
			ps := p.p.GetWaitStatus()
			glog.Infof("KeepAlive Proc: %s, PID: %d stopped, status: %v",
				p.p.Prog(), pid, ps)

			mp.mu.Lock()
			defer mp.mu.Unlock()
			if p.keepAlive {
				pid, err := p.p.Start()
				if err != nil {
					glog.Error("Failed to restart process in keepAlive")
					return
				}
				glog.Infof("Restarted process: %s, PID: %d", p.p.Prog(), pid)
				mp.keepAlive(id)
			}
			return
		}
		glog.Errorf("mProc Failed to wait on PID: %d cannot restart", pid)
	}()
}

// Stop keep alive
func (mp *mProc) EndKeepAlive(id int) error {
	mp.mu.Lock()
	p := mp.getMp(id)
	if p == nil {
		defer mp.mu.Unlock()
		return errors.New(
			fmt.Sprintf("Error: Process with ID: %d does not exist", id))
	}
	pid, _ := p.p.Pid()
	if glog.V(2) {
		glog.Infof("Ending keep alive on PID: %d", pid)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	mp.mu.Unlock()
	p.keepAlive = false
	return nil
}

//getMp: Get a managed proc, for internal use only.
func (mp *mProc) getMp(id int) *managedP {
	defer mp.mu.Unlock()
	mp.mu.Lock()
	return mp.managedProcs[id]
}

//SignalProc: Signal a process with the given pid and signal.
func (mp *mProc) SignalProc(id int, sig os.Signal) error {
	pro := mp.GetProc(id)
	if pro == nil {
		return errors.New(
			fmt.Sprintf("Error: No proc with Id: %d", id))
	}
	return pro.Signal(sig)
}

//WaitProc: Wait on a process with the given pid.
func (mp *mProc) WaitProc(id int) chan error {
	proc := mp.GetProc(id)
	return proc.Wait()
}

//GetProc: Get a Process by pid.
func (mp *mProc) GetProc(id int) *proc.Process {
	defer mp.mu.Unlock()
	mp.mu.Lock()
	p := mp.managedProcs[id]
	if p != nil {
		return p.p
	}
	return nil
}

func (mp *mProc) KillProc(id int) error {
	proc := mp.GetProc(id)
	if proc == nil {
		return errors.New(
			fmt.Sprintf("Error: No proc with Id: %d", id))
	}
	return proc.Kill()
}
