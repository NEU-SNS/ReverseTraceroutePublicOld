package proc

import (
	"os"
	"testing"
)

func TestManageProcess(t *testing.T) {

	mp := New()
	proc := &Process{prog: "/bin/true"}
	if proc == nil {
		t.Error("Could not create process")
	}
	runProc := mp.ManageProcess(proc)
	if runProc.proc == nil {
		t.Error("Process was not started")
	}
}

func TestManageProcessStarted(t *testing.T) {

	mp := New()
	proc := &Process{prog: "/bin/true"}
	//Cheat just for the test because the process would be started
	//in another way
	proc.manager = mp
	_, err := proc.start()
	if err != nil {
		t.Error("Failed to start process, Test TestManageProcessStarted")
	}
	mp.ManageProcess(proc)
}

func TestGetProc(t *testing.T) {

	mp := New()
	proc := &Process{prog: "/bin/true"}
	runProc := mp.ManageProcess(proc)
	getProc := mp.GetProc(runProc.proc.Pid)

	if runProc != getProc {
		t.Error("GetProc returned different Proc")
	}
}

func TestWait(t *testing.T) {
	mp := New()
	proc := &Process{prog: "/bin/true"}
	if proc == nil {
		t.Error("Could not create process")
	}
	runProc := mp.ManageProcess(proc)
	done := <-mp.WaitProc(runProc.proc.Pid)
	if done != nil {
		t.Error("Wait failed: %v", done)
	}
	if runProc.procState == nil {
		t.Error("Wait failed to set ProcState: %v", runProc)
	}
}

/*
func TestKill(t *testing.T) {
	mp := New()
	proc := &Process{prog: "/bin/sleep", argv: []string{"20"}}
	if proc == nil {
		t.Error("Could not create process")
	}
	runProc := mp.ManageProcess(proc)
	mp.KillProc(runProc.proc.Pid)
	done := <-mp.WaitProc(runProc.proc.Pid)
	if done != nil {
		t.Error("Failed to wait in Testkill")
	}
	if runProc.procState.String() != "signal: killed" {
		t.Error("Proc Was not killed: %v", runProc.procState)
	}
}
*/
func TestStartError(t *testing.T) {
	proc := &Process{prog: "true"}
	_, err := proc.start()

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

/*
func TestSignal(t *testing.T) {
	mp := New()
	proc := &Process{prog: "/bin/sleep", argv: []string{"20"}}
	if proc == nil {
		t.Error("Could not create process")
	}
	runProc := mp.ManageProcess(proc)
	mp.SignalProc(runProc.proc.Pid, os.Kill)
	done := <-mp.WaitProc(runProc.proc.Pid)
	if done != nil {
		t.Error("Failed to wait in TestSignal")
	}
	if runProc.procState.String() != "signal: killed" {
		t.Error("Proc Was not signaled: %v", runProc.procState)
	}
}
*/
func TestSignalBadPid(t *testing.T) {
	mp := New()
	proc := &Process{prog: "/bin/sleep", argv: []string{"5"}}
	if proc == nil {
		t.Error("Could not create process")
	}
	mp.ManageProcess(proc)
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
