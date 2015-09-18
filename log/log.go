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
package log

import (
	"flag"
	"io"
	"os"
	"runtime"

	"github.com/Sirupsen/logrus"
)

var logger = logrus.New()

type logLevel struct{}

func (ll logLevel) String() string {
	return logger.Level.String()
}

func (ll logLevel) Set(l string) error {
	level, err := logrus.ParseLevel(l)
	if err != nil {
		return err
	}
	logger.Level = level
	return nil
}

func handleLogFile(path string) (io.Writer, error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return nil, err
	}
	return f, nil
}

type logOutput struct {
	out string
}

func (lo logOutput) String() string {
	return lo.out
}

func (lo logOutput) Set(out string) error {
	switch out {
	case "stdout":
		logger.Out = os.Stdout
	case "stderr":
		logger.Out = os.Stderr
	default:
		w, err := handleLogFile(out)
		if err != nil {
			return err
		}
		logger.Out = w
		return nil
	}
	return nil
}

func callerInfo() *logrus.Entry {
	// From std lib log library
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "???"
		line = 0
	}
	return logger.WithFields(
		logrus.Fields{
			"file": file,
			"line": line,
		},
	)
}

func init() {
	flag.Var(logLevel{}, "loglevel", "Log level")
	flag.Var(logOutput{}, "logoutput", "Where to send log output")
}

func Debugf(format string, args ...interface{}) {
	callerInfo().Debugf(format, args...)
}

func Infof(format string, args ...interface{}) {
	callerInfo().Infof(format, args...)
}

func Printf(format string, args ...interface{}) {
	callerInfo().Printf(format, args...)
}

func Warnf(format string, args ...interface{}) {
	callerInfo().Warnf(format, args...)
}

func Warningf(format string, args ...interface{}) {
	callerInfo().Warnf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	callerInfo().Errorf(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	callerInfo().Fatalf(format, args...)
}

func Panicf(format string, args ...interface{}) {
	callerInfo().Panicf(format, args...)
}

func Debug(args ...interface{}) {
	callerInfo().Debug(args...)
}

func Info(args ...interface{}) {
	callerInfo().Info(args...)
}

func Print(args ...interface{}) {
	callerInfo().Info(args...)
}

func Warn(args ...interface{}) {
	callerInfo().Warn(args...)
}

func Warning(args ...interface{}) {
	callerInfo().Warn(args...)
}

func Error(args ...interface{}) {
	callerInfo().Error(args...)
}

func Fatal(args ...interface{}) {
	callerInfo().Fatal(args...)
}

func Panic(args ...interface{}) {
	callerInfo().Panic(args...)
}

func Debugln(args ...interface{}) {
	callerInfo().Debugln(args...)
}

func Infoln(args ...interface{}) {
	callerInfo().Infoln(args...)
}

func Println(args ...interface{}) {
	callerInfo().Println(args...)
}

func Warnln(args ...interface{}) {
	callerInfo().Warnln(args...)
}

func Warningln(args ...interface{}) {
	callerInfo().Warnln(args...)
}

func Errorln(args ...interface{}) {
	callerInfo().Errorln(args...)
}

func Fatalln(args ...interface{}) {
	callerInfo().Fatalln(args...)
}

func Panicln(args ...interface{}) {
	callerInfo().Panicln(args...)
}
