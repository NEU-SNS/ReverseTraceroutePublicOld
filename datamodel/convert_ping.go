// +build !386

package datamodel

import "github.com/NEU-SNS/ReverseTraceroute/warts"

// ConvertPing converts a warts ping to a DM ping
func ConvertPing(in warts.Ping) Ping {
	p := Ping{}
	p.Src = uint32(in.Flags.Src.Address)
	p.Dst = uint32(in.Flags.Dst.Address)
	p.Type = in.Type
	p.Method = in.Flags.PingMethod.String()
	dmt := &Time{}
	dmt.Sec = in.Flags.StartTime.Sec
	dmt.Usec = in.Flags.StartTime.Usec
	p.Start = dmt
	p.PingSent = uint32(in.Flags.ProbeCount)
	p.ProbeSize = uint32(in.Flags.ProbeSize)
	p.UserId = in.Flags.UserID
	p.Ttl = uint32(in.Flags.ProbeTTL)
	p.Wait = uint32(in.Flags.ProbeWaitS)
	p.Timeout = uint32(in.Flags.ProbeTimeout)
	p.Flags = in.Flags.PF.Strings()
	replies := make([]*PingResponse, in.ReplyCount)
	for i, resp := range in.PingReplies {
		rep := &PingResponse{}
		rep.From = uint32(resp.Addr.Address)
		rep.Seq = uint32(resp.ProbeID)
		rep.ReplySize = uint32(resp.ReplySize)
		rep.ReplyTtl = uint32(resp.ReplyTTL)
		rep.ReplyProto = resp.ReplyProto.String()
		txt := &Time{}
		txt.Sec = resp.Tx.Sec
		txt.Usec = resp.Tx.Usec
		rep.Tx = txt
		rxt := &Time{}
		rxt.Sec = txt.Sec + resp.RTT.Sec
		rxt.Usec = txt.Usec + resp.RTT.Usec
		rep.Rx = rxt
		rep.Rtt = uint32(resp.RTT.Sec*1000000 + resp.RTT.Usec)
		rep.ProbeIpid = uint32(resp.ProbeIPID)
		rep.ReplyIpid = uint32(resp.ReplyIPID)
		rep.IcmpType = uint32((resp.ICMP & 0xFF00) >> 8)
		rep.IcmpCode = uint32(resp.ICMP & 0x00FF)
		if in.IsTsOnly() {
			rep.Tsonly = make([]uint32, 0)
			for _, ts := range resp.V4TS.TimeStamps {
				rep.Tsonly = append(rep.Tsonly, uint32(ts))
			}
		} else if in.IsTsAndAddr() {
			p.Flags = append(p.Flags, "tsandaddr")
			rep.Tsandaddr = make([]*TsAndAddr, 0)
			for i, ts := range resp.V4TS.TimeStamps {
				tsa := &TsAndAddr{}
				tsa.Ip = uint32(resp.V4TS.Addrs[i].Address)
				tsa.Ts = uint32(ts)
				rep.Tsandaddr = append(rep.Tsandaddr, tsa)
			}
		}
		for _, addr := range resp.V4RR.Addrs {
			rep.RR = append(rep.RR, uint32(addr.Address))
		}
		replies[i] = rep
	}
	p.Responses = replies
	stat := in.GetStats()
	pstats := &PingStats{}
	pstats.Loss = float32(stat.Loss)
	pstats.Max = stat.Max
	pstats.Min = stat.Min
	pstats.Avg = stat.Avg
	pstats.Stddev = stat.StdDev
	pstats.Replies = int32(stat.Replies)
	p.Statistics = pstats

	return p
}
