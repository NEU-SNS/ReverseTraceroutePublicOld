package runner

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"reflect"
	"sync"
	"time"

	at "github.com/NEU-SNS/ReverseTraceroute/atlas/client"
	apb "github.com/NEU-SNS/ReverseTraceroute/atlas/pb"
	"github.com/NEU-SNS/ReverseTraceroute/controller/client"
	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/clustermap"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/ip_utils"
	rt "github.com/NEU-SNS/ReverseTraceroute/revtr/reverse_traceroute"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/types"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	"github.com/NEU-SNS/ReverseTraceroute/util/string"
	vpservice "github.com/NEU-SNS/ReverseTraceroute/vpservice/client"
	"golang.org/x/net/context"
)

type optionSet struct {
	ctx context.Context
	cm  clustermap.ClusterMap
	cl  client.Client
	at  at.Atlas
	vps vpservice.VPSource
	as  types.AdjacencySource
}

// RunOption configures how run will behave
type RunOption func(*optionSet)

// WithContext runs revtrs with the context c
func WithContext(c context.Context) RunOption {
	return func(os *optionSet) {
		os.ctx = c
	}
}

// WithClusterMap runs the revts with the clustermap cm
func WithClusterMap(cm clustermap.ClusterMap) RunOption {
	return func(os *optionSet) {
		os.cm = cm
	}
}

// WithClient runs the revts with the client cl
func WithClient(cl client.Client) RunOption {
	return func(os *optionSet) {
		os.cl = cl
	}
}

// WithAtlas runs the revtrs with the atlas at
func WithAtlas(at at.Atlas) RunOption {
	return func(os *optionSet) {
		os.at = at
	}
}

// WithVPSource runs the revtrs with the vpservice vps
func WithVPSource(vps vpservice.VPSource) RunOption {
	return func(os *optionSet) {
		os.vps = vps
	}
}

// WithAdjacencySource runs the revtrs with the adjacnecy source as
func WithAdjacencySource(as types.AdjacencySource) RunOption {
	return func(os *optionSet) {
		os.as = as
	}
}

type runner struct {
}

// Runner is the interface for running reverse traceroutes
type Runner interface {
	Run([]*rt.ReverseTraceroute, ...RunOption) <-chan *rt.ReverseTraceroute
}

// New creates a new Runner
func New() Runner {
	return new(runner)
}

func (r *runner) Run(revtrs []*rt.ReverseTraceroute,
	opts ...RunOption) <-chan *rt.ReverseTraceroute {
	optset := &optionSet{}
	for _, opt := range opts {
		opt(optset)
	}
	if optset.ctx == nil {
		optset.ctx = context.Background()
	}
	rc := make(chan *rt.ReverseTraceroute, len(revtrs))
	batch := &rtBatch{}
	batch.opts = optset
	batch.wg = &sync.WaitGroup{}
	batch.wg.Add(len(revtrs))
	for _, revtr := range revtrs {
		log.Debug("Running ", revtr)
		go batch.run(revtr, rc)
	}
	go func() {
		batch.wg.Wait()
		close(rc)
	}()
	return rc
}

type step func(*rt.ReverseTraceroute) step

type rtBatch struct {
	opts *optionSet
	wg   *sync.WaitGroup
}

func (b *rtBatch) initialStep(revtr *rt.ReverseTraceroute) step {
	if revtr.BackoffEndhost {
		return b.backoffEndhost
	}
	return b.trToSource
}

func (b *rtBatch) backoffEndhost(revtr *rt.ReverseTraceroute) step {
	next := b.assumeSymmetric(revtr)
	if revtr.Reaches(b.opts.cm) {
		revtr.StopReason = rt.Trivial
		revtr.EndTime = time.Now()
	}
	if next == nil {
	}
	log.Debug("Done backing off")
	return next
}

func (b *rtBatch) trToSource(revtr *rt.ReverseTraceroute) step {
	revtr.Stats.TRToSrcRoundCount++
	start := time.Now()
	defer func() {
		done := time.Now()
		dur := done.Sub(start)
		revtr.Stats.TRToSrcDuration += dur
	}()
	var addrs []uint32
	for _, hop := range revtr.CurrPath().LastSeg().Hops() {
		addr, _ := util.IPStringToInt32(hop)
		if iputil.IsPrivate(net.ParseIP(hop)) {
			continue
		}
		addrs = append(addrs, addr)
	}
	hops, tokens, err := intersectingTraceroute(revtr.Src, revtr.Dst, addrs,
		revtr.Staleness, b.opts.at, b.opts.cm)
	if err != nil {
		// and error occured trying to find intersecting traceroutes
		// move on to the next step
		return b.recordRoute
	}
	if tokens != nil {
		revtr.Tokens = tokens
		// received tokens, move on to next step and try later
		return b.recordRoute
	}
	if len(hops.hops) == 0 {
		revtr.Tokens = tokens
		// no hops found move on
		return b.recordRoute
	}
	log.Debug("Creating TRToSrc seg: ", hops, " ", revtr.Src, " ", hops.addr)
	segment := rt.NewTrtoSrcRevSegment(hops.hops, revtr.Src, hops.addr)
	if !revtr.AddBackgroundTRSegment(segment, b.opts.cm) {
		panic("Failed to add TR segment. That's not possible")
	}
	if revtr.Reaches(b.opts.cm) {
		return nil
	}
	panic("Added a TR to source but the revtr didn't reach")
}

type ipstr uint32

func (ips ipstr) String() string {
	s, _ := util.Int32ToIPString(uint32(ips))
	return s
}

func (b *rtBatch) recordRoute(revtr *rt.ReverseTraceroute) step {
	revtr.Stats.RRRoundCount++
	start := time.Now()
	defer func() {
		done := time.Now()
		dur := done.Sub(start)
		revtr.Stats.RRDuration += dur
	}()
	for {
		vps, target := revtr.GetRRVPs(revtr.LastHop(), b.opts.vps)
		if len(vps) == 0 {
			// No vps left, move on to TS
			return b.timestamp
		}
		if stringutil.InArray(vps, "non_spoofed") {
			revtr.Stats.RRProbes++
			rr, err := issueRR(revtr.Src, target,
				revtr.Staleness, b.opts.cl, b.opts.cm)
			if err != nil {
				// Couldn't perform RR measurements
				// move on to try next group for now
				continue
			}
			var segs []rt.Segment
			for i, hop := range rr {
				if hop == "0.0.0.0" {
					continue
				}
				segs = append(segs, rt.NewRRRevSegment(rr[:i+1], revtr.Src, target))
			}
			if !revtr.AddSegments(segs, b.opts.cm) {
				// Failed to anything from the RR hops
				// move on to next group
				continue
			}
			// RR get up hops
			// test if it reaches we're done
			if revtr.Reaches(b.opts.cm) {
				return nil
			}
			// Got hops but we didn't reach
			// try adding from the traceroutes
			// the atlas ran
			// if they don't complete the revtr
			// we're back to the start with trToSource
			return b.checkbgTRs(revtr, b.trToSource)
		}
		revtr.Stats.SpoofedRRProbes += len(vps)
		rrs, err := issueSpoofedRR(revtr.Src, target, vps,
			revtr.Staleness, b.opts.cl, b.opts.cm)
		if err != nil {
			log.Error(err)
		}
		// we got some responses, even if there arent any addresses
		// in the responses make the target as responsive
		if len(rrs) > 0 {
			revtr.MarkResponsiveRRSpoofer(target)
		} else {
			// track unresponsiveness
			revtr.AddUnresponsiveRRSpoofer(target, len(vps))
		}
		var segs []rt.Segment
		// create segments for all the hops in the responses
		for _, rr := range rrs {
			log.Error("Creating segment for ", rr)
			for i, hop := range rr.hops {
				if hop == "0.0.0.0" {
					continue
				}
				segs = append(segs,
					rt.NewSpoofRRRevSegment(rr.hops[:i+1], revtr.Src, target, rr.vp))
			}
		}
		if len(segs) == 0 {
			// so segments created
			continue
		}
		// try to add them
		if !revtr.AddSegments(segs, b.opts.cm) {
			//Couldn't add anything
			// continue
			continue
		}
		// RR get up hops
		// test if it reaches we're done
		if revtr.Reaches(b.opts.cm) {
			return nil
		}

		// Got hops but we didn't reach
		// try adding from the traceroutes
		// the atlas ran
		// if they don't complete the revtr
		// timestamp is next
		return b.checkbgTRs(revtr, b.timestamp)
	}
}

const (
	dummyIP = "128.208.3.77"
)

func (b *rtBatch) timestamp(revtr *rt.ReverseTraceroute) step {
	revtr.Stats.TSRoundCount++
	start := time.Now()
	defer func() {
		done := time.Now()
		dur := done.Sub(start)
		revtr.Stats.TSDuration += dur
	}()
	log.Debug("Trying Timestamp")
	var receiverToSpooferToProbe = make(map[string]map[string][][]string)
	checkMapMagic := func(f, s string) {
		if _, ok := receiverToSpooferToProbe[f]; !ok {
			receiverToSpooferToProbe[f] = make(map[string][][]string)
		}
	}
	checksrctohoptosendspoofedmagic := func(f string) {
		if _, ok := revtr.TSSrcToHopToSendSpoofed[f]; !ok {
			revtr.TSSrcToHopToSendSpoofed[f] = make(map[string]bool)
		}
	}
	target := revtr.LastHop()
	if !revtr.TSIsResponsive(target) {
		// the target is not responsive to ts
		// move on to next step
		return b.backgroundTRS
	}
	for {
		adjs := revtr.GetTSAdjacents(target, b.opts.as)
		if len(adjs) == 0 {
			log.Debug("No adjacents for: ", target)
			// No adjacencies left, move on to the next step
			return b.backgroundTRS
		}
		var dstsDoNotStamp [][]string
		var tsToIssueSrcToProbe = make(map[string][][]string)
		if revtr.TSDstToStampsZero[target] {
			log.Debug("tsDstToStampsZero wtf")
			for _, adj := range adjs {
				dstsDoNotStamp = append(dstsDoNotStamp,
					[]string{revtr.Src, target, adj})
			}
		} else if !revtr.TSSrcToHopToSendSpoofed[revtr.Src][target] {
			log.Debug("Adding Spoofed TS to send")
			for _, adj := range adjs {
				tsToIssueSrcToProbe[revtr.Src] = append(
					tsToIssueSrcToProbe[revtr.Src],
					[]string{revtr.LastHop(),
						revtr.LastHop(),
						adj, adj, dummyIP})
			}
		} else {
			log.Debug("TS Non of the above")
			spfs := revtr.GetTimestampSpoofers(revtr.Src, revtr.LastHop(), b.opts.vps)
			if len(spfs) == 0 {
				log.Debug("no spoofers left")
				return b.backgroundTRS
			}
			for _, adj := range adjs {
				for _, spf := range spfs {
					checkMapMagic(revtr.Src, spf)
					receiverToSpooferToProbe[revtr.Src][spf] = append(
						receiverToSpooferToProbe[revtr.Src][spf],
						[]string{revtr.LastHop(),
							revtr.LastHop(),
							adj, adj, dummyIP})
				}
			}
			// if we haven't already decided whether it is responsive,
			// we'll set it to false, then change to true if we get one
			revtr.TSSetUnresponsive(target)
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
		var revHopsSrcDstToRevSeg = make(map[pair][]rt.Segment)
		var linuxBugToCheckSrcDstVpToRevHops = make(map[triplet][]string)
		var destDoesNotStamp []tripletTs

		processTSCheckForRevHop := func(src, vp string, p *datamodel.Ping) {
			log.Debug("Processing TS: ", p)
			dsts, _ := util.Int32ToIPString(p.Dst)
			segClass := "SpoofTSAdjRevSegment"
			if vp == "non_spoofed" {
				checksrctohoptosendspoofedmagic(src)
				revtr.TSSrcToHopToSendSpoofed[src][dsts] = false
				segClass = "TSAdjRevSegment"
			}
			revtr.TSSetUnresponsive(dsts)
			rps := p.GetResponses()
			if len(rps) > 0 {
				log.Debug("Response ", rps[0].Tsandaddr)
			}
			if len(rps) > 0 && len(rps[0].Tsandaddr) > 2 {
				ts1 := rps[0].Tsandaddr[0]
				ts2 := rps[0].Tsandaddr[1]
				ts3 := rps[0].Tsandaddr[2]
				if ts3.Ts != 0 {
					ss, _ := util.Int32ToIPString(rps[0].Tsandaddr[2].Ip)
					var seg rt.Segment
					if segClass == "SpoofTSAdjRevSegment" {
						seg = rt.NewSpoofTSAdjRevSegment([]string{ss}, src, dsts, vp, false)
					} else {
						seg = rt.NewTSAdjRevSegment([]string{ss}, src, dsts, false)
					}
					revHopsSrcDstToRevSeg[pair{src: src, dst: dsts}] = []rt.Segment{seg}
				} else if ts2.Ts != 0 {
					if ts2.Ts-ts1.Ts > 3 || ts2.Ts < ts1.Ts {
						// if 2nd slot is stamped with an increment from 1st, rev hop
						ts2ips, _ := util.Int32ToIPString(ts2.Ip)
						linuxBugToCheckSrcDstVpToRevHops[triplet{src: src, dst: dsts, vp: vp}] = append(linuxBugToCheckSrcDstVpToRevHops[triplet{src: src, dst: dsts, vp: vp}], ts2ips)
					}
				} else if ts1.Ts == 0 {
					// if dst responds, does not stamp, can try advanced techniques
					ts2ips, _ := util.Int32ToIPString(ts2.Ip)
					revtr.TSDstToStampsZero[dsts] = true
					destDoesNotStamp = append(destDoesNotStamp, tripletTs{src: src, dst: dsts, tsip: ts2ips})
				} else {
					log.Debug("TS probe is ", vp, p, "no reverse hop found")
				}
			}
		}
		log.Debug("tsToIssueSrcToProbe ", tsToIssueSrcToProbe)
		if len(tsToIssueSrcToProbe) > 0 {
			// there should be a uniq thing here but I need to figure out how to do it
			for src, probes := range tsToIssueSrcToProbe {
				for _, probe := range probes {
					checksrctohoptosendspoofedmagic(src)
					if _, ok := revtr.TSSrcToHopToSendSpoofed[src][probe[0]]; ok {
						continue
					}
					// set it to true, then change it to false if we get a response
					revtr.TSSrcToHopToSendSpoofed[src][probe[0]] = true
				}
			}
			log.Debug("Issuing TS probes")
			revtr.Stats.TSProbes += len(tsToIssueSrcToProbe)
			err := issueTimestamps(tsToIssueSrcToProbe, processTSCheckForRevHop,
				revtr.Staleness, b.opts.cl)
			if err != nil {
				log.Error(err)
			}
			log.Debug("Done issuing TS probes ", tsToIssueSrcToProbe)
			for src, probes := range tsToIssueSrcToProbe {
				for _, probe := range probes {
					// if we got a reply, would have set sendspoofed to false
					// so it is still true, we need to try to find a spoofer
					checksrctohoptosendspoofedmagic(src)
					if revtr.TSSrcToHopToSendSpoofed[src][probe[0]] {
						mySpoofers := revtr.GetTimestampSpoofers(src, probe[0], b.opts.vps)
						for _, sp := range mySpoofers {
							log.Debug("Adding spoofed TS probe to send")
							checkMapMagic(src, sp)
							receiverToSpooferToProbe[src][sp] = append(receiverToSpooferToProbe[src][sp], probe)
						}
						// if we haven't already decided whether it is responsive
						// we'll set it to false, then change to true if we get one
						revtr.TSSetUnresponsive(probe[0])
					}
				}
			}
		}
		log.Debug("receiverToSpooferToProbe: ", receiverToSpooferToProbe)
		for _, val := range receiverToSpooferToProbe {
			revtr.Stats.SpoofedTSProbes += len(val)
		}
		if len(receiverToSpooferToProbe) > 0 {
			err := issueSpoofedTimestamps(receiverToSpooferToProbe,
				processTSCheckForRevHop, revtr.Staleness, b.opts.cl)
			if err != nil {
				log.Error(err)
			}
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
					revtr.TSSrcToHopToSendSpoofed[src][dsts] = false
					segClass = "TSAdjRevSegment"
				}
				if ts2.Ts != 0 {
					log.Debug("TS probe is ", vp, p, "linux bug")
					// TODO keep track of linux bugs
					// at least once, i observed a bug not stamp one probe, so
					// this is important, probably then want to do the checks
					// for revhops after all spoofers that are trying have tested
					// for linux bugs
				} else {
					log.Debug("TS probe is ", vp, p, "not linux bug")
					for _, revhop := range linuxBugToCheckSrcDstVpToRevHops[triplet{src: src, dst: dsts, vp: vp}] {

						var seg rt.Segment
						if segClass == "TSAdjRevSegment" {
							seg = rt.NewTSAdjRevSegment([]string{revhop}, src, dsts, false)
						} else {
							seg = rt.NewSpoofTSAdjRevSegment([]string{revhop}, src, dsts, vp, false)
						}
						revHopsSrcDstToRevSeg[pair{src: src, dst: dsts}] = []rt.Segment{seg}
					}
				}
			}
			revtr.Stats.TSProbes += len(tsToIssueSrcToProbe)
			err := issueTimestamps(linuxChecksSrcToProbe,
				processTSCheckForLinuxBug, revtr.Staleness, b.opts.cl)
			if err != nil {
				log.Error(err)
			}
			for _, val := range receiverToSpooferToProbe {
				revtr.Stats.SpoofedTSProbes += len(val)
			}
			err = issueSpoofedTimestamps(linuxChecksSpoofedReceiverToSpooferToProbe,
				processTSCheckForLinuxBug, revtr.Staleness, b.opts.cl)
			if err != nil {
				log.Error(err)
			}
		}
		receiverToSpooferToProbe = make(map[string]map[string][][]string)
		for _, probe := range destDoesNotStamp {
			spoofers := revtr.GetTimestampSpoofers(probe.src, probe.dst, b.opts.vps)
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
				revHopsSrcDstToRevSeg[pair{src: src, dst: dsts}] = []rt.Segment{rt.NewSpoofTSAdjRevSegmentTSZeroDoubleStamp([]string{ts2ips}, src, dsts, vp, false)}
				log.Debug("TS Probe is ", vp, p, "reverse hop from dst that stamps 0!")
			} else if ts1.Ts != 0 {
				log.Debug("TS probe is ", vp, p, "dst does not stamp, but spoofer ", vp, "got a stamp")
				ts1ips, _ := util.Int32ToIPString(ts1.Ip)
				destDoesNotStampToVerifySpooferToProbe[vp] = append(destDoesNotStampToVerifySpooferToProbe[vp], []string{dsts, ts1ips, ts1ips, ts1ips, ts1ips})
				// store something
				vpDstAdjToInterestedSrcs[tripletTs{src: vp, dst: dsts, tsip: ts1ips}] = append(vpDstAdjToInterestedSrcs[tripletTs{src: vp, dst: dsts, tsip: ts1ips}], src)
			} else {
				log.Debug("TS probe is ", vp, p, "no reverse hop for dst that stamps 0")
			}
		}
		if len(destDoesNotStamp) > 0 {
			err := issueSpoofedTimestamps(receiverToSpooferToProbe,
				processTSDestDoesNotStamp,
				revtr.Staleness, b.opts.cl)
			if err != nil {
				log.Error(err)
			}
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
			revHopsVPDstToRevSeg := make(map[pair][]rt.Segment)
			processTSDestDoesNotStampToVerify := func(src, vp string, p *datamodel.Ping) {
				dsts, _ := util.Int32ToIPString(p.Dst)
				rps := p.GetResponses()
				ts1 := rps[0].Tsandaddr[0]
				ts1ips, _ := util.Int32ToIPString(ts1.Ip)
				if ts1.Ts == 0 {
					log.Debug("Reverse hop! TS probe is ", vp, p, "dst does not stamp, but spoofer", vp, "got a stamp and didn't direclty")
					maybeRevhopVPDstAdjToBool[tripletTs{src: src, dst: dsts, tsip: ts1ips}] = true
				} else {
					del := tripletTs{src: src, dst: dsts, tsip: ts1ips}
					for key := range vpDstAdjToInterestedSrcs {
						if key == del {
							delete(vpDstAdjToInterestedSrcs, key)
						}
					}
					log.Debug("Can't verify reverse hop! TS probe is ", vp, p, "potential hop stamped on non-spoofed path for VP")
				}
			}
			log.Debug("Issuing to verify for dest does not stamp")
			err := issueTimestamps(destDoesNotStampToVerifySpooferToProbe,
				processTSDestDoesNotStampToVerify,
				revtr.Staleness,
				b.opts.cl)
			if err != nil {
				log.Error(err)
			}
			for k := range maybeRevhopVPDstAdjToBool {
				for _, origsrc := range vpDstAdjToInterestedSrcs[tripletTs{src: k.src, dst: k.dst, tsip: k.tsip}] {
					revHopsVPDstToRevSeg[pair{src: origsrc, dst: k.dst}] =
						append(revHopsVPDstToRevSeg[pair{src: origsrc, dst: k.dst}],
							rt.NewSpoofTSAdjRevSegmentTSZeroDoubleStamp(
								[]string{k.tsip}, origsrc, k.dst, k.src, false))
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
		if segments, ok := revHopsSrcDstToRevSeg[pair{src: revtr.Src, dst: revtr.LastHop()}]; ok {
			if revtr.AddSegments(segments, b.opts.cm) {
				// added a segment
				// if it reaches we're done
				if revtr.Reaches(b.opts.cm) {
					return nil
				}
				return b.checkbgTRs(revtr, b.trToSource)
			}
		}
		// continue to next set
	}
}

// this is used to check background traceroute.
// it is different than the step backgroundTRS
// This checks for background trs that were issued for
// current round. It is called after a step finds hops
// the step next is returned if the background trs dont
// reach
func (b *rtBatch) checkbgTRs(revtr *rt.ReverseTraceroute, next step) step {
	ns := b.backgroundTRS(revtr)
	// if ns is nil backgroundTRS reaches
	// and we're done
	if ns == nil {
		return ns
	}
	// if ns is not nil, we're not done
	// but the background traceroutes didn't
	// finish or they weren't useful
	// return the next step
	return next
}

func (b *rtBatch) backgroundTRS(revtr *rt.ReverseTraceroute) step {
	revtr.Stats.BackgroundTRSRoundCount++
	start := time.Now()
	defer func() {
		done := time.Now()
		dur := done.Sub(start)
		revtr.Stats.BackgroundTRSDuration += dur
	}()
	tokens := revtr.Tokens
	revtr.Tokens = nil
	tr, err := retreiveTraceroutes(tokens, b.opts.at, b.opts.cm)
	if err != nil {
		log.Error(err)
		// Failed to find a intersection
		return b.assumeSymmetric
	}
	log.Debug("Creating TRToSrc seg: ", tr.hops, " ", revtr.Src, " ", tr.addr)
	segment := rt.NewTrtoSrcRevSegment(tr.hops, revtr.Src, tr.addr)
	if !revtr.AddBackgroundTRSegment(segment, b.opts.cm) {
		panic("Failed to add background TR segment. That's not possible")
	}
	if revtr.Reaches(b.opts.cm) {
		return nil
	}
	panic("Added a TR to source but the revtr didn't reach")
}

func (b *rtBatch) assumeSymmetric(revtr *rt.ReverseTraceroute) step {
	revtr.Stats.AssumeSymmetricRoundCount++
	start := time.Now()
	defer func() {
		done := time.Now()
		dur := done.Sub(start)
		revtr.Stats.AssumeSymmetricDuration += dur
	}()
	// if last hop is assumed, add one more from that tr
	if reflect.TypeOf(revtr.CurrPath().LastSeg()) == reflect.TypeOf(&rt.DstSymRevSegment{}) {
		log.Debug("Backing off along current path for ", revtr.Src, " ", revtr.Dst)
		// need to not ignore the hops in the last segment, so can't just
		// call add_hops(revtr.hops + revtr.deadends)
		newSeg := revtr.CurrPath().LastSeg().Clone().(*rt.DstSymRevSegment)
		log.Debug("newSeg: ", newSeg)
		var allHops []string
		for i, seg := range *revtr.CurrPath().Path {
			// Skip the last one
			if i == len(*revtr.CurrPath().Path)-1 {
				continue
			}
			allHops = append(allHops, seg.Hops()...)
		}
		allHops = append(allHops, revtr.Deadends()...)
		log.Debug("all hops: ", allHops)
		err := newSeg.AddHop(allHops)
		if err != nil {
			log.Error(err)
		}
		log.Debug("New seg: ", newSeg)
		added := revtr.AddAndReplaceSegment(newSeg)
		if added {
			log.Debug("Added hop from another DstSymRevSegment")
			if revtr.Reaches(b.opts.cm) {
				return nil
			}
			return b.trToSource
		}
		panic("Should never get here")
	}
	trace, err := issueTraceroute(b.opts.cl, b.opts.cm,
		revtr.Src, revtr.LastHop(), revtr.Staleness)
	if err != nil {
		log.Debug("Issue traceroute err: ", err)
		revtr.ErrorDetails.WriteString("Error running traceroute\n")
		revtr.ErrorDetails.WriteString(err.Error() + "\n")
		revtr.FailCurrPath()
		if revtr.Failed() {
			// we failed so we're done
			revtr.FailReason = "Traceroue failed when trying to assume symmetric"
			return nil
		}
		// move on to the top of the loop
		return b.trToSource
	}
	var hToIgnore []string
	hToIgnore = append(hToIgnore, revtr.Hops()...)
	hToIgnore = append(hToIgnore, revtr.Deadends()...)
	log.Debug("Attempting to add hop from tr ", trace.hops)
	if revtr.AddSegments([]rt.Segment{
		rt.NewDstSymRevSegment(revtr.Src,
			revtr.LastHop(),
			trace.hops, 1,
			hToIgnore)},
		b.opts.cm) {
		if revtr.Reaches(b.opts.cm) {
			// done
			return nil
		}
		return b.trToSource
	}
	// everything failed
	revtr.FailCurrPath()
	if revtr.Failed() {
		revtr.FailReason = "Failed to find hops for any path."
		return nil
	}
	return b.trToSource
}

func issueTimestamps(issue map[string][][]string,
	fn func(string, string, *datamodel.Ping),
	staleness int64,
	cl client.Client) error {

	log.Debug("Issuing timestamps")
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
			log.Debug("tss string: ", tss)
			if iputil.IsPrivate(net.ParseIP(probe[0])) {
				continue
			}
			p := &datamodel.PingMeasurement{
				Src:        srcip,
				Dst:        dstip,
				TimeStamp:  tss,
				Timeout:    10,
				Count:      "1",
				CheckCache: true,
				CheckDb:    true,
				Staleness:  staleness,
			}
			pings = append(pings, p)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	st, err := cl.Ping(ctx, &datamodel.PingArg{Pings: pings})
	if err != nil {
		log.Error(err)
		return err
	}
	for {
		pr, err := st.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			return err
		}
		srcs, _ := util.Int32ToIPString(pr.Src)
		fn(srcs, "non_spoofed", pr)
	}
	return nil
}

func issueSpoofedTimestamps(issue map[string]map[string][][]string,
	fn func(string, string, *datamodel.Ping),
	staleness int64,
	cl client.Client) error {

	log.Debug("Issuing spoofed timestamps")
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
					CheckCache:  true,
					CheckDb:     true,
					Staleness:   staleness,
				}
				pings = append(pings, p)
			}
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	st, err := cl.Ping(ctx, &datamodel.PingArg{Pings: pings})
	if err != nil {
		log.Error(err)
		return err
	}
	for {
		pr, err := st.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			return err
		}
		srcs, _ := util.Int32ToIPString(pr.Src)
		vp, _ := util.Int32ToIPString(pr.SpoofedFrom)
		fn(srcs, vp, pr)
	}
	return nil
}

func (b *rtBatch) run(revtr *rt.ReverseTraceroute,
	ret chan<- *rt.ReverseTraceroute) {
	defer b.wg.Done()
	currStep := b.initialStep
	for {
		select {
		case <-b.opts.ctx.Done():
			revtr.StopReason = rt.Canceled
			ret <- revtr
			return
		default:
			currStep = currStep(revtr)
			if currStep == nil {
				log.Debug("Done running ", revtr)
				ret <- revtr
				return
			}
		}
	}
}

var (
	errPrivateIP = fmt.Errorf("The target is a private IP addr")
)

type intersectingTR struct {
	addr string
	hops []string
}

type sprrhops struct {
	hops []string
	vp   string
}

type tracerouteError struct {
	err   error
	trace *datamodel.Traceroute
	extra string
}

func (te tracerouteError) Error() string {
	var buf bytes.Buffer
	if te.err != nil {
		buf.WriteString(te.err.Error() + "\n")
	}
	if te.trace.Error != "" {
		buf.WriteString(fmt.Sprintf("Error running traceroute %v ", te.trace) + "\n")
		if te.extra != "" {
			buf.WriteString(te.extra + "\n")
		}
		return buf.String()
	}
	buf.WriteString(te.trace.ErrorString() + "\n")
	if te.extra != "" {
		buf.WriteString(te.extra + "\n")
	}
	return buf.String()
}

type traceroute struct {
	src, dst string
	hops     []string
}

func issueTraceroute(cl client.Client, cm clustermap.ClusterMap,
	src, dst string, staleness int64) (traceroute, error) {

	srci, _ := util.IPStringToInt32(src)
	dsti, _ := util.IPStringToInt32(dst)
	if iputil.IsPrivate(net.ParseIP(dst)) {
		return traceroute{}, errPrivateIP
	}
	tr := datamodel.TracerouteMeasurement{
		Src:        srci,
		Dst:        dsti,
		CheckCache: true,
		CheckDb:    true,
		Staleness:  staleness,
		Timeout:    30,
		Wait:       "2",
		Attempts:   "1",
		LoopAction: "1",
		Loops:      "3",
	}
	log.Debug("Issuing traceroute src: ", src, " dst: ", dst)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	st, err := cl.Traceroute(ctx, &datamodel.TracerouteArg{
		Traceroutes: []*datamodel.TracerouteMeasurement{&tr},
	})
	if err != nil {
		log.Error(err)
		return traceroute{}, fmt.Errorf("Failed to run traceroute: %v", err)
	}
	for {
		trace, err := st.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			return traceroute{}, fmt.Errorf("Error running traceroute: %v", err)
		}
		if trace.Error != "" {
			return traceroute{}, tracerouteError{trace: trace}
		}
		trdst, _ := util.Int32ToIPString(trace.Dst)
		var hopst []string
		cls := cm.Get(trdst)
		log.Debug("Got traceroute: ", tr)
		for i, hop := range trace.GetHops() {
			if i != len(trace.GetHops())-1 {
				j := hop.ProbeTtl + 2
				for j < trace.GetHops()[i].ProbeTtl {
					hopst = append(hopst, "*")
				}
			}
			addrst, _ := util.Int32ToIPString(hop.Addr)
			hopst = append(hopst, addrst)
		}
		if len(hopst) == 0 {
			log.Debug("Received traceroute with no hops")

			return traceroute{}, tracerouteError{trace: trace}
		}
		if cm.Get(hopst[len(hopst)-1]) != cls {
			return traceroute{}, tracerouteError{
				err:   fmt.Errorf("Traceroute didn't reach destination"),
				trace: trace,
				extra: fmt.Sprintf("<a href=\"/runrevtr?src=%s&dst=%s\">Try rerunning from the last responsive hop! </a>", src, hopst[len(hopst)-1])}
		}
		log.Debug("Got traceroute ", hopst)
		return traceroute{src: src, dst: dst, hops: hopst}, nil
	}
	return traceroute{}, fmt.Errorf("Issue traceroute failed to do anything")
}

func intersectingTraceroute(src, dst string, addrs []uint32,
	staleness int64, atl at.Atlas,
	cm clustermap.ClusterMap) (intersectingTR, []*apb.IntersectionResponse, error) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	as, err := atl.GetIntersectingPath(ctx)
	if err != nil {
		log.Error(err)
		return intersectingTR{}, nil, err
	}
	dest, _ := util.IPStringToInt32(src)
	srci, _ := util.IPStringToInt32(dst)
	for _, addr := range addrs {
		log.Debug("Attempting to find TR for hop: ", addr,
			"(", ipstr(addr).String(), ")", " to ", src)
		is := apb.IntersectionRequest{
			UseAliases: true,
			Staleness:  staleness,
			Dest:       dest,
			Address:    addr,
			Src:        srci,
		}
		err := as.Send(&is)
		if err != nil {
			log.Error(err)
			return intersectingTR{}, nil, err
		}
	}
	err = as.CloseSend()
	if err != nil {
		log.Error(err)
		return intersectingTR{}, nil, err
	}
	var tokens []*apb.IntersectionResponse
	for {
		itr, err := as.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			return intersectingTR{}, nil, err
		}
		log.Debug("Received Response: ", itr)
		switch itr.Type {
		case apb.IResponseType_PATH:
			var hs []string
			var found bool
			addr, _ := util.Int32ToIPString(itr.Path.Address)
			for _, h := range itr.Path.GetHops() {
				hss, _ := util.Int32ToIPString(h.Ip)
				log.Debug("Fixing up hop: ", hss)
				if !found && cm.Get(addr) != cm.Get(hss) {
					continue
				}
				found = true
				hs = append(hs, hss)
			}
			return intersectingTR{hops: hs, addr: addr}, nil, nil
		case apb.IResponseType_NONE_FOUND:
			log.Debug("Found no path for ", itr)
		case apb.IResponseType_TOKEN:
			tokens = append(tokens, itr)
		}
	}
	return intersectingTR{}, tokens, nil
}

func retreiveTraceroutes(reqs []*apb.IntersectionResponse, atl at.Atlas,
	cm clustermap.ClusterMap) (intersectingTR, error) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	as, err := atl.GetPathsWithToken(ctx)
	if err != nil {
		return intersectingTR{}, err
	}

	for _, req := range reqs {
		log.Debug("Sending for token ", req)
		err := as.Send(&apb.TokenRequest{
			Token: req.Token,
		})
		if err != nil {
			log.Error(err)
			return intersectingTR{}, err
		}
	}
	err = as.CloseSend()
	if err != nil {
		log.Error(err)
	}
	for {
		resp, err := as.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			return intersectingTR{}, err
		}
		log.Debug("Received token response: ", resp)
		switch resp.Type {
		case apb.IResponseType_PATH:
			var hs []string
			var found bool
			addr, _ := util.Int32ToIPString(resp.Path.Address)
			for _, h := range resp.Path.GetHops() {
				hss, _ := util.Int32ToIPString(h.Ip)
				if !found && cm.Get(addr) != cm.Get(hss) {
					continue
				}
				found = true
				hs = append(hs, hss)
			}
			return intersectingTR{hops: hs, addr: addr}, nil
		}
	}
	return intersectingTR{}, fmt.Errorf("no traceroute found")
}
func issueSpoofedRR(recv, dst string, srcs []string, staleness int64,
	cl client.Client, cm clustermap.ClusterMap) ([]sprrhops, error) {
	if iputil.IsPrivate(net.ParseIP(dst)) {
		return nil, errPrivateIP
	}
	dsti, _ := util.IPStringToInt32(dst)
	var pms []*datamodel.PingMeasurement
	for _, src := range srcs {
		log.Debug("Creating spoofed ping from ", src, " to ", dst, " recieved by ", recv)
		srci, _ := util.IPStringToInt32(src)
		pms = append(pms, &datamodel.PingMeasurement{
			Src:        srci,
			Dst:        dsti,
			SAddr:      recv,
			Timeout:    5,
			Count:      "1",
			Staleness:  staleness,
			CheckCache: true,
			Spoof:      true,
			RR:         true,
		})
	}
	var rrs []sprrhops
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	st, err := cl.Ping(ctx, &datamodel.PingArg{
		Pings: pms,
	})
	if err != nil {
		log.Error(err)
		return nil, err
	}
	for {
		p, err := st.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			return rrs, err
		}
		pr := p.GetResponses()
		if len(pr) == 0 {
			continue
		}
		sspoofer, _ := util.Int32ToIPString(p.SpoofedFrom)
		rrs = append(rrs, sprrhops{
			hops: processRR(recv, dst, pr[0].RR, true, cm),
			vp:   sspoofer,
		})
	}
	return rrs, nil
}

type rrhops []string

func issueRR(src, dst string, staleness int64,
	cl client.Client, cm clustermap.ClusterMap) (rrhops, error) {
	if iputil.IsPrivate(net.ParseIP(dst)) {
		return nil, errPrivateIP
	}
	srci, _ := util.IPStringToInt32(src)
	dsti, _ := util.IPStringToInt32(dst)
	pm := &datamodel.PingMeasurement{
		Src:        srci,
		Dst:        dsti,
		RR:         true,
		Timeout:    10,
		Count:      "1",
		CheckCache: true,
		CheckDb:    true,
		Staleness:  staleness,
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	st, err := cl.Ping(ctx, &datamodel.PingArg{
		Pings: []*datamodel.PingMeasurement{
			pm,
		},
	})
	if err != nil {
		return nil, err
	}
	for {
		p, err := st.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			return nil, err
		}
		pr := p.GetResponses()
		if len(pr) == 0 {
			return nil, fmt.Errorf("no responses")
		}
		return processRR(src, dst, pr[0].RR, true, cm), nil
	}
	return nil, fmt.Errorf("no responses")
}

func processRR(src, dst string, hops []uint32,
	removeLoops bool, cm clustermap.ClusterMap) rrhops {
	if len(hops) == 0 {
		return []string{}
	}
	dstcls := cm.Get(dst)
	var hopss []string
	for _, s := range hops {

		hs, _ := util.Int32ToIPString(s)
		hopss = append(hopss, hs)
	}
	log.Debug("Processing RR for src: ", src, " dst ", dst, " hops: ", hopss)
	if cm.Get(hopss[len(hopss)-1]) == dstcls {
		return []string{}
	}
	i := len(hops) - 1
	var found bool
	// check if we reached dst with at least one hop to spare
	for !found && i > 0 {
		i--
		if dstcls == cm.Get(hopss[i]) {
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
				clusters = append(clusters, cm.Get(hop))
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
