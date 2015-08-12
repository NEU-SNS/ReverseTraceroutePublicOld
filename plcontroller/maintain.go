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
package plcontroller

import (
	"fmt"
	"os/exec"
	"regexp"
	"sync"
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/dataaccess"
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/golang/glog"
)

type procStatus string

const (
	notFound  procStatus = "Service not installed"
	stopped   procStatus = "Service stopped"
	running   procStatus = "Service running"
	pidExists procStatus = "Service dead but pid file exists"
	unknown   procStatus = "Could not get service status"

	sshPath string = "/usr/bin/ssh"
	status  string = "sudo /sbin/service plvp status"
	restart string = "sudo /sbin/service plvp restart"
	start   string = "sudo /sbin/service plvp start"
)

var (
	errorNilVP            = fmt.Errorf("Nil VantagePoint")
	errorVpTimeout        = fmt.Errorf("Timeout while checking VP")
	errorActiveNotRunning = fmt.Errorf("Service active but not running")
	errorCouldntGetStatus = fmt.Errorf("Could not get status of connected VP")
	errorFailedToRestart  = fmt.Errorf("Could not reset service")
	errorFailedToStart    = fmt.Errorf("Could not start service")
	errorUnknownService   = fmt.Errorf("Unknown service")

	run     = regexp.MustCompile("running\\.\\.\\.")
	stop    = regexp.MustCompile("stopped")
	nf      = regexp.MustCompile("unrecognized")
	failed  = regexp.MustCompile("FAILED")
	pidLeft = regexp.MustCompile("plvp dead but pid file exists")

	args = []string{"-o", "ConnectTimeout=20", "-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null"}
)

func maintainVPs(vps []*dm.VantagePoint, uname, certpath string, db dataaccess.VPProvider) error {
	var wg sync.WaitGroup
	for _, vp := range vps {
		wg.Add(1)
		go func(v *dm.VantagePoint) {
			defer wg.Done()
			err := checkVP(v, uname, certpath)
			var res string
			if err != nil {
				res = err.Error()
			} else {
				res = "Healthy"
			}
			err = db.UpdateCheckStatus(v.Ip, res)
			if err != nil {
				glog.Errorf("Failed to update Check Status: %v", err)
			}
		}(vp)

	}
	wg.Wait()
	return nil
}

func getCmd(vp *dm.VantagePoint, uname, certPath, cmds string) *exec.Cmd {
	creds := fmt.Sprintf("%s@%s", uname, vp.Hostname)
	port := fmt.Sprintf("%d", vp.Port)
	cmdArg := []string{
		creds,
		"-i",
		certPath,
		"-p",
		port,
	}
	cmdArg = append(cmdArg, args...)
	cmdArg = append(cmdArg, cmds)
	cmd := exec.Command(sshPath, cmdArg...)
	return cmd
}

func checkVP(vp *dm.VantagePoint, uname, certPath string) error {
	if vp == nil {
		return errorNilVP
	}
	stat, err := checkRunning(getCmd(vp, uname, certPath, status))
	if err != nil {
		return err
	}
	switch stat {
	case running:
		return handleRunning(vp, getCmd(vp, uname, certPath, restart))
	case stopped:
		return handleStopped(getCmd(vp, uname, certPath, start))
	case notFound:
		return errorUnknownService
	case pidExists:
		resetService(getCmd(vp, uname, certPath, restart))
	case unknown:
		return errorUnknownService
	}
	return nil
}

func handleStopped(cmd *exec.Cmd) error {
	ec := make(chan error, 1)
	go func() {
		out, err := cmd.CombinedOutput()
		if _, ok := err.(*exec.ExitError); err != nil && !ok {
			ec <- err
			return
		}
		if failed.Match(out) {
			ec <- errorFailedToStart
		}
	}()
	select {
	case <-time.After(time.Second * 25):
		err := cmd.Process.Kill()
		if err != nil {
			return err
		}
	case err := <-ec:
		return err
	}
	return nil
}

func resetService(cmd *exec.Cmd) error {
	ec := make(chan error, 1)
	go func() {
		out, err := cmd.CombinedOutput()
		if _, ok := err.(*exec.ExitError); err != nil && !ok {
			ec <- err
			return
		}
		if failed.Match(out) {
			ec <- errorFailedToRestart
		}
	}()
	select {
	case <-time.After(time.Second * 25):
		err := cmd.Process.Kill()
		if err != nil {
			return err
		}
	case err := <-ec:
		return err
	}
	return nil
}

func handleRunning(vp *dm.VantagePoint, cmd *exec.Cmd) error {
	if vp.Controller != 0 {
		return nil
	}
	return resetService(cmd)
	return nil
}

func checkRunning(cmd *exec.Cmd) (procStatus, error) {
	ec := make(chan error, 1)
	ps := make(chan procStatus, 1)
	go func() {
		out, err := cmd.CombinedOutput()
		if _, ok := err.(*exec.ExitError); err != nil && !ok {
			ec <- err
			return
		}
		switch {
		case run.Match(out):
			ps <- running
		case stop.Match(out):
			ps <- stopped
		case nf.Match(out):
			ps <- notFound
		case pidLeft.Match(out):
			ps <- pidExists
		}
	}()

	select {
	case <-time.After(time.Second * 25):
		err := cmd.Process.Kill()
		if err != nil {
			return unknown, err
		}
	case err := <-ec:
		return unknown, err
	case stat := <-ps:
		return stat, nil
	}
	return unknown, nil
}
