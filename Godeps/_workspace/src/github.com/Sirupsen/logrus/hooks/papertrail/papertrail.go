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
package logrus_papertrail

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/Sirupsen/logrus"
)

const (
	format = "Jan 2 15:04:05"
)

// PapertrailHook to send logs to a logging service compatible with the Papertrail API.
type PapertrailHook struct {
	Host    string
	Port    int
	AppName string
	UDPConn net.Conn
}

// NewPapertrailHook creates a hook to be added to an instance of logger.
func NewPapertrailHook(host string, port int, appName string) (*PapertrailHook, error) {
	conn, err := net.Dial("udp", fmt.Sprintf("%s:%d", host, port))
	return &PapertrailHook{host, port, appName, conn}, err
}

// Fire is called when a log event is fired.
func (hook *PapertrailHook) Fire(entry *logrus.Entry) error {
	date := time.Now().Format(format)
	msg, _ := entry.String()
	payload := fmt.Sprintf("<22> %s %s: %s", date, hook.AppName, msg)

	bytesWritten, err := hook.UDPConn.Write([]byte(payload))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to send log line to Papertrail via UDP. Wrote %d bytes before error: %v", bytesWritten, err)
		return err
	}

	return nil
}

// Levels returns the available logging levels.
func (hook *PapertrailHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
	}
}
