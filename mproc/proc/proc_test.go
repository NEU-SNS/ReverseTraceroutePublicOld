package proc

import (
	"os"
	"testing"
)

func TestWait(t *testing.T) {
	proc := &Process{prog: "/bin/true"}
	if proc == nil {
		t.Error("Could not create process")
	}
	_, err := proc.Start()
	if err != nil {
		t.Error("TestWait Failed to start process.")

	}
	done := <-proc.Wait()
	if done != nil {
		t.Error("TestWaitWait failed: ", done)
	}
	waitStatus := proc.GetWaitStatus()

	if waitStatus == nil {
		t.Error("TestWait Wait failed to set ProcState: ", waitStatus)
	}
}

func TestKill(t *testing.T) {
	proc := &Process{prog: "/bin/sleep", argv: []string{"20"}}
	if proc == nil {
		t.Error("Could not create process")
	}
	_, err := proc.Start()
	if err != nil {
		t.Error("Failed to start proc")
	}
	if e := proc.Kill(); e != nil {
		t.Error("TestKill Kill failed: ", e)
	}

	done := <-proc.Wait()
	if done != nil {
		t.Error("Failed to wait in Testkill")
	}
	if proc.GetWaitStatus().String() != "signal: killed" {
		t.Error("Proc Was not killed: ", proc.GetWaitStatus())
	}
}

func TestStartError(t *testing.T) {
	proc := &Process{prog: "true"}
	_, err := proc.Start()

	if err == nil {
		t.Error("Failed Proc didn't return error")
	}
}

func TestSignal(t *testing.T) {
	proc := &Process{prog: "/bin/sleep", argv: []string{"20"}}
	if proc == nil {
		t.Error("Could not create process")
	}
	_, err := proc.Start()
	if err != nil {
		t.Error("TestSignal Failed to start process")
	}

	if e := proc.Signal(os.Kill); e != nil {
		t.Error("TestSignal could not signal process: ", e)
	}

	done := <-proc.Wait()
	if done != nil {
		t.Error("Failed to wait in TestSignal")
	}
	if proc.GetWaitStatus().String() != "signal: killed" {
		t.Error("Proc Was not signaled: ", proc.GetWaitStatus())
	}

}

func TestGetPid(t *testing.T) {
	proc := &Process{prog: "/bin/sleep", argv: []string{"2"}}
	if proc == nil {
		t.Error("Could not create process")
	}
	_, err := proc.Start()
	if err != nil {
		t.Error("TestSignal Failed to start process")
	}
	_, e := proc.Pid()
	if e != nil {
		t.Error("TestGetPid Error: ", e)
	}
}

func TestHasProc(t *testing.T) {
	proc := &Process{prog: "/bin/sleep", argv: []string{"2"}}
	if proc == nil {
		t.Error("Could not create process")
	}
	hasProc := proc.HasProc()
	if hasProc != false {
		t.Error("TestHasProc didn't return false for an unstarted process")
	}
}
