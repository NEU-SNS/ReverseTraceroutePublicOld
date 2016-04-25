package reversetraceroute

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"reflect"
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/ip_utils"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	"github.com/prometheus/log"
	"golang.org/x/net/context"
)

func (rt *ReverseTraceroute) issueRecordRoutes(ping *datamodel.PingMeasurement) error {
	var cls, sdst, ssrc *string
	cls = new(string)
	sdst = new(string)
	ssrc = new(string)
	*ssrc, _ = util.Int32ToIPString(ping.Src)
	if rrVPsByCluster {
		*sdst, _ = util.Int32ToIPString(ping.Dst)
		*cls = ipToCluster.Get(*sdst)
	} else {
		*sdst, _ = util.Int32ToIPString(ping.Dst)
		*cls = *sdst
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	st, err := rt.cl.Ping(ctx, &datamodel.PingArg{
		Pings: []*datamodel.PingMeasurement{
			ping,
		},
	})
	if err != nil {
		return err
	}
	for {
		p, err := st.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			rt.error(err)
			return err
		}
		pr := p.GetResponses()
		rt.debug("Received RR response: ", pr)
		if len(pr) > 0 {
			inner, ok := rt.rrsSrcToDstToVPToRevHops[*ssrc]
			if ok {
				f, ok := inner[*cls]
				if ok {
					res := processRR(*ssrc, *sdst, pr[0].RR, true)
					f["non_spoofed"] = res
				} else {
					inner[*cls] = make(map[string][]string)
					res := processRR(*ssrc, *sdst, pr[0].RR, true)
					inner[*cls]["non_spoofed"] = res
				}
			} else {
				rt.debug("Setting hops")
				tmp := make(map[string]map[string][]string)
				tmp[*cls] = make(map[string][]string)
				res := processRR(*ssrc, *sdst, pr[0].RR, true)
				tmp[*cls]["non_spoofed"] = res
				rt.rrsSrcToDstToVPToRevHops[*ssrc] = tmp

			}
		}
	}
	return nil
}

func stringSliceRIndex(ss []string, s string) int {
	var rindex int
	rindex = -1
	for i, sss := range ss {
		if sss == s {
			rindex = i
		}
	}
	return rindex
}

func processRR(src, dst string, hops []uint32, removeLoops bool) []string {
	if len(hops) == 0 {
		return []string{}
	}
	dstcls := ipToCluster.Get(dst)
	var hopss []string
	for _, s := range hops {
		hs, _ := util.Int32ToIPString(s)
		hopss = append(hopss, hs)
	}
	log.Debug("Processing RR for src: ", src, " dst ", dst, " hops: ", hopss)
	if ipToCluster.Get(hopss[len(hopss)-1]) == dstcls {
		return []string{}
	}
	i := len(hops) - 1
	var found bool
	// check if we reached dst with at least one hop to spare
	for !found && i > 0 {
		i--
		if dstcls == ipToCluster.Get(hopss[i]) {
			found = true
		}
	}
	if found {
		log.Debug("Found hops RR at: ", i)
		hopss = hopss[i:]
		// remove cluster level loops
		if removeLoops {
			var clusters []string
			for _, hop := range hopss {
				clusters = append(clusters, ipToCluster.Get(hop))
			}
			var retHops []string
			for i, hop := range hopss {
				if i == 0 {
					retHops = append(retHops, hop)
					continue
				}
				if retHops[len(retHops)-1] != hop {
					retHops = append(retHops, hop)
				}
			}
			log.Debug("Got Hops: ", retHops)
			return retHops
		}
		log.Debug("Got Hops: ", hopss)
		return hopss
	}
	return []string{}
}

func (rt *ReverseTraceroute) reverseHopsTRToSrc() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	as, err := rt.at.GetIntersectingPath(ctx)
	var errs []error
	if err != nil {
		return nil
	}
	for _, hop := range rt.CurrPath().LastSeg().Hops() {
		dest, _ := util.IPStringToInt32(rt.Src)
		hops, _ := util.IPStringToInt32(hop)
		rt.debug("Attempting to find TR for hop: ", hop, "(", hops, ")", " to ", dest)
		is := datamodel.IntersectionRequest{
			UseAliases: true,
			Staleness:  rt.Staleness,
			Dest:       dest,
			Address:    hops,
		}
		err := as.Send(&is)
		if err != nil {
			return err
		}
	}
	err = as.CloseSend()
	if err != nil {
		rt.error(err)
	}
	for {
		tr, err := as.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			rt.error(err)
			return err
		}
		rt.debug("Received response: ", tr)
		switch tr.Type {
		case datamodel.IResponseType_PATH:
			rt.debug("Got PATH Response")
			var hs []string
			var found bool
			for _, h := range tr.Path.GetHops() {
				hss, _ := util.Int32ToIPString(h.Ip)
				rt.debug("Fixing up hop: ", hss)
				addr, _ := util.Int32ToIPString(tr.Path.Address)
				if !found && ipToCluster.Get(addr) != ipToCluster.Get(hss) {
					continue
				}
				found = true
				hs = append(hs, hss)
			}
			addrs, _ := util.Int32ToIPString(tr.Path.Address)
			rt.debug("Creating TRtoSrc seg: ", hs, " ", rt.Src, " ", addrs)
			segment := NewTrtoSrcRevSegment(hs, rt.Src, addrs)
			if !rt.AddBackgroundTRSegment(segment) {
				errs = append(errs, fmt.Errorf("Failed to add segment"))
			} else {
				return nil
			}
		case datamodel.IResponseType_NONE_FOUND:
			errs = append(errs, fmt.Errorf("None Found"))
		case datamodel.IResponseType_ERROR:
			errs = append(errs, fmt.Errorf(tr.Error))
		case datamodel.IResponseType_TOKEN:
			rt.tokens = append(rt.tokens, tr)
			errs = append(errs, fmt.Errorf("Token Received: %v", tr))
		}

	}
	if len(errs) == len(rt.CurrPath().LastSeg().Hops()) {
		var errstring bytes.Buffer
		for _, err := range errs {
			errstring.WriteString(err.Error() + " ")
		}
		return fmt.Errorf(errstring.String())
	}
	return nil
}

func (rt *ReverseTraceroute) revtreiveBackgroundTRS() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	as, err := rt.at.GetPathsWithToken(ctx)
	if err != nil {
		return nil
	}
	for _, token := range rt.tokens {
		rt.debug("Sending for token: ", token)
		err := as.Send(&datamodel.TokenRequest{
			Token: token.Token,
		})
		if err != nil {
			return err
		}
	}
	rt.tokens = nil
	err = as.CloseSend()
	if err != nil {
		rt.error(err)
	}
	var errs []error
	for {
		resp, err := as.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			rt.error(err)
			return err
		}
		rt.debug("Received token response: ", resp)
		switch resp.Type {
		case datamodel.IResponseType_PATH:
			var hs []string
			var found bool
			for _, h := range resp.Path.GetHops() {
				hss, _ := util.Int32ToIPString(h.Ip)
				addr, _ := util.Int32ToIPString(resp.Path.Address)
				if !found && ipToCluster.Get(addr) != ipToCluster.Get(hss) {
					continue
				}
				found = true
				hs = append(hs, hss)
			}
			addrs, _ := util.Int32ToIPString(resp.Path.Address)
			segment := NewTrtoSrcRevSegment(hs, rt.Src, addrs)
			if !rt.AddBackgroundTRSegment(segment) {
				errs = append(errs, fmt.Errorf("Failed to add segment"))
			}
		case datamodel.IResponseType_NONE_FOUND:
			errs = append(errs, fmt.Errorf("None Found"))
		case datamodel.IResponseType_ERROR:
			errs = append(errs, fmt.Errorf(resp.Error))
		}
	}
	rt.debug(errs, " Hops ", rt.CurrPath().LastSeg().Hops())
	if len(errs) == len(rt.CurrPath().LastSeg().Hops()) {
		var errstring bytes.Buffer
		for _, err := range errs {
			errstring.WriteString(err.Error() + " ")
		}
		return fmt.Errorf(errstring.String())
	}
	return nil
}

func (rt *ReverseTraceroute) issueTraceroute() error {
	src, _ := util.IPStringToInt32(rt.Src)
	dst, _ := util.IPStringToInt32(rt.LastHop())
	if iputil.IsPrivate(net.ParseIP(rt.LastHop())) {
		return ErrPrivateIP
	}
	tr := datamodel.TracerouteMeasurement{
		Src:        src,
		Dst:        dst,
		CheckCache: true,
		CheckDb:    true,
		Staleness:  rt.Staleness,
		Timeout:    40,
		Wait:       "2",
		Attempts:   "1",
		LoopAction: "1",
		Loops:      "3",
	}
	rt.ProbeCount["tr"]++
	rt.debug("Issuing traceroute src: ", rt.Src, " dst: ", rt.LastHop())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	st, err := rt.cl.Traceroute(ctx, &datamodel.TracerouteArg{
		Traceroutes: []*datamodel.TracerouteMeasurement{&tr},
	})
	if err != nil {
		rt.error(err)
		rt.errorDetails.WriteString("Couldn't complete traceroute.\n")
		return err
	}
	for {
		tr, err := st.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			rt.errorDetails.WriteString("Error running traceroute.\n")
			rt.error(err)
			return err
		}
		if tr.Error != "" {
			rt.error(tr.Error)
			rt.errorDetails.WriteString("Error running traceroute.\n")
			rt.errorDetails.WriteString(tr.Error + "\n")
			return fmt.Errorf("Traceroute failed: %s", tr.Error)
		}
		dstSt, _ := util.Int32ToIPString(tr.Dst)
		cls := ipToCluster.Get(dstSt)
		hops := tr.GetHops()
		var hopst []string
		rt.debug("Got traceroute: ", tr)
		for i, hop := range hops {
			if i != len(hops)-1 {
				j := hop.ProbeTtl + 2
				for j < hops[i].ProbeTtl {
					hopst = append(hopst, "*")
				}
			}
			addrst, _ := util.Int32ToIPString(hop.Addr)
			hopst = append(hopst, addrst)
		}
		if len(hopst) == 0 {
			rt.debug("Found no hops with traceroute")
			rt.errorDetails.WriteString("Traceroute didn't find any hops.\n")
			return fmt.Errorf("Traceroute didn't find hops")
		}
		if len(hopst) > 0 && hopst[len(hopst)-1] != rt.LastHop() {
			rt.errorDetails.WriteString("Traceroute didn't reach destination.\n")
			rt.errorDetails.WriteString(tr.ErrorString() + "\n")
			rt.errorDetails.WriteString(fmt.Sprintf("<a href=\"/runrevtr?src=%s&dst=%s\">Try rerunning from the last responseiv hop!</a>", rt.Src, hopst[len(hopst)-1]))
			return fmt.Errorf("Traceroute didn't reach destination")
		}
		rt.debug("got traceroute ", hopst)
		if in, ok := rt.trsSrcToDstToPath[rt.Src]; ok {
			in[cls] = hopst
		} else {
			rt.trsSrcToDstToPath[rt.Src] = make(map[string][]string)
			rt.trsSrcToDstToPath[rt.Src][cls] = hopst
		}
	}
	return nil
}

func (rt *ReverseTraceroute) reverseHopsAssumeSymmetric() error {
	// if last hop is assumed, add one more from that tr
	if reflect.TypeOf(rt.CurrPath().LastSeg()) == reflect.TypeOf(&DstSymRevSegment{}) {
		rt.debug("Backing off along current path for ", rt.Src, " ", rt.Dst)
		// need to not ignore the hops in the last segment, so can't just
		// call add_hops(revtr.hops + revtr.deadends)
		newSeg := rt.CurrPath().LastSeg().Clone().(*DstSymRevSegment)
		rt.debug("newSeg: ", newSeg)
		var allHops []string
		for i, seg := range *rt.CurrPath().Path {
			// Skip the last one
			if i == len(*rt.CurrPath().Path)-1 {
				continue
			}
			allHops = append(allHops, seg.Hops()...)
		}
		allHops = append(allHops, rt.Deadends()...)
		rt.debug("all hops: ", allHops)
		newSeg.AddHop(allHops)
		rt.debug("New seg: ", newSeg)
		added := rt.AddAndReplaceSegment(newSeg)
		if added {
			rt.debug("Added hop from another DstSymRevSegment")
			return nil
		}
	}
	tr, ok := rt.trsSrcToDstToPath[rt.Src][ipToCluster.Get(rt.LastHop())]
	if !ok {
		err := rt.issueTraceroute()
		if err != nil {
			rt.debug("Issue traceroute err: ", err)
			return ErrNoHopFound
		}
		tr, ok := rt.trsSrcToDstToPath[rt.Src][ipToCluster.Get(rt.LastHop())]
		if ok && len(tr) > 0 && ipToCluster.Get(tr[len(tr)-1]) == ipToCluster.Get(rt.LastHop()) {
			var hToIgnore []string
			hToIgnore = append(hToIgnore, rt.Hops()...)
			hToIgnore = append(hToIgnore, rt.Deadends()...)
			rt.debug("Attempting to add hop from tr ", tr)
			if !rt.AddSegments([]Segment{NewDstSymRevSegment(rt.Src, rt.LastHop(), tr, 1, hToIgnore)}) {
				return ErrNoHopFound
			}
			return nil
		}
		return ErrNoHopFound
	}
	rt.debug("Adding hop from traceroute: ", tr)
	if ok && len(tr) > 0 && ipToCluster.Get(tr[len(tr)-1]) == ipToCluster.Get(rt.LastHop()) {
		var hToIgnore []string
		hToIgnore = append(hToIgnore, rt.Hops()...)
		hToIgnore = append(hToIgnore, rt.Deadends()...)
		rt.debug("Attempting to add hop from tr ", tr)
		if !rt.AddSegments([]Segment{NewDstSymRevSegment(rt.Src, rt.LastHop(), tr, 1, hToIgnore)}) {
			return ErrNoHopFound
		}
		return nil
	}
	return ErrNoHopFound
}

// give a partial reverse traceroute, try to find reverse hops using TS option
// get the ranked se of adjacents, for each, the set of sources to try (self, spoofers)
// if you get a reply, try different source.
// mark info per destination (current hop), then reuse with other adjacents as needed
//
// each time we call this, it will try one potential adjacency per revtr
// (assuming one exists for that revtr). at the end of execution, it shoulod
// either have found that adjacency, is a rev hop or found that it will not be able
// to determine that it is (either it isn't, or our techniques weon't be able to
// tell us if it is)
//
// info I need for each:
// for revtr: s,d,r, the set of R left to consider, or if there are none
// for a given s,d pair, the set of VPs to try using-- start it at self +
// the good spoofers
// whether d is an overstamper
// whether d doesn't stamp
// whether d doesn't stamp but will respond
const (
	dummyIP = "128.208.3.77"
)

func (rt *ReverseTraceroute) reverseHopsTS() error {

	var tsToIssueSrcToProbe = make(map[string][][]string)
	var receiverToSpooferToProbe = make(map[string]map[string][][]string)
	var dstsDoNotStamp [][]string

	checkMapMagic := func(f, s string) {
		if _, ok := receiverToSpooferToProbe[f]; !ok {
			receiverToSpooferToProbe[f] = make(map[string][][]string)
		}
	}
	checksrctohoptosendspoofedmagic := func(f string) {
		if _, ok := rt.tsSrcToHopToSendSpoofed[f]; !ok {
			rt.tsSrcToHopToSendSpoofed[f] = make(map[string]bool)
		}
	}
	rt.initTsSrcToHopToResponseive(rt.Src)
	if rt.tsSrcToHopToResponsive[rt.Src][rt.LastHop()] != 0 {
		rt.debug("No VPS found for ", rt.Src, " last hop: ", rt.LastHop())
		return ErrNoVPs
	}
	adjacents := rt.GetTSAdjacents(ipToCluster.Get(rt.LastHop()))
	rt.debug("Adjacents: ", adjacents)
	if len(adjacents) == 0 {
		rt.debug("No adjacents found")
		return ErrNoAdj
	}
	if rt.tsDstToStampsZero[rt.LastHop()] {
		rt.debug("tsDstToStampsZero wtf")
		for _, adj := range adjacents {
			dstsDoNotStamp = append(dstsDoNotStamp, []string{rt.Src, rt.LastHop(), adj})
		}
	} else if !rt.tsSrcToHopToSendSpoofed[rt.Src][rt.LastHop()] {
		rt.debug("Adding Spoofed TS to send")
		for _, adj := range adjacents {
			tsToIssueSrcToProbe[rt.Src] = append(tsToIssueSrcToProbe[rt.Src], []string{rt.LastHop(), rt.LastHop(), adj, adj, dummyIP})
		}
	} else {
		rt.debug("TS Non of the above")
		spfs := rt.getTimestampSpoofers(rt.Src, rt.LastHop())
		for _, adj := range adjacents {
			for _, spf := range spfs {
				checkMapMagic(rt.Src, spf)
				receiverToSpooferToProbe[rt.Src][spf] = append(receiverToSpooferToProbe[rt.Src][spf], []string{rt.LastHop(), rt.LastHop(), adj, adj, dummyIP})
			}
		}
		// if we haven't already decided whether it is responsive,
		// we'll set it to false, then change to true if we get one
		rt.initTsSrcToHopToResponseive(rt.Src)
		if _, ok := rt.tsSrcToHopToResponsive[rt.Src][rt.LastHop()]; !ok {
			rt.tsSrcToHopToResponsive[rt.Src][rt.LastHop()] = 1
		}
	}

	type pair struct {
		src, dst string
	}
	type triplet struct {
		src, dst, vp string
	}
	type tripletTs struct {
		src, dst, tsip string
	}
	var revHopsSrcDstToRevSeg = make(map[pair][]Segment)
	var linuxBugToCheckSrcDstVpToRevHops = make(map[triplet][]string)
	var destDoesNotStamp []tripletTs

	processTSCheckForRevHop := func(src, vp string, p *datamodel.Ping) {
		rt.debug("Processing TS: ", p)
		dsts, _ := util.Int32ToIPString(p.Dst)
		segClass := "SpoofTSAdjRevSegment"
		if vp == "non_spoofed" {
			checksrctohoptosendspoofedmagic(src)
			rt.tsSrcToHopToSendSpoofed[src][dsts] = false
			segClass = "TSAdjRevSegment"
		}
		rt.initTsSrcToHopToResponseive(src)
		rt.tsSrcToHopToResponsive[src][dsts] = 1
		rps := p.GetResponses()
		if len(rps) > 0 {
			rt.debug("Response ", rps[0].Tsandaddr)
		}
		if len(rps) > 0 && len(rps[0].Tsandaddr) > 2 {
			ts1 := rps[0].Tsandaddr[0]
			ts2 := rps[0].Tsandaddr[1]
			ts3 := rps[0].Tsandaddr[2]
			if ts3.Ts != 0 {
				ss, _ := util.Int32ToIPString(rps[0].Tsandaddr[2].Ip)
				var seg Segment
				if segClass == "SpoofTSAdjRevSegment" {
					seg = NewSpoofTSAdjRevSegment([]string{ss}, src, dsts, vp, false)
				} else {
					seg = NewTSAdjRevSegment([]string{ss}, src, dsts, false)
				}
				revHopsSrcDstToRevSeg[pair{src: src, dst: dsts}] = []Segment{seg}
			} else if ts2.Ts != 0 {
				if ts2.Ts-ts1.Ts > 3 || ts2.Ts < ts1.Ts {
					// if 2nd slot is stamped with an increment from 1st, rev hop
					ts2ips, _ := util.Int32ToIPString(ts2.Ip)
					linuxBugToCheckSrcDstVpToRevHops[triplet{src: src, dst: dsts, vp: vp}] = append(linuxBugToCheckSrcDstVpToRevHops[triplet{src: src, dst: dsts, vp: vp}], ts2ips)
				}
			} else if ts1.Ts == 0 {
				// if dst responds, does not stamp, can try advanced techniques
				ts2ips, _ := util.Int32ToIPString(ts2.Ip)
				rt.tsDstToStampsZero[dsts] = true
				destDoesNotStamp = append(destDoesNotStamp, tripletTs{src: src, dst: dsts, tsip: ts2ips})
			} else {
				rt.debug("TS probe is ", vp, p, "no reverse hop found")
			}
		}
	}
	rt.debug("tsToIssueSrcToProbe ", tsToIssueSrcToProbe)
	if len(tsToIssueSrcToProbe) > 0 {
		// there should be a uniq thing here but I need to figure out how to do it
		for src, probes := range tsToIssueSrcToProbe {
			for _, probe := range probes {
				checksrctohoptosendspoofedmagic(src)
				if _, ok := rt.tsSrcToHopToSendSpoofed[src][probe[0]]; ok {
					continue
				}
				// set it to true, then change it to false if we get a response
				rt.tsSrcToHopToSendSpoofed[src][probe[0]] = true
			}
		}
		rt.debug("Issuing TS probes")
		rt.issueTimestamps(tsToIssueSrcToProbe, processTSCheckForRevHop)
		rt.debug("Done issuing TS probes ", tsToIssueSrcToProbe)
		for src, probes := range tsToIssueSrcToProbe {
			for _, probe := range probes {
				// if we got a reply, would have set sendspoofed to false
				// so it is still true, we need to try to find a spoofer
				checksrctohoptosendspoofedmagic(src)
				if rt.tsSrcToHopToSendSpoofed[src][probe[0]] {
					mySpoofers := rt.getTimestampSpoofers(src, probe[0])
					for _, sp := range mySpoofers {
						rt.debug("Adding spoofed TS probe to send")
						checkMapMagic(src, sp)
						receiverToSpooferToProbe[src][sp] = append(receiverToSpooferToProbe[src][sp], probe)
					}
					// if we haven't already decided whether it is responsive
					// we'll set it to false, then change to true if we get one
					if _, ok := rt.tsSrcToHopToResponsive[src][probe[0]]; !ok {
						rt.initTsSrcToHopToResponseive(src)
						rt.tsSrcToHopToResponsive[src][probe[0]] = 1
					}
				}
			}
		}
	}
	rt.debug("receiverToSpooferToProbe: ", receiverToSpooferToProbe)
	if len(receiverToSpooferToProbe) > 0 {
		rt.issueSpoofedTimestamps(receiverToSpooferToProbe, processTSCheckForRevHop)
	}
	if len(linuxBugToCheckSrcDstVpToRevHops) > 0 {
		var linuxChecksSrcToProbe = make(map[string][][]string)
		var linuxChecksSpoofedReceiverToSpooferToProbe = make(map[string]map[string][][]string)
		for sdvp := range linuxBugToCheckSrcDstVpToRevHops {
			p := []string{sdvp.dst, sdvp.dst, dummyIP, dummyIP}
			if sdvp.vp == "non_spoofed" {
				linuxChecksSrcToProbe[sdvp.src] = append(linuxChecksSrcToProbe[sdvp.src], p)
			} else {
				if val, ok := linuxChecksSpoofedReceiverToSpooferToProbe[sdvp.src]; ok {
					val[sdvp.vp] = append(linuxChecksSpoofedReceiverToSpooferToProbe[sdvp.src][sdvp.vp], p)
				} else {
					linuxChecksSpoofedReceiverToSpooferToProbe[sdvp.src] = make(map[string][][]string)
					linuxChecksSpoofedReceiverToSpooferToProbe[sdvp.src][sdvp.vp] = append(linuxChecksSpoofedReceiverToSpooferToProbe[sdvp.src][sdvp.vp], p)
				}
			}
		}
		// once again leaving out a check for uniqness
		processTSCheckForLinuxBug := func(src, vp string, p *datamodel.Ping) {
			dsts, _ := util.Int32ToIPString(p.Dst)
			rps := p.GetResponses()
			ts2 := rps[0].Tsandaddr[1]

			segClass := "SpoofTSAdjRevSegment"
			// if I got a response, must not be filtering, so dont need to use spoofing
			if vp == "non_spoofed" {
				checksrctohoptosendspoofedmagic(src)
				rt.tsSrcToHopToSendSpoofed[src][dsts] = false
				segClass = "TSAdjRevSegment"
			}
			if ts2.Ts != 0 {
				rt.debug("TS probe is ", vp, p, "linux bug")
				// TODO keep track of linux bugs
				// at least once, i observed a bug not stamp one probe, so
				// this is important, probably then want to do the checks
				// for revhops after all spoofers that are trying have tested
				// for linux bugs
			} else {
				rt.debug("TS probe is ", vp, p, "not linux bug")
				for _, revhop := range linuxBugToCheckSrcDstVpToRevHops[triplet{src: src, dst: dsts, vp: vp}] {

					var seg Segment
					if segClass == "TSAdjRevSegment" {
						seg = NewTSAdjRevSegment([]string{revhop}, src, dsts, false)
					} else {
						seg = NewSpoofTSAdjRevSegment([]string{revhop}, src, dsts, vp, false)
					}
					revHopsSrcDstToRevSeg[pair{src: src, dst: dsts}] = []Segment{seg}
				}
			}
		}
		rt.issueTimestamps(linuxChecksSrcToProbe, processTSCheckForLinuxBug)
		rt.issueSpoofedTimestamps(linuxChecksSpoofedReceiverToSpooferToProbe, processTSCheckForLinuxBug)
	}
	receiverToSpooferToProbe = make(map[string]map[string][][]string)
	for _, probe := range destDoesNotStamp {
		spoofers := rt.getTimestampSpoofers(probe.src, probe.dst)
		for _, s := range spoofers {
			checkMapMagic(probe.src, s)
			receiverToSpooferToProbe[probe.src][s] = append(receiverToSpooferToProbe[probe.src][s], []string{probe.dst, probe.tsip, probe.tsip, probe.tsip, probe.tsip})
		}
	}
	// if I get the response, need to then do the non-spoofed version
	// for that, I can get everything I need to know from the probe
	// send the duplicates
	// then, for each of those that get responses but don't stamp
	// I can delcare it a revhop-- I just need to know which src to declare it
	// for
	// so really what I ened is a  map from VP,dst,adj to the list of
	// sources/revtrs waiting for it
	destDoesNotStampToVerifySpooferToProbe := make(map[string][][]string)
	vpDstAdjToInterestedSrcs := make(map[tripletTs][]string)
	processTSDestDoesNotStamp := func(src, vp string, p *datamodel.Ping) {
		dsts, _ := util.Int32ToIPString(p.Dst)
		rps := p.GetResponses()
		ts1 := rps[0].Tsandaddr[0]
		ts2 := rps[0].Tsandaddr[1]
		ts4 := rps[0].Tsandaddr[3]
		// if 2 stamps, we assume one was forward, one was reverse
		// if 1 or 4, we need to verify it was reverse
		// 3 should not happend according to justine?
		if ts2.Ts != 0 && ts4.Ts == 0 {
			// declare reverse hop
			ts2ips, _ := util.Int32ToIPString(ts2.Ts)
			revHopsSrcDstToRevSeg[pair{src: src, dst: dsts}] = []Segment{NewSpoofTSAdjRevSegmentTSZeroDoubleStamp([]string{ts2ips}, src, dsts, vp, false)}
			rt.debug("TS Probe is ", vp, p, "reverse hop from dst that stamps 0!")
		} else if ts1.Ts != 0 {
			rt.debug("TS probe is ", vp, p, "dst does not stamp, but spoofer ", vp, "got a stamp")
			ts1ips, _ := util.Int32ToIPString(ts1.Ip)
			destDoesNotStampToVerifySpooferToProbe[vp] = append(destDoesNotStampToVerifySpooferToProbe[vp], []string{dsts, ts1ips, ts1ips, ts1ips, ts1ips})
			// store something
			vpDstAdjToInterestedSrcs[tripletTs{src: vp, dst: dsts, tsip: ts1ips}] = append(vpDstAdjToInterestedSrcs[tripletTs{src: vp, dst: dsts, tsip: ts1ips}], src)
		} else {
			rt.debug("TS probe is ", vp, p, "no reverse hop for dst that stamps 0")
		}
	}
	if len(destDoesNotStamp) > 0 {
		rt.issueSpoofedTimestamps(receiverToSpooferToProbe, processTSDestDoesNotStamp)
	}

	// if you don't get a response, add it with false
	// then at the end
	if len(destDoesNotStampToVerifySpooferToProbe) > 0 {
		for vp, probes := range destDoesNotStampToVerifySpooferToProbe {
			probes = append(probes, probes...)
			probes = append(probes, probes...)
			destDoesNotStampToVerifySpooferToProbe[vp] = probes
		}
		maybeRevhopVPDstAdjToBool := make(map[tripletTs]bool)
		revHopsVPDstToRevSeg := make(map[pair][]Segment)
		processTSDestDoesNotStampToVerify := func(src, vp string, p *datamodel.Ping) {
			dsts, _ := util.Int32ToIPString(p.Dst)
			rps := p.GetResponses()
			ts1 := rps[0].Tsandaddr[0]
			ts1ips, _ := util.Int32ToIPString(ts1.Ip)
			if ts1.Ts == 0 {
				rt.debug("Reverse hop! TS probe is ", vp, p, "dst does not stamp, but spoofer", vp, "got a stamp and didn't direclty")
				maybeRevhopVPDstAdjToBool[tripletTs{src: src, dst: dsts, tsip: ts1ips}] = true
			} else {
				del := tripletTs{src: src, dst: dsts, tsip: ts1ips}
				for key := range vpDstAdjToInterestedSrcs {
					if key == del {
						delete(vpDstAdjToInterestedSrcs, key)
					}
				}
				rt.debug("Can't verify reverse hop! TS probe is ", vp, p, "potential hop stamped on non-spoofed path for VP")
			}
		}
		rt.debug("Issuing to verify for dest does not stamp")
		rt.issueTimestamps(destDoesNotStampToVerifySpooferToProbe, processTSDestDoesNotStampToVerify)
		for k := range maybeRevhopVPDstAdjToBool {
			for _, origsrc := range vpDstAdjToInterestedSrcs[tripletTs{src: k.src, dst: k.dst, tsip: k.tsip}] {
				revHopsVPDstToRevSeg[pair{src: origsrc, dst: k.dst}] = append(revHopsVPDstToRevSeg[pair{src: origsrc, dst: k.dst}], NewSpoofTSAdjRevSegmentTSZeroDoubleStamp([]string{k.tsip}, origsrc, k.dst, k.src, false))
			}

		}
	}

	// Ping V/S->R:R',R',R',R'
	// (i think, but justine has it nested differently) if stamp twice,
	// declare rev hop, # else if i get one:
	// if i get responses:
	// n? times: Ping V/V->R:R',R',R',R'
	// if (never stamps) // could be a false positive, maybe R' just didn't
	// feel like stamping this time
	// return R'
	// if stamps more thane once, decl,
	if segments, ok := revHopsSrcDstToRevSeg[pair{src: rt.Src, dst: rt.LastHop()}]; ok {
		if rt.AddSegments(segments) {
			return nil
		}
	}
	return ErrNoHopFound
}

func (rt *ReverseTraceroute) initTsSrcToHopToResponseive(s string) {
	if _, ok := rt.tsSrcToHopToResponsive[s]; ok {
		return
	}
	rt.tsSrcToHopToResponsive[s] = make(map[string]int)
}

func (rt *ReverseTraceroute) issueTimestamps(issue map[string][][]string, fn func(string, string, *datamodel.Ping)) error {
	rt.debug("Issuing timestamps")
	var pings []*datamodel.PingMeasurement
	for src, probes := range issue {
		srcip, _ := util.IPStringToInt32(src)
		for _, probe := range probes {
			dstip, _ := util.IPStringToInt32(probe[0])
			var tsString bytes.Buffer
			tsString.WriteString("tsprespec=")
			for i, p := range probe {
				if i == 0 {
					continue
				}
				tsString.WriteString(p)
				if len(probe)-1 != i {
					tsString.WriteString(",")
				}
			}
			tss := tsString.String()
			rt.debug("tss string: ", tss)
			if iputil.IsPrivate(net.ParseIP(probe[0])) {
				continue
			}
			p := &datamodel.PingMeasurement{
				Src:        srcip,
				Dst:        dstip,
				TimeStamp:  tss,
				Timeout:    10,
				Count:      "1",
				CheckDb:    true,
				CheckCache: true,
				Staleness:  rt.Staleness,
			}
			pings = append(pings, p)
		}
	}
	rt.ProbeCount["ts"] += len(pings)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	st, err := rt.cl.Ping(ctx, &datamodel.PingArg{Pings: pings})
	if err != nil {
		rt.error(err)
		return err
	}
	for {
		pr, err := st.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			rt.error(err)
			return err
		}
		srcs, _ := util.Int32ToIPString(pr.Src)
		fn(srcs, "non_spoofed", pr)
	}
	return nil
}

func (rt *ReverseTraceroute) issueSpoofedTimestamps(issue map[string]map[string][][]string, fn func(string, string, *datamodel.Ping)) error {
	rt.debug("Issuing spoofed timestamps")
	var pings []*datamodel.PingMeasurement
	for reciever, spooferToProbes := range issue {
		recip, _ := util.IPStringToInt32(reciever)
		for spoofer, probes := range spooferToProbes {
			spoofip, _ := util.IPStringToInt32(spoofer)
			for _, probe := range probes {
				dstip, _ := util.IPStringToInt32(probe[0])
				var tsString bytes.Buffer
				for i, p := range probe {
					if i == 0 {
						continue
					}
					tsString.WriteString(p)
					if i != len(probe)-1 {
						tsString.WriteString(",")
					}
				}
				if iputil.IsPrivate(net.ParseIP(probe[0])) {
					continue
				}
				p := &datamodel.PingMeasurement{
					Src:         spoofip,
					Spoof:       true,
					Dst:         dstip,
					SpooferAddr: recip,
					Timeout:     40,
					Count:       "1",
					CheckDb:     true,
					CheckCache:  true,
					Staleness:   rt.Staleness,
				}
				pings = append(pings, p)
			}
		}
	}
	rt.ProbeCount["spoof-ts"] = len(pings)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	st, err := rt.cl.Ping(ctx, &datamodel.PingArg{Pings: pings})
	if err != nil {
		rt.error(err)
		return err
	}
	for {
		pr, err := st.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			rt.error(err)
			return err
		}
		srcs, _ := util.Int32ToIPString(pr.Src)
		vp, _ := util.Int32ToIPString(pr.SpoofedFrom)
		fn(srcs, vp, pr)
	}
	return nil
}
