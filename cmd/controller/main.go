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
	"os"
	"os/signal"
	"syscall"

	"github.com/NEU-SNS/ReverseTraceroute/cache"
	"github.com/NEU-SNS/ReverseTraceroute/config"
	"github.com/NEU-SNS/ReverseTraceroute/controller"
	"github.com/NEU-SNS/ReverseTraceroute/dataaccess/sql"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/util"
)

var conf = controller.NewConfig()

func init() {
	config.SetEnvPrefix("REVTR")
	config.AddConfigPath("./controller.config")

	flag.StringVar(conf.Local.Addr, "a", ":35000",
		"The address that the controller will bind to.")
	flag.BoolVar(conf.Local.CloseStdDesc, "D", false,
		"Determines if the sandard file descriptors are closed")
	flag.StringVar(conf.Local.PProfAddr, "pprof", "localhost:55555",
		"The port for pprof")
	flag.BoolVar(conf.Local.AutoConnect, "auto-connect", false,
		"Autoconnect to 0.0.0.0 and will use port 35000")
	flag.StringVar(conf.Local.CertFile, "cert-file", "cert.pem",
		"The path the the cert file for the the server")
	flag.IntVar(conf.Local.Port, "p", 4382,
		"The port that the controller will use.")
	flag.StringVar(conf.Local.KeyFile, "key-file", "key.pem",
		"The path to the private key for the file")
	flag.Int64Var(conf.Local.ConnTimeout, "conn-timeout", 60,
		"How long to wait for an rpc connection to timeout")
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
	var parseConf controller.Config
	err := config.Parse(flag.CommandLine, &parseConf)
	if err != nil {
		log.Errorf("Failed to parse config: %v", err)
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
		log.Errorf("Failed to create db: %v", err)
		exit(1)
	}

	err = <-controller.Start(conf, db, cache.New())

	if err != nil {
		log.Errorf("Controller Start returned with error: %v", err)
		exit(1)
	}
}

func sigHandle() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGKILL, syscall.SIGINT,
		syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGSTOP)
	for sig := range c {
		log.Infof("Got signal: %v", sig)
		controller.HandleSig(sig)
		exit(1)
	}
}

func exit(status int) {
	os.Exit(status)
}
