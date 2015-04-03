package scamper

import (
	"fmt"
	"github.com/NEU-SNS/ReverseTraceroute/mproc/proc"
	"github.com/golang/glog"
	"os"
)

const (
	IPv4       = "-4"
	IPv6       = "-6"
	PORT       = "-P"
	SOCKET_DIR = "-U"
	SUDO       = "/usr/bin/sudo"
)

type scamperTool struct {
	sockDir string
}

func makeScamperDir(sockDir string) error {
	return os.Mkdir(sockDir, os.ModeDir)
}

func checkScamperSockDir(sockDir string) error {
	fi, err := os.Stat(sockDir)
	if err != nil {
		if os.IsNotExist(err) {
			return makeScamperDir(sockDir)
		}
		return err
	}
	if !fi.IsDir() {
		return fmt.Errorf("Socket directory path: %s is not a directory",
			sockDir)
	}
	return nil
}

func GetScamperProc(sockDir, scampPort, scamperPath string) *proc.Process {

	err := checkScamperSockDir(sockDir)
	if err != nil {
		glog.Fatal("Error with scamper socket directory: %v", err)
	}
	return proc.New(SUDO, nil, scamperPath,
		IPv4, PORT, scampPort, SOCKET_DIR, sockDir)
}

func GetScamperMeasurementTool(sockDir string) *scamperTool {
	return nil
}

func (st *scamperTool) TraceRoute() {

}

func (st *scamperTool) Ping() {

}

func (st *scamperTool) RRPing() {

}

func (st *scamperTool) TSPing() {

}

func (st *scamperTool) SpoofTr() {

}
