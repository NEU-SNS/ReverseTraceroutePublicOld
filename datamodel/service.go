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
package datamodel

import (
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/log"
)

// Service represents a service
type Service struct {
	Url  string
	Key  ServiceT
	Port int
	//Protects Ips and last load times

	mu          sync.Mutex
	lastUpdated time.Time
	ips         []string
}

func (s *Service) GetIp() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.ips) == 0 || time.Since(s.lastUpdated) > time.Minute*5 {
		log.Infof("Resolving url: %s", s.Url)
		ips, err := net.LookupHost(s.Url)
		if err != nil {
			return "", err
		}
		log.Infof("Got IPs: %v", ips)
		s.lastUpdated = time.Now()
		s.ips = ips
	}
	rand.Seed(time.Now().UnixNano())
	ip := s.ips[rand.Intn(len(s.ips))]
	return fmt.Sprintf("%s:%d", ip, s.Port), nil
}

// UnmarshalYAML is for the yaml library
func (s *ServiceT) UnmarshalYAML(unm func(interface{}) error) error {
	var text string
	err := unm(&text)
	if err != nil {
		return err
	}
	if val, ok := ServiceT_value[text]; ok {
		*s = ServiceT(val)
		return nil
	}
	return fmt.Errorf("Invalid Value for ServiceT")
}

// MarshalYAML is for the yaml library
func (s *ServiceT) MarshalYAML() (interface{}, error) {
	text := ServiceT_name[int32(*s)]
	return &text, nil
}
