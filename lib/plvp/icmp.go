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
package plvp

import (
	"fmt"
	"github.com/NEU-SNS/ReverseTraceroute/lib/util"
	"github.com/golang/glog"
	"golang.org/x/net/icmp"
	"golang.org/x/net/internal/iana"
	"net"
)

const ()

type SpoofPingMonitor struct {
	conn *icmp.PacketConn
}

func (sm *SpoofPingMonitor) Start(addr string, ips chan int64, ec chan error) {
	glog.Infof("Starting SpoofPingMonitor on addr: %s:", addr)
	pc, err := icmp.ListenPacket("ip4:icmp", addr)
	if err != nil {
		glog.Errorf("Error starting SpoofPingMonitor: %v", err)
		ec <- err
		return
	}
	sm.conn = pc
	for {
		buf := make([]byte, 1024)
		_, _, err := sm.conn.ReadFrom(buf)
		if err != nil {
			glog.Errorf("Could not read packet")
			ec <- err
			continue
		}
		mess, err := icmp.ParseMessage(iana.ProtocolICMP, buf)
		if err != nil {
			glog.Errorf("Couldn't parse IPv4 message: %v", err)
			ec <- err
			continue
		}
		if echo, ok := mess.Body.(*icmp.Echo); ok {
			if echo.ID == 0xf0f1 && echo.Seq == 0xf2f3 {
				glog.Info("Received echo reply with magic numbers")
				ipb := echo.Data[:3]
				ip := net.IPv4(ipb[0], ipb[1], ipb[2], ipb[3])
				if ip == nil {
					ec <- fmt.Errorf("Could not create IP from echo reply body")
					continue
				}
				ipn, err := util.IpStringToInt64(ip.String())
				if err != nil {
					ec <- fmt.Errorf("Could not create ip from parsed echo body")
					continue
				}
				glog.Infof("Got spoofed echo-reply from: %s", ip)
				ips <- ipn
				continue

			}
			glog.Info("Got non-spoofed echo-reply")
			continue
		}
		glog.Info("Received non-echo icmpi")
	}
}

func (sm *SpoofPingMonitor) Stop() error {
	return sm.conn.Close()
}
