// +build !386

package datamodel

import (
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/warts"
)

func ConvertTraceroute(in warts.Traceroute) Traceroute {
	t := Traceroute{}
	t.Type = "trace"
	t.UserId = in.Flags.UserID
	t.Src = uint32(in.Flags.Src.Address)
	t.Dst = uint32(in.Flags.Dst.Address)
	t.Method = in.Flags.TraceType.String()
	t.Sport = uint32(in.Flags.SourcePort)
	t.Dport = uint32(in.Flags.DestPort)
	t.StopReason = in.Flags.StopReason.String()
	t.StopData = uint32(in.Flags.StopData)
	tt := TracerouteTime{}
	tt.Sec = in.Flags.StartTime.Sec
	tt.Usec = in.Flags.StartTime.Usec
	tt.Ftime = time.Unix(tt.Sec, tt.Usec*1000).Format(`"` + TRACETIME + `"`)
	t.Start = &tt
	t.HopCount = uint32(in.HopCount)
	t.Attempts = uint32(in.Flags.Attempts)
	t.Hoplimit = uint32(in.Flags.HopLimit)
	t.Firsthop = uint32(in.Flags.StartTTL)
	t.Wait = uint32(in.Flags.TimeoutS)
	t.WaitProbe = uint32(in.Flags.MinWaitCenti)
	t.Tos = uint32(in.Flags.IPToS)
	t.ProbeSize = uint32(in.Flags.ProbeSize)
	hops := convertHops(in)
	t.Hops = hops
	return t
}

func convertHops(in warts.Traceroute) []*TracerouteHop {
	hops := in.Hops
	retHops := make([]*TracerouteHop, in.HopCount)
	for i, hop := range hops {
		h := &TracerouteHop{}
		h.Addr = uint32(hop.Address.Address)
		h.ProbeTtl = uint32(hop.ProbeTTL)
		h.ProbeId = uint32(hop.ProbeID)
		h.ProbeSize = uint32(hop.ProbeSize)
		h.Rtt = &RTT{}
		h.Rtt.Sec = hop.RTT.Sec
		h.Rtt.Usec = hop.RTT.Usec
		h.ReplyTtl = uint32(hop.ReplyTTL)
		h.ReplyTos = uint32(hop.ToS)
		h.ReplySize = uint32(hop.ReplySize)
		h.ReplyIpid = uint32(hop.IPID)
		h.IcmpType = uint32((hop.ICMPTypeCode & 0xFF00) >> 8)
		h.IcmpCode = uint32(hop.ICMPTypeCode & 0x00FF)
		h.IcmpQTtl = uint32(hop.QuotedTTL)
		h.IcmpQIpl = uint32(hop.QuotedIPLength)
		h.IcmpQTos = uint32(hop.QuotesToS)
		retHops[i] = h
	}
	return retHops
}
