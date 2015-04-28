/*
 Copyright (c) 2015, Northeastern University
 All rights reserved.

 Redistribution and use in source and binary forms, with or without
 modification, are permitted provided that the following conditions are met:
     * Redistributions of source code must retain the above copyright
       notice, this list of conditions and the following disclaimer.
     * Redistributions in binary form must reproduce the above copyright
       notice, this list of conditions and the following disclaimer in the
       documentation and/or other materials provided with the distribution.
     * Neither the name of the Northeastern University nor the
       names of its contributors may be used to endorse or promote products
       derived from this software without specific prior written permission.

 THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
 ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 DISCLAIMED. IN NO EVENT SHALL Northeastern University BE LIABLE FOR ANY
 DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/
package mproc

import (
	"errors"
	"fmt"
	"github.com/NEU-SNS/ReverseTraceroute/lib/mproc/proc"
	"github.com/golang/glog"
	"os"
	"sync"
	"time"
)

const (
	DELAY = 2
)

type FailFunc func(err error, ps *os.ProcessState) bool

func noop(err error, ps *os.ProcessState) bool {
	return false
}

var id uint32

type MProc interface {
	ManageProcess(p *proc.Process, ka bool, retry uint, f FailFunc) (uint32, error)
	KillAll()
	EndKeepAlive(id uint32) error
	SignalProc(id uint32, sig os.Signal) error
	WaitProc(id uint32) chan error
	GetProc(id uint32) *proc.Process
	KillProc(id uint32) error
}

type mProc struct {
	mu           sync.Mutex
	managedProcs map[uint32]*managedP
}

type managedP struct {
	p         *proc.Process
	mu        sync.Mutex
	keepAlive bool
	retry     uint
	remRetry  uint
	f         FailFunc
}

func New() MProc {
	return &mProc{managedProcs: make(map[uint32]*managedP, 10)}

}

func create(p *proc.Process, keepAlive bool, retry uint, f FailFunc) *managedP {
	if f == nil {
		f = noop
	}
	return &managedP{p: p, keepAlive: keepAlive, retry: retry, remRetry: retry, f: f}
}

func (mp *mProc) KillAll() {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	for _, v := range mp.managedProcs {
		v.mu.Lock()
		v.p.Kill()
		v.mu.Unlock()
	}
}

//If you want the process to restart indef. just use MaxUint32
func (mp *mProc) ManageProcess(p *proc.Process, ka bool, retry uint, f FailFunc) (uint32, error) {

	if p == nil {
		return 0, errors.New("ManageProcess Argument nil: p")
	}
	defer mp.mu.Unlock()
	mp.mu.Lock()
	glog.Infof("Starting process: %s", p.String())
	_, err := p.Start()
	if err != nil {
		return 0, err
	}
	manp := create(p, ka, retry, f)
	mp.managedProcs[id] = manp
	if ka {
		mp.keepAlive(id)
	}
	rid := id
	id = id + 1
	return rid, err
}

func (mp *mProc) keepAlive(id uint32) {
	go func() {
		p := mp.getMp(id)
		err := <-p.p.Wait()
		exit := p.f(err, p.p.GetWaitStatus())
		if exit {
			p.keepAlive = false
			p.remRetry = 0
			return
		}
		glog.V(1).Infof("Keep Alive just returned from wait")
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
			if p.keepAlive && p.remRetry > 0 {
				<-time.After(DELAY * time.Second)
				pid, err := p.p.Start()
				if err != nil {
					exit := p.f(err, p.p.GetWaitStatus())
					glog.Error("Failed to restart process in keepAlive")
					if exit {
						return
					}
				}
				glog.Infof("Restarted process: %s, PID: %d", p.p.Prog(), pid)
				p.remRetry -= 1
				mp.keepAlive(id)
			}
			return
		}
		glog.Errorf("mProc Failed to wait on PID: %d cannot restart", pid)

	}()
}

func (mp *mProc) EndKeepAlive(id uint32) error {
	p := mp.getMp(id)
	if p == nil {
		return errors.New(
			fmt.Sprintf("Error: Process with ID: %d does not exist", id))
	}

	pid, _ := p.p.Pid()
	if glog.V(2) {
		glog.Infof("Ending keep alive on PID: %d", pid)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	mp.mu.Lock()
	p.keepAlive = false

	return nil
}

func (mp *mProc) getMp(id uint32) *managedP {
	defer mp.mu.Unlock()
	mp.mu.Lock()
	return mp.managedProcs[id]
}

func (mp *mProc) SignalProc(id uint32, sig os.Signal) error {
	pro := mp.GetProc(id)
	if pro == nil {
		return errors.New(
			fmt.Sprintf("Error: No proc with Id: %d", id))
	}
	return pro.Signal(sig)
}

func (mp *mProc) WaitProc(id uint32) chan error {
	proc := mp.GetProc(id)
	return proc.Wait()
}

func (mp *mProc) GetProc(id uint32) *proc.Process {
	defer mp.mu.Unlock()
	mp.mu.Lock()
	p := mp.managedProcs[id]
	if p != nil {
		return p.p
	}
	return nil
}

func (mp *mProc) KillProc(id uint32) error {
	proc := mp.GetProc(id)
	if proc == nil {
		return errors.New(
			fmt.Sprintf("Error: No proc with Id: %d", id))
	}
	return proc.Kill()
}
