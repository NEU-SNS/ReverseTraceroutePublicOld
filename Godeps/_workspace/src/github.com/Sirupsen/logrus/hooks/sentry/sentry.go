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
package logrus_sentry

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/getsentry/raven-go"
)

var (
	severityMap = map[logrus.Level]raven.Severity{
		logrus.DebugLevel: raven.DEBUG,
		logrus.InfoLevel:  raven.INFO,
		logrus.WarnLevel:  raven.WARNING,
		logrus.ErrorLevel: raven.ERROR,
		logrus.FatalLevel: raven.FATAL,
		logrus.PanicLevel: raven.FATAL,
	}
)

func getAndDel(d logrus.Fields, key string) (string, bool) {
	var (
		ok  bool
		v   interface{}
		val string
	)
	if v, ok = d[key]; !ok {
		return "", false
	}

	if val, ok = v.(string); !ok {
		return "", false
	}
	delete(d, key)
	return val, true
}

func getAndDelRequest(d logrus.Fields, key string) (*http.Request, bool) {
	var (
		ok  bool
		v   interface{}
		req *http.Request
	)
	if v, ok = d[key]; !ok {
		return nil, false
	}
	if req, ok = v.(*http.Request); !ok || req == nil {
		return nil, false
	}
	delete(d, key)
	return req, true
}

// SentryHook delivers logs to a sentry server.
type SentryHook struct {
	// Timeout sets the time to wait for a delivery error from the sentry server.
	// If this is set to zero the server will not wait for any response and will
	// consider the message correctly sent
	Timeout time.Duration

	client *raven.Client
	levels []logrus.Level
}

// NewSentryHook creates a hook to be added to an instance of logger
// and initializes the raven client.
// This method sets the timeout to 100 milliseconds.
func NewSentryHook(DSN string, levels []logrus.Level) (*SentryHook, error) {
	client, err := raven.New(DSN)
	if err != nil {
		return nil, err
	}
	return &SentryHook{100 * time.Millisecond, client, levels}, nil
}

// NewWithTagsSentryHook creates a hook with tags to be added to an instance
// of logger and initializes the raven client. This method sets the timeout to
// 100 milliseconds.
func NewWithTagsSentryHook(DSN string, tags map[string]string, levels []logrus.Level) (*SentryHook, error) {
	client, err := raven.NewWithTags(DSN, tags)
	if err != nil {
		return nil, err
	}
	return &SentryHook{100 * time.Millisecond, client, levels}, nil
}

// NewWithClientSentryHook creates a hook using an initialized raven client.
// This method sets the timeout to 100 milliseconds.
func NewWithClientSentryHook(client *raven.Client, levels []logrus.Level) (*SentryHook, error) {
	return &SentryHook{100 * time.Millisecond, client, levels}, nil
}

// Called when an event should be sent to sentry
// Special fields that sentry uses to give more information to the server
// are extracted from entry.Data (if they are found)
// These fields are: logger, server_name and http_request
func (hook *SentryHook) Fire(entry *logrus.Entry) error {
	packet := &raven.Packet{
		Message:   entry.Message,
		Timestamp: raven.Timestamp(entry.Time),
		Level:     severityMap[entry.Level],
		Platform:  "go",
	}

	d := entry.Data

	if logger, ok := getAndDel(d, "logger"); ok {
		packet.Logger = logger
	}
	if serverName, ok := getAndDel(d, "server_name"); ok {
		packet.ServerName = serverName
	}
	if req, ok := getAndDelRequest(d, "http_request"); ok {
		packet.Interfaces = append(packet.Interfaces, raven.NewHttp(req))
	}
	packet.Extra = map[string]interface{}(d)

	_, errCh := hook.client.Capture(packet, nil)
	timeout := hook.Timeout
	if timeout != 0 {
		timeoutCh := time.After(timeout)
		select {
		case err := <-errCh:
			return err
		case <-timeoutCh:
			return fmt.Errorf("no response from sentry server in %s", timeout)
		}
	}
	return nil
}

// Levels returns the available logging levels.
func (hook *SentryHook) Levels() []logrus.Level {
	return hook.levels
}
