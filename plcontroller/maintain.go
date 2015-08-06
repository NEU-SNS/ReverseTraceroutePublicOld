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
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"regexp"
	"time"

	"golang.org/x/crypto/ssh"

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

	run     = regexp.MustCompile("running")
	stop    = regexp.MustCompile("stopped")
	nf      = regexp.MustCompile("unrecognized")
	failed  = regexp.MustCompile("FAILED")
	pidLeft = regexp.MustCompile("plvp dead but pid file exists")
)

func maintainVPs(vps []*dm.VantagePoint, uname, certpath string, db dataaccess.VPProvider) error {
	s, err := getCert(certpath)
	if err != nil {
		return err
	}
	conf := &ssh.ClientConfig{
		User: uname,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(s),
		},
	}
	for _, vp := range vps {
		err := checkVP(vp, conf)
		var res string
		if err != nil {
			res = err.Error()
		} else {
			res = "Healthy"
		}
		err = db.UpdateCheckStatus(vp.Ip, res)
		if err != nil {
			glog.Errorf("Failed to update Check Status: %v", err)
		}

	}
	return nil
}
func getCert(path string) (ssh.Signer, error) {
	f, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	s, err := ssh.ParsePrivateKey(f)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func checkVP(vp *dm.VantagePoint, config *ssh.ClientConfig) error {
	if vp == nil {
		return errorNilVP
	}
	cl, err := dial("tcp4", fmt.Sprintf("%s:%d", vp.Hostname, vp.Port), config, time.Second*5)
	if err != nil {
		return err
	}
	defer cl.Close()
	sess, err := cl.NewSession()
	if err != nil {
		return err
	}
	stat, err := checkRunning(sess)
	if _, ok := err.(*ssh.ExitError); err != nil && !ok {
		return err
	}
	sess, err = cl.NewSession()
	if err != nil {
		return err
	}
	switch stat {
	case running:
		return handleRunning(vp, sess)
	case stopped:
		return handleStopped(sess)
	case notFound:
	case pidExists:
		resetService(sess)
	case unknown:
		return errorUnknownService
	}
	return nil
}

func handleStopped(sess *ssh.Session) error {
	defer sess.Close()
	var out bytes.Buffer
	sess.Stdout = &out
	err := sess.Run(start)
	if _, ok := err.(*ssh.ExitError); err != nil && !ok {
		return err
	}
	if failed.Match(out.Bytes()) {
		return errorFailedToStart
	}
	return nil
}

func resetService(sess *ssh.Session) error {
	var out bytes.Buffer
	sess.Stdout = &out
	err := sess.Run(restart)
	if _, ok := err.(*ssh.ExitError); err != nil && !ok {
		return err
	}
	if failed.Match(out.Bytes()) {
		glog.Errorf("Failed to restart: %s", out.String())
		return errorFailedToRestart
	}
	return nil
}

func handleRunning(vp *dm.VantagePoint, sess *ssh.Session) error {
	defer sess.Close()
	if vp.Controller != 0 {
		return nil
	}
	return resetService(sess)
	return nil
}

func checkRunning(sess *ssh.Session) (procStatus, error) {
	var out bytes.Buffer
	defer sess.Close()
	sess.Stdout = &out
	err := sess.Run(status)
	if err != nil {
		return unknown, err
	}
	if run.Match(out.Bytes()) {
		return running, nil
	}
	if stop.Match(out.Bytes()) {
		return stopped, nil
	}
	if nf.Match(out.Bytes()) {
		return notFound, nil
	}
	if pidLeft.Match(out.Bytes()) {
		return pidExists, nil
	}
	return unknown, nil
}

func dial(n, addr string, conf *ssh.ClientConfig, to time.Duration) (*ssh.Client, error) {
	con, err := net.DialTimeout(n, addr, to)
	if err != nil {
		return nil, err
	}
	conn, nc, rc, err := ssh.NewClientConn(con, addr, conf)
	if err != nil {
		return nil, err
	}
	return ssh.NewClient(conn, nc, rc), nil
}
