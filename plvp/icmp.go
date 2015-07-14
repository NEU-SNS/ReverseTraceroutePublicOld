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

// Package plvp is the library for creating a vantage poing on a planet-lab node
package plvp

import (
	"fmt"
	"net"

	"github.com/golang/glog"
	opt "github.com/rhansen2/ipv4optparser"
	"golang.org/x/net/icmp"
	"golang.org/x/net/internal/iana"
	"golang.org/x/net/ipv4"
)

const (
	// ID is the ICMDPID magic number
	ID = 0xf0f1
	// SEQ is the ICMP seq number magic number
	SEQ = 0xf2f3
)

// SpoofPingMonitor monitors for ICMP echo replies that match the magic numbers
type SpoofPingMonitor struct {
	conn *ipv4.RawConn
}

// Start the SpoofPingMonitor
func (sm *SpoofPingMonitor) Start(addr string, ips chan net.IP, ec chan error) {
	glog.Infof("Starting SpoofPingMonitor on addr: %s:", addr)
	pc, err := net.ListenPacket(fmt.Sprintf("ip4:%d", iana.ProtocolICMP), addr)
	if err != nil {
		glog.Errorf("Error starting SpoofPingMonitor: %v", err)
		ec <- err
		return
	}
	sm.conn, err = ipv4.NewRawConn(pc)
	if err != nil {
		ec <- err
		return
	}
	for {
		buf := make([]byte, 1500)

		header, pload, _, err := sm.conn.ReadFrom(buf)
		if err != nil {
			glog.Errorf("Could not read packet")
			ec <- err
			continue
		}
		mess, err := icmp.ParseMessage(iana.ProtocolICMP, pload)
		if err != nil {
			glog.Errorf("Couldn't parse IPv4 message: %v", err)
			ec <- err
			continue
		}

		if echo, ok := mess.Body.(*icmp.Echo); ok {
			if echo.ID == ID && echo.Seq == SEQ {
				if len(echo.Data) < 4 {
					glog.Infof("Not enough data in echo %v", echo.Data)
					continue
				}
				ip := net.IPv4(echo.Data[0],
					echo.Data[1],
					echo.Data[2],
					echo.Data[3])
				if ip == nil {
					ec <- fmt.Errorf("Could not create IP from echo reply body")
					continue
				}
				var id uint32
				id |= uint32(echo.Data[4]) << 24
				id |= uint32(echo.Data[5]) << 16
				id |= uint32(echo.Data[6]) << 8
				id |= uint32(echo.Data[7])
				options, err := opt.Parse(header.Options)
				if err != nil {
					glog.Errorf("Failed to parse IPv4 options: %v", err)
				}
				for _, option := range options {
					glog.Info(option.Type)
					if option.Type == opt.RecordRoute {
						_, err = option.ToRecordRoute()
						if err != nil {
							glog.Errorf("Error parsing record route: %v", err)
							continue
						}
					}
					if option.Type == opt.InternetTimestamp {
						_, err = option.ToTimeStamp()
						if err != nil {
							glog.Errorf("Error parsing timestamp: %v", err)
							continue
						}
					}
				}
				glog.Infof("Got spoofed echo-reply from: %s, with id: %d", ip, id)
				ips <- ip
				continue

			}
			glog.Info("Got non-spoofed echo-reply")
			continue
		}
		glog.Info("Received non-echo icmp packet")
	}
}

// Stop stops the SpoofPingMonitor
func (sm *SpoofPingMonitor) Stop() error {
	return sm.conn.Close()
}
