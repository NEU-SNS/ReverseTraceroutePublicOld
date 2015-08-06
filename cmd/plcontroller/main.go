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
package main

import (
	"flag"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"github.com/NEU-SNS/ReverseTraceroute/config"
	"github.com/NEU-SNS/ReverseTraceroute/dataaccess/sql"
	"github.com/NEU-SNS/ReverseTraceroute/plcontroller"
	"github.com/NEU-SNS/ReverseTraceroute/scamper"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	"github.com/golang/glog"
)

var (
	defaultConfig = "./plcontroller.config"
	configPath    string
)

var conf = plcontroller.NewConfig()

func init() {
	config.SetEnvPrefix("REVTR")
	if configPath == "" {
		config.AddConfigPath(defaultConfig)
	} else {
		config.AddConfigPath(configPath)
	}

	flag.Int64Var(conf.Local.Timeout, "t", 60,
		"The default timeout used for measurement requests.")
	flag.StringVar(conf.Local.Addr, "a", "0.0.0.0",
		"The address that the controller will bind to.")
	flag.IntVar(conf.Local.Port, "p", 4380,
		"The port that the controller will use.")
	flag.BoolVar(conf.Local.CloseStdDesc, "D", false,
		"Determines if the sandard file descriptors are closed.")
	flag.StringVar(conf.Scamper.Port, "scamper-port", "4381",
		"Port that Scamper will use.")
	flag.StringVar(conf.Scamper.SockDir, "socket-dir", "/tmp/scamper_sockets",
		"Directory that scamper will use for its sockets")
	flag.StringVar(conf.Scamper.BinPath, "scamper-bin", "/usr/local/bin/sc_remoted",
		"Path to the scamper binary")
	flag.StringVar(conf.Scamper.ConverterPath, "converter-path", "/usr/local/bin/sc_warts2json",
		"Path for warts parser")
	flag.StringVar(conf.Local.PProfAddr, "pprof", ":55556",
		"The port for pprof")
	flag.StringVar(conf.Local.CertFile, "cert-file", "cert.pem",
		"The path the the cert file for the the server")
	flag.StringVar(conf.Local.KeyFile, "key-file", "key.pem",
		"The path to the private key for the file")
	flag.StringVar(conf.Local.SSHKeyPath, "sshkey-path", "",
		"The path to the key for connecting to planet-lab")
	flag.StringVar(conf.Local.PLUName, "pluname", "",
		"The username to use for logging into planet-lab nodes")
	flag.StringVar(conf.Db.UName, "db-uname", "",
		"The username for the database")
	flag.StringVar(conf.Db.Password, "db-pass", "",
		"The password for the database")
	flag.StringVar(conf.Db.Db, "db-name", "",
		"The name of the database to use")
	flag.StringVar(conf.Db.Host, "db-host", "localhost",
		"The host of the database")
	flag.StringVar(conf.Db.Port, "db-port", "3306",
		"The port used for the database connection")
}

func main() {
	go sigHandle()
	defer glog.Flush()
	var parseConf plcontroller.Config
	err := config.Parse(flag.CommandLine, &parseConf)
	if err != nil {
		glog.Errorf("Failed to parse config: %v", err)
		exit(1)
	}

	util.CloseStdFiles(*conf.Local.CloseStdDesc)

	db, err := sql.NewDB(sql.DbConfig{
		UName:    *conf.Db.UName,
		Password: *conf.Db.Password,
		Host:     *conf.Db.Host,
		Port:     *conf.Db.Port,
		Db:       *conf.Db.Db,
	})

	if err != nil {
		glog.Errorf("Failed to create db: %v", err)
		exit(1)
	}

	err = <-plcontroller.Start(conf, false, db, scamper.NewClient(), plcontroller.ControllerSender{})

	if err != nil {
		glog.Errorf("PLController Start returned with error: %v", err)
		exit(1)
	}
}

func sigHandle() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGKILL, syscall.SIGINT, syscall.SIGTERM,
		syscall.SIGQUIT, syscall.SIGSTOP)
	for sig := range c {
		glog.Infof("Got signal: %v", sig)
		plcontroller.HandleSig(sig)
		exit(1)
	}
}

func exit(status int) {
	glog.Flush()
	os.Exit(status)
}
