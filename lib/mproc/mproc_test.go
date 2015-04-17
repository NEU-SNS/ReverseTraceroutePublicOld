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
     * Neither the name of the University of Washington nor the
       names of its contributors may be used to endorse or promote products
       derived from this software without specific prior written permission.

 THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
 ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 DISCLAIMED. IN NO EVENT SHALL <COPYRIGHT HOLDER> BE LIABLE FOR ANY
 DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/
package mproc

import (
	"github.com/NEU-SNS/ReverseTraceroute/lib/mproc/proc"
	"os"
	"testing"
)

func TestManageProcess(t *testing.T) {

	mp := New()
	proc := proc.New("/bin/true", nil)
	if proc == nil {
		t.Error("Could not create process")
	}
	_, err := mp.ManageProcess(proc, false)
	if err != nil {
		t.Error("Process was not started")
	}
}

func TestEndKeepAlive(t *testing.T) {
	mp := New()
	proc := proc.New("/bin/sleep", nil, "10")
	if proc == nil {
		t.Error("Could not create process")
	}
	id, err := mp.ManageProcess(proc, true)
	if err != nil {
		t.Error("Process was not started")
	}

	err = mp.EndKeepAlive(id)
	if err != nil {
		t.Errorf("EndKeepAlive failed: %v", err)
	}

}

func TestGetProc(t *testing.T) {

	mp := New()
	proc := proc.New("/bin/true", nil)
	_, err := mp.ManageProcess(proc, false)
	if err != nil {
		t.Error("TestGetProc failed to manage proc")
	}
}

func TestWait(t *testing.T) {
	mp := New()
	proc := proc.New("/bin/true", nil)
	id, err := mp.ManageProcess(proc, false)
	if err != nil {
		t.Error("TestWait failed to manage process")
	}
	done := <-mp.WaitProc(id)
	if done != nil {
		t.Error("Wait failed: %v", done)
	}
	ws := mp.GetProc(id).GetWaitStatus()
	if ws == nil {
		t.Error("Wait failed to set ProcState: %v", ws)
	}
}

func TestKill(t *testing.T) {
	mp := New()
	proc := proc.New("/bin/sleep", nil, "20")
	if proc == nil {
		t.Error("Could not create process")
	}
	id, err := mp.ManageProcess(proc, false)
	if err != nil {
		t.Error("TestKill failed to manage process")
	}
	mp.KillProc(id)
	done := <-mp.WaitProc(id)
	if done != nil {
		t.Error("Failed to wait in Testkill")
	}
	ps := proc.GetWaitStatus()
	if ps.String() != "signal: killed" {
		t.Error("Proc Was not killed: %v", ps)
	}
}

func TestStartError(t *testing.T) {
	mp := New()
	proc := proc.New("true", nil)
	_, err := mp.ManageProcess(proc, false)

	if err == nil {
		t.Error("Failed Proc didn't return error")
	}
}

func TestKillNoProc(t *testing.T) {
	mp := New()
	err := mp.KillProc(-1)
	if err == nil {
		t.Error("Kill, Err not returned for invalid pid")
	}

}

func TestSignal(t *testing.T) {
	mp := New()
	proc := proc.New("/bin/sleep", nil, "20")
	if proc == nil {
		t.Error("Could not create process")
	}
	id, err := mp.ManageProcess(proc, false)
	if err != nil {
		t.Error("TestSignal failed to manage process")
	}
	e := mp.SignalProc(id, os.Kill)
	if e != nil {
		t.Error("TestSignal failed to signal proc")
	}
	done := <-mp.WaitProc(id)
	if done != nil {
		t.Error("Failed to wait in TestSignal")
	}
	ps := proc.GetWaitStatus()
	if ps.String() != "signal: killed" {
		t.Error("Proc Was not signaled: %v", ps)
	}
}

func TestSignalBadPid(t *testing.T) {
	mp := New()
	err := mp.SignalProc(-1, os.Kill)
	if err == nil {
		t.Error("Failed to return error on invalid Pid")
	}
}

/*
func TestRestart(t *testing.T) {
	mp := New()
	proc := &Process{prog: "/bin/sleep", argv: []string{"5"}, restart: true}
	if proc == nil {
		t.Error("Could not create process")
	}
	mp.ManageProcess(proc)
	select {
	case <-time.After(6 * time.Second):
		err := <-mp.WaitProc(proc.proc.Pid)
		if err != nil {
			t.Error("Could not wait on Proc")
		}
		_, nerr := os.FindProcess(proc.proc.Pid)
		if nerr != nil {
			t.Error("Process did not restart")
		}
	}
	proc.restart = false
	mp.KillProc(proc.proc.Pid)
}*/
