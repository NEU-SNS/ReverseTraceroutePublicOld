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
package airbrake

import (
	"errors"
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/tobi/airbrake-go"
)

// AirbrakeHook to send exceptions to an exception-tracking service compatible
// with the Airbrake API.
type airbrakeHook struct {
	APIKey      string
	Endpoint    string
	Environment string
}

func NewHook(endpoint, apiKey, env string) *airbrakeHook {
	return &airbrakeHook{
		APIKey:      apiKey,
		Endpoint:    endpoint,
		Environment: env,
	}
}

func (hook *airbrakeHook) Fire(entry *logrus.Entry) error {
	airbrake.ApiKey = hook.APIKey
	airbrake.Endpoint = hook.Endpoint
	airbrake.Environment = hook.Environment

	var notifyErr error
	err, ok := entry.Data["error"].(error)
	if ok {
		notifyErr = err
	} else {
		notifyErr = errors.New(entry.Message)
	}

	airErr := airbrake.Notify(notifyErr)
	if airErr != nil {
		return fmt.Errorf("Failed to send error to Airbrake: %s", airErr)
	}

	return nil
}

func (hook *airbrakeHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.ErrorLevel,
		logrus.FatalLevel,
		logrus.PanicLevel,
	}
}
