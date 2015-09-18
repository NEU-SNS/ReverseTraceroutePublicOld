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

	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	opt "github.com/rhansen2/ipv4optparser"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

const (
	// ID is the ICMDPID magic number
	ID = 0xf0f1
	// SEQ is the ICMP seq number magic number
	SEQ = 0xf2f3
)

var (
	dummy           ipv4.ICMPType
	icmpProtocolNum = dummy.Protocol()
)

// SpoofPingMonitor monitors for ICMP echo replies that match the magic numbers
type SpoofPingMonitor struct {
	quit chan struct{}
}

func NewSpoofPingMonitor() *SpoofPingMonitor {
	qc := make(chan struct{})
	return &SpoofPingMonitor{quit: qc}
}

func reconnect(addr string) (*ipv4.RawConn, error) {
	pc, err := net.ListenPacket(fmt.Sprintf("ip4:%d", icmpProtocolNum), addr)
	if err != nil {
		return nil, err
	}
	return ipv4.NewRawConn(pc)
}

var (
	ErrorNotICMPEcho           = fmt.Errorf("Received Non ICMP Probe")
	ErrorNonSpoofedProbe       = fmt.Errorf("Received ICMP Probe that was not spoofed")
	ErrorSpoofedProbeNoID      = fmt.Errorf("Received a spoofed probe with no id")
	ErrorNoSpooferIP           = fmt.Errorf("No spoofer IP found in packet")
	ErrorFailedToParseOptions  = fmt.Errorf("Failed to parse IPv4 options")
	ErrorFailedToConvertOption = fmt.Errorf("Failed to convert IPv4 option")
	ErrorSpooferIP             = fmt.Errorf("Failed to convert spoofer ip")
	ErrorReadError             = fmt.Errorf("Failed to read from conn")
)

func makeId(a, b, c, d byte) uint32 {
	var id uint32
	id |= uint32(a) << 24
	id |= uint32(b) << 16
	id |= uint32(c) << 8
	id |= uint32(d)
	return id
}

func makeRecordRoute(rr opt.RecordRouteOption) (dm.RecordRoute, error) {
	rec := dm.RecordRoute{}
	for _, r := range rr.Routes {
		rec.Hops = append(rec.Hops, uint32(r))
	}
	return rec, nil
}

func makeTimestamp(ts opt.TimeStampOption) (dm.TimeStamp, error) {
	time := dm.TimeStamp{}
	time.Type = dm.TSType(ts.Flags)
	for _, st := range ts.Stamps {
		nst := dm.Stamp{Time: uint32(st.Time), Ip: uint32(st.Addr)}
		time.Stamps = append(time.Stamps, &nst)
	}
	return time, nil
}

func getProbe(conn *ipv4.RawConn) (*dm.Probe, error) {
	// 1500 should be good because we're sending small packets and its the standard MTU
	pBuf := make([]byte, 1500)
	probe := &dm.Probe{}
	// Try and get a packet
	header, pload, _, err := conn.ReadFrom(pBuf)
	if err != nil {
		return nil, ErrorReadError
	}
	// Parse the payload for ICMP stuff
	mess, err := icmp.ParseMessage(icmpProtocolNum, pload)
	if err != nil {
		return nil, err
	}
	if echo, ok := mess.Body.(*icmp.Echo); ok {
		if echo.ID != ID || echo.Seq != SEQ {
			return nil, ErrorNonSpoofedProbe
		}
		if len(echo.Data) < 8 {
			return nil, ErrorSpoofedProbeNoID
		}
		// GetIP of spoofer out of packet
		ip := net.IPv4(echo.Data[0],
			echo.Data[1],
			echo.Data[2],
			echo.Data[3])
		if ip == nil {
			return nil, ErrorNoSpooferIP
		}
		// Get the Id out of the data
		id := makeId(echo.Data[4], echo.Data[5], echo.Data[6], echo.Data[7])
		probe.ProbeId = id
		probe.SpooferIp, err = util.IPtoInt32(ip)
		if err != nil {
			return nil, ErrorSpooferIP
		}
		probe.Src, err = util.IPtoInt32(header.Src)
		probe.Dst, err = util.IPtoInt32(header.Dst)
		// Parse the options
		options, err := opt.Parse(header.Options)
		if err != nil {
			log.Errorf("Failed to parse IPv4 options: %v", err)
			return nil, ErrorFailedToParseOptions
		}
		probe.SeqNum = uint32(echo.Seq)
		probe.Id = uint32(echo.ID)
		for _, option := range options {
			switch option.Type {
			case opt.RecordRoute:
				rr, err := option.ToRecordRoute()
				if err != nil {
					return nil, ErrorFailedToConvertOption
				}
				rec, err := makeRecordRoute(rr)
				if err != nil {
					return nil, ErrorFailedToConvertOption
				}
				probe.RR = &rec
			case opt.InternetTimestamp:
				ts, err := option.ToTimeStamp()
				if err != nil {
					return nil, ErrorFailedToConvertOption
				}
				nts, err := makeTimestamp(ts)
				if err != nil {
					return nil, ErrorFailedToConvertOption
				}
				probe.Ts = &nts
			}
		}
		return probe, nil
	}
	return nil, ErrorNotICMPEcho
}

func (sm *SpoofPingMonitor) poll(addr string, probes chan<- dm.Probe, ec chan error) {
	c, err := reconnect(addr)
	if err != nil {
		log.Errorf("Error starting SpoofPingMonitor: %v", err)
		ec <- err
		return
	}
	for {
		select {
		case <-sm.quit:
			c.Close()
		default:
			var pr *dm.Probe
			if pr, err = getProbe(c); err != nil {
				ec <- err
				switch err {
				case ErrorReadError:
					c, err = reconnect(addr)
					if err != nil {
						log.Errorf("Failed to reconnect: %v", err)
						ec <- err
						return
					}
				}
				continue
			}
			probes <- *pr
		}
	}
}

// Start the SpoofPingMonitor
func (sm *SpoofPingMonitor) Start(addr string, probes chan<- dm.Probe, ec chan error) {
	log.Infof("Starting SpoofPingMonitor on addr: %s:", addr)
	go sm.poll(addr, probes, ec)
}

func (s *SpoofPingMonitor) Quit() {
	close(s.quit)
}
