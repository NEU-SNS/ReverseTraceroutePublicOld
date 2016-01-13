package plcontroller

import (
	"fmt"
	"net/url"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/dataaccess"
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/httpupdate"
	"github.com/NEU-SNS/ReverseTraceroute/log"
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
	install string = "sudo bash << EOF\n" +
		"wget http://www.ccs.neu.edu/home/rhansen2/plvp.tar.gz\n" +
		"tar xzf plvp.tar.gz\n" +
		"rm plvp.tar.gz\n" +
		"cd plvp\n" +
		"sudo /home/uw_geoloc4/plvp/install.sh" +
		"EOF\n"

	version string = "sudo /home/uw_geoloc4/plvp/plvp --version"
	update  string = "sudo bash << EOF\n" +
		"/sbin/service plvp stop\n" +
		"/usr/bin/yum remove -y plvp\n" +
		"cd /tmp\n" +
		"mkdir %d\n" +
		"cd %d\n" +
		"/usr/bin/wget %s\n" +
		"/usr/bin/yum install -y --nogpgcheck %s\n" +
		"/sbin/service plvp start\n" +
		"EOF\n"
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

func maintainVPs(vps []*dm.VantagePoint, uname, certpath, updateURL string, db *dataaccess.DataAccess, dc chan struct{}) error {
	var wg sync.WaitGroup
	for _, vp := range vps {
		wg.Add(1)
		go func(v *dm.VantagePoint) {
			defer wg.Done()
			err := checkVP(v, uname, certpath, updateURL)
			var res string
			if err != nil {
				res = err.Error()
			} else {
				res = "Healthy"
			}
			select {
			case <-dc:
				return
			default:
				err = db.UpdateCheckStatus(v.Ip, res)
				if err != nil {
					log.Errorf("Failed to update Check Status: %v", err)
				}
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

func checkVP(vp *dm.VantagePoint, uname, certPath, updateURL string) error {
	if vp == nil {
		return errorNilVP
	}
	stat, err := checkRunning(getCmd(vp, uname, certPath, status))
	if err != nil {
		return err
	}
	switch stat {
	case running:
		return handleRunning(vp, uname, certPath, updateURL)
	case stopped:
		return handleStopped(getCmd(vp, uname, certPath, start))
	case notFound:
		httpupdate.CheckUpdate(updateURL, "0.0.0")
		urlString := httpupdate.FetchUrl()
		_, err := url.Parse(urlString)
		if err != nil {
			return err
		}
		err = installService(
			getCmd(
				vp,
				uname,
				certPath,
				install),
		)

		if err != nil {
			return err
		}
		return handleStopped(getCmd(vp, uname, certPath, start))
	case pidExists:
		resetService(getCmd(vp, uname, certPath, restart))
	case unknown:
		return errorUnknownService
	}
	return nil
}

func handleStopped(cmd *exec.Cmd) error {
	ec := make(chan error, 1)
	dc := make(chan struct{})
	go func() {
		out, err := cmd.CombinedOutput()
		if _, ok := err.(*exec.ExitError); err != nil && !ok {
			ec <- err
			return
		}
		if failed.Match(out) {
			ec <- errorFailedToStart
			return
		}
		close(dc)
	}()
	select {
	case <-time.After(time.Second * 25):
		err := cmd.Process.Kill()
		if err != nil {
			return err
		}
		return errorVpTimeout
	case err := <-ec:
		return err
	case <-dc:
	}
	return nil
}

func resetService(cmd *exec.Cmd) error {
	ec := make(chan error, 1)
	dc := make(chan struct{})
	go func() {
		out, err := cmd.CombinedOutput()
		if _, ok := err.(*exec.ExitError); err != nil && !ok {
			ec <- err
			return
		}
		if failed.Match(out) {
			ec <- errorFailedToRestart
			return
		}
		close(dc)
	}()
	select {
	case <-time.After(time.Second * 25):
		err := cmd.Process.Kill()
		if err != nil {
			return err
		}
		return errorVpTimeout
	case err := <-ec:
		return err
	case <-dc:
	}
	return nil
}

func handleRunning(vp *dm.VantagePoint, uname, certPath, updateURL string) error {
	if vp.Controller == 0 {
		return resetService(getCmd(vp, uname, certPath, restart))
	}
	/*
		v, err := checkVersion(getCmd(vp, uname, certPath, version))
		if err != nil {
			log.Info("Returning error from handleRunning")
			log.Infof("Version: %v", v)
			return err
		}
		update, err := httpupdate.CheckUpdate(updateURL, v)
		if err != nil {
			return err
		}
		if update {
			log.Infof("Updating, got version: %v")
			urlString := httpupdate.FetchUrl()
			_, err := url.Parse(urlString)
			if err != nil {
				return err
			}
			err = updateService(getCmd(
				vp,
				uname,
				certPath,
				install,
			))
			if err != nil {
				return err
			}
		}
	*/
	return nil
}

func updateService(cmd *exec.Cmd) error {
	ec := make(chan error, 1)
	go func() {
		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Errorf("Failed to update service: %s", out)
		}
		ec <- err
		return
	}()
	select {
	case <-time.After(time.Second * 25):
		err := cmd.Process.Kill()
		if err != nil {
			log.Error("Failed killing process, update service")
			return err
		}
		return errorVpTimeout
	case err := <-ec:
		return err
	}
}

func checkVersion(cmd *exec.Cmd) (string, error) {
	ec := make(chan error, 1)
	res := make(chan string, 1)
	go func() {
		out, err := cmd.CombinedOutput()
		if _, ok := err.(*exec.ExitError); err != nil && !ok {
			ec <- err
			return
		}
		res <- string(out)
	}()
	select {
	case err := <-ec:
		return "", err
	case v := <-res:
		vlines := strings.Split(v, "\n")
		vline := vlines[len(vlines)-2]
		log.Infof("Got version: %v", vline)
		return vline, nil
	case <-time.After(time.Second * 25):
		err := cmd.Process.Kill()
		if err != nil {
			return "", err
		}
		return "", errorVpTimeout
	}
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
			log.Debugf("Got response: %s", out)
			ps <- running
		case stop.Match(out):
			log.Debugf("Got response: %s", out)
			ps <- stopped
		case nf.Match(out):
			log.Debugf("Got response: %s", out)
			ps <- notFound
		case pidLeft.Match(out):
			log.Debugf("Got response: %s", out)
			ps <- pidExists
		default:
			log.Errorf("Got response: %s", out)
			ec <- errorUnknownService
		}
	}()

	select {
	case err := <-ec:
		return unknown, err
	case stat := <-ps:
		return stat, nil
	case <-time.After(time.Second * 25):
		err := cmd.Process.Kill()
		if err != nil {
			return unknown, err
		}
		return unknown, errorVpTimeout
	}
}

func installService(cmd *exec.Cmd) error {
	ec := make(chan error, 1)
	go func() {
		_, err := cmd.CombinedOutput()
		if err != nil {
			log.Errorf("Failed to install service: %s", err)
		}
		ec <- err
		return
	}()
	select {
	case err := <-ec:
		return err
	case <-time.After(time.Second * 25):
		err := cmd.Process.Kill()
		if err != nil {
			log.Error("Failed killing process, install service")
			return err
		}
		return errorVpTimeout

	}
}
