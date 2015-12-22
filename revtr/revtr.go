package revtr

import (
	"fmt"
	"io"
	"reflect"
	"sort"
	"sync"

	atlas "github.com/NEU-SNS/ReverseTraceroute/atlas/client"
	"github.com/NEU-SNS/ReverseTraceroute/controller/client"
	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/util"
)

var plHost2IP map[string]string
var ipToCluster map[string]string
var tsAdjsByCluster bool
var vps []*datamodel.VantagePoint
var rrVPsByCluster bool

// Adjacency are adjacencies
type Adjacency struct {
	IP1, IP2, Cnt uint32
}

// ReversePath is a reverse path
type ReversePath struct {
	Src, Dst string
	Path     []Segment
}

// NewReversePath creates a reverse path
func NewReversePath(src, dst string, path []Segment) *ReversePath {
	ret := ReversePath{
		Src: src,
		Dst: dst,
	}
	if len(path) == 0 {
		ret.Path = []Segment{NewDstRevSegment([]string{dst}, src, dst)}
	} else {
		ret.Path = path
	}
	return &ret
}

// Clone clones a ReversePath
func (rp *ReversePath) Clone() *ReversePath {
	ret := ReversePath{
		Src: rp.Src,
		Dst: rp.Dst,
	}
	for _, seg := range rp.Path {
		ret.Path = append(ret.Path, seg.Clone())
	}
	return &ret
}

// Hops gets the hops from each segment
func (rp *ReversePath) Hops() []string {
	var segs [][]string
	for _, p := range rp.Path {
		segs = append(segs, p.Hops())
	}
	var hops []string
	for _, seg := range segs {
		for _, h := range seg {
			hops = append(hops, h)
		}
	}
	return hops
}

// Length returns the length of all the segments
func (rp *ReversePath) Length() int {
	var length int
	for _, seg := range rp.Path {
		length += seg.Length(false)
	}
	return length
}

// LastHop gets the last hop of the last segment
func (rp *ReversePath) LastHop() string {
	return rp.Path[len(rp.Path)-1].LastHop()
}

// LastSeg gets the last segment
func (rp *ReversePath) LastSeg() Segment {
	return rp.Path[len(rp.Path)-1]
}

// Pop pops a segment off of the path
func (rp *ReversePath) Pop() Segment {
	length := len(rp.Path)
	last := rp.Path[length-1]
	rp.Path = rp.Path[:length-1]
	return last
}

// Reaches returns weather or not the last segment reaches
func (rp *ReversePath) Reaches() bool {
	return rp.LastSeg().Reaches()
}

// SymmetricAssumptions returns the number of symmetric assumptions
func (rp *ReversePath) SymmetricAssumptions() int {
	var total int
	for _, seg := range rp.Path {
		total += seg.SymmetricAssumptions()
	}
	return total
}

// Add adds a segment to the path
func (rp *ReversePath) Add(s Segment) {
	rp.Path = append(rp.Path, s)
}

func (rp *ReversePath) String() string {
	return fmt.Sprintf("RevPath_D%s_S%s_%v", rp.Dst, rp.Src, rp.Path)
}

const (
	// RateLimit is the ReverseTraceroute Rate Limit
	RateLimit int = 3
)

// ReverseTraceroute is a reverse traceroute
// Paths is a stack of ReversePaths
// DeadEnd is a has of ip -> bool of paths (stored as lasthop) we know don't work
// and shouldn't be tried again
//		rrhop2ratelimit is from cluster to max number of probes to send to it in a batch
//		rrhop2vpsleft is from cluster to the VPs that haven't probed it yet,
//		in prioritized order
// tshop2ratelimit is max number of adjacents to probe for at once
// tshop2adjsleft is from cluster to the set of adjacents to try in prioritized order
// [] means we've tried them all. if it is missing the key, that means we still need to
// initialize it
type ReverseTraceroute struct {
	Paths           []*ReversePath
	DeadEnd         map[string]bool
	RRHop2RateLimit map[string]int
	RRHop2VPSLeft   map[string][]string
	TSHop2RateLimit map[string]int
	TSHop2AdjsLeft  map[string][]string
	Src, Dst        string
}

// NewReverseTraceroute creates a new reverse traceroute
func NewReverseTraceroute(src, dst string) *ReverseTraceroute {
	ret := ReverseTraceroute{
		Src:   src,
		Dst:   dst,
		Paths: []*ReversePath{NewReversePath(src, dst, nil)},
	}
	return &ret
}

// SymmetricAssumptions returns the number of symmetric
// assumptions of the last path
func (rt *ReverseTraceroute) SymmetricAssumptions() int {
	return rt.Paths[len(rt.Paths)-1].SymmetricAssumptions()
}

// Deadends returns the ips of all the deadends
func (rt *ReverseTraceroute) Deadends() []string {
	var keys []string
	for k := range rt.DeadEnd {
		keys = append(keys, k)
	}
	return keys
}

func (rt *ReverseTraceroute) rrVPSInitializedForHop(hop string) bool {
	_, ok := rt.RRHop2VPSLeft[hop]
	return ok
}

func (rt *ReverseTraceroute) setRRVPSForHop(hop string, vps []string) {
	rt.RRHop2VPSLeft[hop] = vps
}

// CurrPath gets the last Path in the Paths "stack"
func (rt *ReverseTraceroute) CurrPath() *ReversePath {
	return rt.Paths[len(rt.Paths)-1]
}

// Hops gets the hops from the last path
func (rt *ReverseTraceroute) Hops() []string {
	if len(rt.Paths) == 0 {
		return []string{}
	}
	return rt.Paths[len(rt.Paths)-1].Hops()
}

// LastHop gets the last hop from the last path
func (rt *ReverseTraceroute) LastHop() string {
	return rt.Paths[len(rt.Paths)-1].LastHop()
}

// Reaches checks if the last path reaches
func (rt *ReverseTraceroute) Reaches() bool {
	// Assume that any path reaches if and only if the last one reaches
	return rt.Paths[len(rt.Paths)-1].Reaches()
}

// Failed returns whether we have any options lefts to explore
func (rt *ReverseTraceroute) Failed(backoffEndhost bool) bool {
	return len(rt.Paths) == 0 || (backoffEndhost && len(rt.Paths) == 1 &&
		len(rt.Paths[0].Path) == 1 && reflect.TypeOf(rt.Paths[0].Path[0]) == reflect.TypeOf(&DstRevSegment{}))
}

// FailCurrPath fails the current path
func (rt *ReverseTraceroute) FailCurrPath() {
	rt.DeadEnd[rt.LastHop()] = true
	// keep popping until we find something that is either on a path
	// we are assuming symmetric (we know it started at src so goes to whole way)
	// or is not known to be a deadend
	for !rt.Failed(false) && rt.DeadEnd[rt.LastHop()] && reflect.TypeOf(rt.CurrPath().LastSeg()) !=
		reflect.TypeOf(&DstSymRevSegment{}) {
		// Pop
		rt.Paths = rt.Paths[:len(rt.Paths)-1]
	}
}

// AddAndReplaceSegment adds a new path, equal to the current one but with the last
// segment replaced by the new one
// returns a bool of whether it was added
// might not be added if it is a deadend
func (rt *ReverseTraceroute) AddAndReplaceSegment(s Segment) bool {
	if rt.DeadEnd[s.LastHop()] {
		return false
	}
	basePath := rt.CurrPath()
	basePath.Pop()
	basePath.Add(s)
	rt.Paths = append(rt.Paths, basePath)
	return true
}

/*
TODO
I'm not entirely sure that this sort will match  the ruby one
It will need to be tested and verified
*/
type magicSort []Segment

func (ms magicSort) Len() int           { return len(ms) }
func (ms magicSort) Swap(i, j int)      { ms[i], ms[j] = ms[j], ms[i] }
func (ms magicSort) Less(i, j int) bool { return ms[i].Order(ms[j]) < 0 }

// AddSegments returns a bool of whether any were added
// might not be added if they are deadends
// or if all hops would cause loops
func (rt *ReverseTraceroute) AddSegments(segs []Segment) bool {
	var added *bool
	added = new(bool)
	// sort based on the magic compare
	// or how long the path is?
	sort.Sort(magicSort(segs))
	basePath := rt.CurrPath()
	for _, s := range segs {
		if !rt.DeadEnd[s.LastHop()] {
			// add loop removal here
			s.RemoveHops(basePath.Hops())
			if s.Length(false) == 0 {
				log.Debug("Skipping loop-causing segment ", s)
				continue
			}
			*added = true
			basePath.Add(s)
			rt.Paths = append(rt.Paths, basePath)
		}
	}
	return *added
}

type adjSettings struct {
	timeout      int
	maxnum       int
	maxalert     string
	retryCommand bool
}

func getAdjacenciesForIPToSrc(cls string, src string, settings adjSettings) ([]string, error) {
	return nil, nil
}

// InitializeTSAdjacents ...
func (rt *ReverseTraceroute) InitializeTSAdjacents(cls string) error {
	adjs, err := getAdjacenciesForIPToSrc(cls, rt.Src, adjSettings{})
	if err != nil {
		return err
	}
	var cleaned []string
	for _, ip := range adjs {
		if cls != ipToCluster[ip] {
			cleaned = append(cleaned, ip)
		}
	}
	rt.TSHop2AdjsLeft[cls] = cleaned
	return nil
}

// GetTSAdjacents get the set of adjacents to try for a hop
// for revtr:s,d,r, the set of R' left to consider, or if there are none
// will return the number that we want to probe at a time
func (rt *ReverseTraceroute) GetTSAdjacents(hop string) []string {
	var cls string
	if tsAdjsByCluster {
		cls = ipToCluster[hop]
	} else {
		cls = hop
	}
	if _, ok := rt.TSHop2AdjsLeft[cls]; !ok {
		rt.InitializeTSAdjacents(cls)
	}
	log.Debug(rt.Src, rt.Dst, rt.LastHop(), len(rt.TSHop2AdjsLeft[cls]), "TS adjacents left to try")

	// CASES:
	// 1. no adjacents left return nil
	if len(rt.TSHop2AdjsLeft[cls]) == 0 {
		return nil
	}
	// CAN EVENTUALLY MOVE TO REUSSING PROBES TO THIS DST FROM ANOTHER
	// REVTR, BUT NOT FOR NOW
	// 2. For now, we just take the next batch and send them
	var min int
	var rate int
	if val, ok := rt.TSHop2RateLimit[cls]; ok {
		rate = val
	} else {
		rate = 1
	}
	if rate > len(rt.TSHop2AdjsLeft[cls]) {
		min = len(rt.TSHop2AdjsLeft[cls])
	} else {
		min = rate
	}
	adjacents := rt.TSHop2AdjsLeft[cls][:min]
	rt.TSHop2AdjsLeft[cls] = rt.TSHop2AdjsLeft[cls][min:]
	return adjacents
}

func chooseOneSpooferPerSite() map[string]*datamodel.VantagePoint {
	ret := make(map[string]*datamodel.VantagePoint)
	for _, vp := range vps {
		if vp.CanSpoof {
			ret[vp.Site] = vp
		}
	}
	return ret
}

// for now we're just ignoring the src dst and choosing randomly
func getTimestampSpoofers(src, dst string) []string {
	siteToSpoofer := chooseOneSpooferPerSite()
	var spoofers []string
	for _, val := range siteToSpoofer {
		if val.Timestamp {
			ips, _ := util.Int32ToIPString(val.Ip)
			spoofers = append(spoofers, ips)
		}
	}
	return spoofers
}

// InitializeRRVPs initializes the rr vps for a cls
func (rt *ReverseTraceroute) InitializeRRVPs(cls string) error {
	log.Debug("Initializing RR VPs individually for spoofers for ", cls)
	rt.RRHop2RateLimit[cls] = RateLimit
	siteToSpoofer := chooseOneSpooferPerSite()
	var sitesForTarget []*datamodel.VantagePoint
	sitesForTarget = nil
	spoofersForTarget := []string{"non_spoofed"}

	if sitesForTarget == nil {
		// This should be the same as sorting the values randomly
		// since iterating over a map is randomized by the runtime
		for _, val := range siteToSpoofer {
			ipsrc, _ := util.Int32ToIPString(val.Ip)
			if ipsrc == rt.Src {
				continue
			}
			spoofersForTarget = append(spoofersForTarget, ipsrc)
		}
	} else {
		// TODO
		// This is the case for using smarter results for vp selection
		// currently we don't have this so nothing is gunna happen
	}
	rt.RRHop2VPSLeft[cls] = spoofersForTarget
	return nil
}

func cloneStringSlice(ss []string) []string {
	var ret []string
	for _, s := range ss {
		ret = append(ret, s)
	}
	return ret
}

var batchInitRRVPs = true
var rrsSrcToDstToVPToRevHops = make(map[string]map[string]map[string][]string)
var rrsMu sync.Mutex
var maxUnresponsive = 10

func stringSliceMinus(l, r []string) []string {
	old := make(map[string]bool)
	var ret []string
	for _, s := range l {
		old[s] = true
	}
	for _, s := range r {
		if _, ok := old[s]; ok {
			delete(old, s)
		}
	}
	for key := range old {
		ret = append(ret, key)
	}
	return ret
}

func stringSliceIndex(segs []string, seg string) int {
	for i, s := range segs {
		if s == seg {
			return i
		}
	}
	return -1
}

// AddBackgroundTRSegment need a different function because a TR segment might intersect
// at an IP back up the TR chain, want to delete anything that has been added along the way
// THIS IS ALMOST DEFINITLY WRONG AND WILL NEED DEBUGGING
func (rt *ReverseTraceroute) AddBackgroundTRSegment(trSeg Segment) bool {
	var found *ReversePath
	var index int
	// iterate through the paths, trying to find one that contains
	// intersection point
	for _, chunk := range rt.Paths {
		c := chunk
		found = chunk
		var ch *ReversePath
		var foundOne bool
		for i := range chunk.Hops() {
			if trSeg.Hops()[0] == chunk.Hops()[i] {
				foundOne = true
				index = 1
				ch = c.Clone()
			}
		}
		if foundOne {
			// Iterate through all the segments until you find the
			// hop where they intersect. After reaching the hop
			// where they intersect, delete any subsequent hops
			// within the same segment.
			// Then delete any segments after.
			var k int // Which IP Hop we're at
			var j int // Which segment we're at
			for len(ch.Hops())-1 > index {
				// get current segment
				seg := ch.Path[j]
				// if we're past the intersection point then delete the whole segment
				if k > index {
					ch.Path, ch.Path[len(ch.Path)-1] = append(ch.Path[:j], ch.Path[j+1:]...), nil
				} else if k+len(seg.Hops())-1 > index {
					l := stringSliceIndex(seg.Hops(), trSeg.Hops()[0]) + 1
					for k+len(seg.Hops())-1 > index {
						ch.Path, ch.Path[len(ch.Path)-1] = append(ch.Path[:l], ch.Path[l+1:]...), nil
					}
				} else {
					j++
					k += len(seg.Hops())
				}
			}
			break
		}
	}
	if found != nil {
		return false
	}
	rt.Paths = append(rt.Paths, found)
	// Now that the traceroute is cleaned up, add the new segment
	// this sequence slightly breaks how add_segment normally works here
	// we append a cloned path (with any hops past the intersection trimmed).
	// then, we call add_segment. Add_segment clones the last path,
	// then adds the segemnt to it. so we end up with an extra copy of found,
	// that might have soem hops trimmed off it. not the end of the world,
	// but something to be aware of
	success := rt.AddSegments([]Segment{trSeg})
	if !success {
		for i, s := range rt.Paths {
			if found == s {
				rt.Paths, rt.Paths[len(rt.Paths)-1] = append(rt.Paths[:i], rt.Paths[i+1:]...), nil
			}
		}
	}
	return success
}

// GetRRVPs returns the next set to probe from, plus the next destination to probe
// nil means none left, already probed from anywhere
// the first time, initialize the set of VPs
// if any exist that have already probed the dst but haven't been used in
// this reverse traceroute, return them, otherwise return [:non_spoofed] first
// then set of spoofing VPs on subsequent calls
func (rt *ReverseTraceroute) GetRRVPs(dst string) ([]string, string) {
	// we either use destination or cluster, depending on how flag is set
	if !batchInitRRVPs {
		hops := rt.CurrPath().LastSeg().Hops()
		for _, hop := range hops {
			cls := &hop
			if rrVPsByCluster {
				*cls = ipToCluster[dst]
			}
			if _, ok := rt.RRHop2VPSLeft[*cls]; !ok {
				rt.InitializeRRVPs(*cls)
			}
		}
	}
	// CASES:
	segHops := cloneStringSlice(rt.CurrPath().LastSeg().Hops())
	var target, cls *string
	target = new(string)
	cls = new(string)
	var foundVPs bool
	for !foundVPs && len(segHops) > 0 {
		*target, segHops = segHops[len(segHops)-1], segHops[:len(segHops)-1]
		*cls = *target
		if rrVPsByCluster {
			*cls = ipToCluster[*target]
		}
		var vals [][]string
		for _, val := range rrsSrcToDstToVPToRevHops[rt.Src][*cls] {
			if len(val) > 0 {
				vals = append(vals, val)
			}
		}
		// 0. destination seems to be unresponsive
		if len(rrsSrcToDstToVPToRevHops[rt.Src][*cls]) >= maxUnresponsive &&
			// this may not match exactly but I think it does
			len(vals) == 0 {
			continue
		}
		// 1. no VPs left, return nil
		if len(rt.RRHop2VPSLeft[*cls]) == 0 {
			continue
		}
		foundVPs = true
	}
	if !foundVPs {
		return nil, ""
	}
	log.Debug(rt.Src, rt.Dst, *target, len(rt.RRHop2VPSLeft[*cls]), "RR VPs left to try")
	// 2. probes to this dst that were already issues for other reverse
	// traceroutes, but not in this reverse traceroute
	var keys []string
	tmp := rrsSrcToDstToVPToRevHops[rt.Src][*cls]
	for k := range tmp {
		keys = append(keys, k)
	}
	usedVps := stringSet(keys).union(stringSet(rt.RRHop2VPSLeft[*cls]))
	rt.RRHop2VPSLeft[*cls] = stringSliceMinus(rt.RRHop2VPSLeft[*cls], usedVps)
	var finalUsedVPs []string
	for _, uvp := range usedVps {
		idk, ok := rrsSrcToDstToVPToRevHops[rt.Src][*cls][uvp]
		if ok && len(idk) > 0 {
			continue
		}
		finalUsedVPs = append(finalUsedVPs, uvp)
	}
	if len(finalUsedVPs) > 0 {
		return finalUsedVPs, *target
	}

	// 3. send non-spoofed version if it is in the next batch
	min := rt.RRHop2RateLimit[*cls]
	if len(rt.RRHop2VPSLeft[*cls]) < min {
		min = len(rt.RRHop2VPSLeft[*cls])
	}
	if rt.RRHop2VPSLeft[*cls][0:min][0] == "non_spoofed" {
		rt.RRHop2VPSLeft[*cls] = rt.RRHop2VPSLeft[*cls][1:]
		return []string{"non_spoofed"}, *target
	}

	// 4. use unused spoofing VPs
	// if the current last hop was discovered with spoofed, and it
	// hasn't been used yet, use it
	notEmpty := len(rt.Paths) > 0
	var isRRRev, containsKey *bool
	isRRRev = new(bool)
	containsKey = new(bool)
	spoofer := new(string)
	if rrev, ok := rt.CurrPath().LastSeg().(*SpoofRRRevSegment); ok {
		*isRRRev = true
		*spoofer = rrev.SpoofSource
		if _, ok := rrsSrcToDstToVPToRevHops[rt.Src][*cls][rrev.SpoofSource]; ok {
			*containsKey = true
		}
	}
	if notEmpty && *isRRRev && !*containsKey {
		log.Debug("Found recent spoofer to use ", *spoofer)
		var newleft []string
		for _, s := range rt.RRHop2VPSLeft[*cls] {
			if s == *spoofer {
				continue
			}
			newleft = append(newleft, s)
		}
		rt.RRHop2VPSLeft[*cls] = newleft
		min := rt.RRHop2RateLimit[*cls] - 1
		if len(rt.RRHop2VPSLeft[*cls]) < min {
			min = len(rt.RRHop2VPSLeft[*cls])
		}
		vps := append([]string{*spoofer}, rt.RRHop2VPSLeft[*cls][:min]...)
		rt.RRHop2VPSLeft[*cls] = rt.RRHop2VPSLeft[*cls][min:]
		return vps, *target
	}
	min = rt.RRHop2RateLimit[*cls]
	if len(rt.RRHop2VPSLeft[*cls]) < min {
		min = len(rt.RRHop2VPSLeft[*cls])
	}
	vps := rt.RRHop2VPSLeft[*cls][:min]
	rt.RRHop2VPSLeft[*cls] = rt.RRHop2VPSLeft[*cls][min:]
	return vps, *target
}

var dstMustBeReachable = true

// ReverseTracerouteReq isa revtr req
type ReverseTracerouteReq struct {
	Src, Dst string
}

func reverseTraceroute(revtr ReverseTracerouteReq, backoffEndhost bool, cl client.Client, at atlas.Atlas) (*ReverseTraceroute, string, map[string]int, error) {
	probeCount := make(map[string]int)
	rt := NewReverseTraceroute(revtr.Src, revtr.Dst)
	if backoffEndhost || dstMustBeReachable {
		err := reverseHopsAssumeSymmetric(rt, cl, probeCount)
		if err != nil {
			return rt, "NO_HOPS", probeCount, err
		}
		if rt.Reaches() {
			return rt, "TRIVIAL", probeCount, nil
		}

	}
	for {
		err := reverseHopsTRToSrc(rt, at)
		if rt.Reaches() {
			return rt, "REACHES", probeCount, nil
		}
		if err == nil {
			continue
		}
		err = reverseHopsRR(rt, cl)
		if rt.Reaches() {
			return rt, "REACHES", probeCount, nil
		}
		if err == nil {
			continue
		}
		err = reverseHopsAssumeSymmetric(rt, cl, probeCount)
		if rt.Reaches() {
			return rt, "REACHES", probeCount, nil
		}
		if err == nil {
			continue
		}
	}
}

func stringInSlice(ss []string, s string) bool {
	for _, item := range ss {
		if s == item {
			return true
		}
	}
	return false
}

func reverseHopsRR(revtr *ReverseTraceroute, cl client.Client) error {
	vps, target := revtr.GetRRVPs(revtr.LastHop())
	receiverToSpooferToTarget := make(map[string]map[string][]string)
	var pings []*datamodel.PingMeasurement
	if len(vps) == 0 {
		return fmt.Errorf("No VPs found")
	}
	var cls *string
	cls = new(string)
	if rrVPsByCluster {
		*cls = ipToCluster[target]
	} else {
		*cls = target
	}
	var keys []string
	for k := range rrsSrcToDstToVPToRevHops[revtr.Src][*cls] {
		keys = append(keys, k)
	}
	vps = stringSliceMinus(vps, keys)
	if stringInSlice(vps, "non_spoofed") {
		var nvps []string
		for _, vp := range vps {
			if vp != "non_spoofed" {
				nvps = append(nvps)
			}
		}
		vps = nvps
		srcs, _ := util.IPStringToInt32(revtr.Src)
		dsts, _ := util.IPStringToInt32(target)
		pings = append(pings, &datamodel.PingMeasurement{
			Src: srcs,
			Dst: dsts,
		})
	}
	for _, vp := range vps {
		receiverToSpooferToTarget[revtr.Src][vp] = append(receiverToSpooferToTarget[revtr.Src][vp], target)
	}
	if len(pings) == 1 {
		issueRecordRoutes(pings[0], cl)
	}
	issueSpoofedRecordRoutes(receiverToSpooferToTarget, cl, true)
	var segs []Segment
	if rrVPsByCluster {
		target = ipToCluster[target]
	}
	for _, vp := range vps {
		hops := rrsSrcToDstToVPToRevHops[revtr.Src][target][vp]
		if len(hops) > 0 {
			// for every non-zero hop, build a revsegment
			for i, hop := range hops {
				if hop == "0.0.0.0" {
					continue
				}
				if vp == "non_spoofed" {
					segs = append(segs, NewRRRevSegment(hops[:i], revtr.Src, target))
				} else {
					segs = append(segs, NewSpoofRRRevSegment(hops[:i], revtr.Src, target, vp))
				}
			}
		}
	}
	if !revtr.AddSegments(segs) {
		return fmt.Errorf("No hops found")
	}
	return nil
}

var rrsstdMu sync.Mutex

func issueSpoofedRecordRoutes(recvToSpooferToTarget map[string]map[string][]string, cl client.Client, deleteUnresponsive bool) error {
	var pings []*datamodel.PingMeasurement
	for rec, spoofToTarg := range recvToSpooferToTarget {
		for spoofer, targets := range spoofToTarg {
			for _, target := range targets {
				ssrc, _ := util.IPStringToInt32(rec)
				sspoofer, _ := util.IPStringToInt32(spoofer)
				sdst, _ := util.IPStringToInt32(target)
				pings = append(pings, &datamodel.PingMeasurement{
					Spoof:       true,
					RR:          true,
					SpooferAddr: ssrc,
					Src:         sspoofer,
					Dst:         sdst,
				})
			}
		}
	}
	st, err := cl.Ping(&datamodel.PingArg{
		Pings: pings,
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
			return err
		}
		pr := p.GetResponses()
		if len(pr) > 0 {
			rrsstdMu.Lock()
			ssrc, _ := util.Int32ToIPString(p.Src)
			sdst, _ := util.Int32ToIPString(p.Dst)
			sspoofer, _ := util.Int32ToIPString(p.SpoofedFrom)
			rrs := pr[0].RR
			cls := new(string)
			if rrVPsByCluster {
				*cls = ipToCluster[sdst]
			} else {
				*cls = sdst
			}
			rrsSrcToDstToVPToRevHops[ssrc][sdst][sspoofer] = processRR(ssrc, sdst, rrs, true)
			rrsstdMu.Unlock()
		}
	}
	return nil
}

func issueRecordRoutes(ping *datamodel.PingMeasurement, cl client.Client) error {
	var cls, sdst, ssrc *string
	cls = new(string)
	sdst = new(string)
	ssrc = new(string)
	*ssrc, _ = util.Int32ToIPString(ping.Src)
	if rrVPsByCluster {
		*sdst, _ = util.Int32ToIPString(ping.Dst)
		*cls = ipToCluster[*sdst]
	} else {
		*sdst, _ = util.Int32ToIPString(ping.Dst)
		*cls = *sdst
	}
	st, err := cl.Ping(&datamodel.PingArg{
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
			return err
		}
		pr := p.GetResponses()
		if len(pr) > 0 {
			rrsstdMu.Lock()
			rrsSrcToDstToVPToRevHops[*sdst][*cls]["non_spoofed"] = processRR(*ssrc, *sdst, pr[0].RR, true)
			rrsstdMu.Unlock()
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
	dstcls := ipToCluster[dst]
	var hopss []string
	for _, s := range hops {
		hs, _ := util.Int32ToIPString(s)
		hopss = append(hopss, hs)
	}
	if ipToCluster[hopss[len(hopss)-1]] == dstcls {
		return []string{}
	}
	i := len(hops) - 1
	var found bool
	// check if we reached dst with at least one hop to spare
	for !found && i > 0 {
		i--
		if dstcls == ipToCluster[hopss[i]] {
			found = true
		}
	}
	if found {
		hopss = hopss[i:]
		// remove cluster level loops
		if removeLoops {
			var currIndex int
			var clusters []string
			for _, hop := range hopss {
				clusters = append(clusters, ipToCluster[hop])
			}
			for currIndex < len(hopss)-1 {
				for x := currIndex; currIndex < stringSliceRIndex(clusters, clusters[currIndex]); x++ {
					hops[x] = hops[currIndex]
					clusters[x] = clusters[currIndex]
				}
				currIndex++
			}
		}
		return hopss
	}
	return []string{}
}

func reverseHopsTRToSrc(revtr *ReverseTraceroute, cl atlas.Atlas) error {
	as, err := cl.GetIntersectingPath()
	if err != nil {
		return nil
	}
	for _, hop := range revtr.CurrPath().LastSeg().Hops() {
		dest, _ := util.IPStringToInt32(revtr.Dst)
		hops, _ := util.IPStringToInt32(hop)
		is := datamodel.IntersectionRequest{
			UseAliases: true,
			Staleness:  15,
			Dest:       dest,
			Address:    hops,
		}
		err := as.Send(&is)
		if err != nil {
			return err
		}
	}
	for {
		tr, err := as.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if tr.Type == datamodel.IResponseType_PATH {
			var hs []string
			for _, h := range tr.Path.GetHops() {
				hss, _ := util.Int32ToIPString(h.Ip)
				hs = append(hs, hss)
			}
			addrs, _ := util.Int32ToIPString(tr.Path.Address)
			segment := NewTrtoSrcRevSegment(hs, revtr.Src, addrs)
			revtr.AddBackgroundTRSegment(segment)
		}
	}
	return nil
}

var trsSrcToDstToPath = make(map[string]map[string][]string)
var trsMu sync.Mutex

func issueTraceroute(revtr *ReverseTraceroute, cl client.Client) error {
	src, _ := util.IPStringToInt32(revtr.Src)
	dst, _ := util.IPStringToInt32(revtr.LastHop())
	tr := datamodel.TracerouteMeasurement{
		Src: src,
		Dst: dst,
	}
	st, err := cl.Traceroute(&datamodel.TracerouteArg{
		Traceroutes: []*datamodel.TracerouteMeasurement{&tr},
	})
	if err != nil {
		log.Error(err)
		return err
	}
	for {
		tr, err := st.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			return err
		}
		dstSt, _ := util.Int32ToIPString(tr.Dst)
		cls := ipToCluster[dstSt]
		var hopst []string
		hops := tr.GetHops()
		for _, hop := range hops {
			addrst, _ := util.Int32ToIPString(hop.Addr)
			hopst = append(hopst, addrst)
		}
		trsMu.Lock()
		trsSrcToDstToPath[revtr.Src][cls] = hopst
		trsMu.Unlock()
	}
	return nil
}

func reverseHopsAssumeSymmetric(revtr *ReverseTraceroute, cl client.Client, probeCounts map[string]int) error {
	// if last hop is assumed, add one more from that tr
	if reflect.TypeOf(revtr.CurrPath().LastSeg()) == reflect.TypeOf(&DstSymRevSegment{}) {
		log.Debug("Backing off along current path for ", revtr.Src, revtr.Dst)
		// need to not ignore the hops in the last segment, so can't just
		// call add_hops(revtr.hops + revtr.deadends)
		newSeg := revtr.CurrPath().LastSeg().(*DstSymRevSegment)
		var allHops []string
		for _, seg := range revtr.CurrPath().Path {
			allHops = append(allHops, seg.Hops()...)
		}
		allHops = append(allHops, revtr.Deadends()...)
		newSeg.AddHop(allHops)
		added := revtr.AddAndReplaceSegment(newSeg)
		if added {
			return nil
		}
	}
	trsMu.Lock()
	_, ok := trsSrcToDstToPath[revtr.Src][ipToCluster[revtr.LastHop()]]
	trsMu.Unlock()
	if !ok {
		err := issueTraceroute(revtr, cl)
		if err != nil {
			return err
		}
		trsMu.Lock()
		tr, ok := trsSrcToDstToPath[revtr.Src][ipToCluster[revtr.LastHop()]]
		trsMu.Unlock()
		if ok && ipToCluster[tr[len(tr)-1]] == ipToCluster[revtr.LastHop()] {
			var hToIgnore []string
			hToIgnore = append(hToIgnore, revtr.Hops()...)
			hToIgnore = append(hToIgnore, revtr.Deadends()...)
			if !revtr.AddSegments([]Segment{NewDstSymRevSegment(revtr.Src, revtr.LastHop(), tr, 1, hToIgnore)}) {
				return fmt.Errorf("No Hops Added")
			}
		}
		return fmt.Errorf("No Hops Added")
	}
	return nil
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

func reverseHopsTS(revtr *ReverseTraceroute, cl client.Client, probeCount map[string]int) error {

	var tsToIssueSrcToProbe = make(map[string][][]string)
	var receiverToSpooferToProbe = make(map[string]map[string][][]string)
	var dstsDoNotStamp [][]string
	if tsSrcToHopToResponsive[revtr.Src][revtr.LastHop()] != 0 {
		return fmt.Errorf("No VPS")
	}
	adjacents := revtr.GetTSAdjacents(ipToCluster[revtr.LastHop()])
	if len(adjacents) == 0 {
		return fmt.Errorf("No Adjacents found")
	}
	if tsDstToStampsZero[revtr.LastHop()] {
		for _, adj := range adjacents {
			dstsDoNotStamp = append(dstsDoNotStamp, []string{revtr.Src, revtr.LastHop(), adj})
		}
	} else if tsSrcToHopToSendSpoofed[revtr.Src][revtr.LastHop()] {
		for _, adj := range adjacents {
			tsToIssueSrcToProbe[revtr.Src] = append(tsToIssueSrcToProbe[revtr.Src], []string{revtr.LastHop(), revtr.LastHop(), adj, adj, dummyIP})
		}
	} else {
		spfs := getTimestampSpoofers(revtr.Src, revtr.LastHop())
		for _, adj := range adjacents {
			for _, spf := range spfs {
				receiverToSpooferToProbe[revtr.Src][spf] = append(receiverToSpooferToProbe[revtr.Src][spf], []string{revtr.LastHop(), revtr.LastHop(), adj, adj, dummyIP})
			}
		}
		// if we haven't already decided whether it is responsive,
		// we'll set it to false, then change to true if we get one
		if _, ok := tsSrcToHopToResponsive[revtr.Src][revtr.LastHop()]; !ok {
			tsSrcToHopToResponsive[revtr.Src][revtr.LastHop()] = 1
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
		dsts, _ := util.Int32ToIPString(p.Dst)
		segClass := "SpoofTSAdjRevSegment"
		if vp == "non_spoofed" {
			tsSrcToHopToSendSpoofed[src][dsts] = false
			segClass = "TSAdjRevSegment"
		}
		tsSrcToHopToResponsive[src][dsts] = 1
		rps := p.GetResponses()
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
				tsDstToStampsZero[dsts] = true
				destDoesNotStamp = append(destDoesNotStamp, tripletTs{src: src, dst: dsts, tsip: ts2ips})
			} else {
				log.Debug("TS probe is ", vp, p, "no reverse hop found")
			}
		}
	}
	if len(tsToIssueSrcToProbe) > 0 {
		// there should be a uniq thing here but I need to figure out how to do it
		for src, probes := range tsToIssueSrcToProbe {
			for _, probe := range probes {
				if _, ok := tsSrcToHopToSendSpoofed[src][probe[0]]; ok {
					continue
				}
				tsSrcToHopToSendSpoofed[src][probe[0]] = true
			}
		}
		issueTimestamps(tsToIssueSrcToProbe, cl, probeCount, processTSCheckForRevHop)
		for src, probes := range tsToIssueSrcToProbe {
			for _, probe := range probes {
				// if we got a reply, would have set sendspoofed to false
				// so it is still true, we need to try to find a spoofer
				if tsSrcToHopToSendSpoofed[src][probe[0]] {
					mySpoofers := getTimestampSpoofers(src, probe[0])
					for _, sp := range mySpoofers {
						receiverToSpooferToProbe[src][sp] = append(receiverToSpooferToProbe[src][sp], probe)
					}
					// if we haven't already decided whether it is responsive
					// we'll set it to false, then change to true if we get one
					if _, ok := tsSrcToHopToResponsive[src][probe[0]]; !ok {
						tsSrcToHopToResponsive[src][probe[0]] = 1
					}
				}
			}
		}
	}
	if len(receiverToSpooferToProbe) > 0 {
		issueSpoofedTimestamps(receiverToSpooferToProbe, cl, probeCount, processTSCheckForRevHop)
	}
	if len(linuxBugToCheckSrcDstVpToRevHops) > 0 {
		var linuxChecksSrcToProbe = make(map[string][][]string)
		var linuxChecksSpoofedReceiverToSpooferToProbe = make(map[string]map[string][][]string)
		for sdvp := range linuxBugToCheckSrcDstVpToRevHops {
			p := []string{sdvp.dst, sdvp.dst, dummyIP, dummyIP}
			if sdvp.vp == "non_spoofed" {
				linuxChecksSrcToProbe[sdvp.src] = append(linuxChecksSrcToProbe[sdvp.src], p)
			} else {
				linuxChecksSpoofedReceiverToSpooferToProbe[sdvp.src][sdvp.vp] = append(linuxChecksSpoofedReceiverToSpooferToProbe[sdvp.src][sdvp.vp], p)
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
				tsSrcToHopToSendSpoofed[src][dsts] = false
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
		issueTimestamps(linuxChecksSrcToProbe, cl, probeCount, processTSCheckForLinuxBug)
		issueSpoofedTimestamps(linuxChecksSpoofedReceiverToSpooferToProbe, cl, probeCount, processTSCheckForLinuxBug)
	}
	receiverToSpooferToProbe = make(map[string]map[string][][]string)
	for _, probe := range destDoesNotStamp {
		spoofers := getTimestampSpoofers(probe.src, probe.dst)
		for _, s := range spoofers {
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
		issueSpoofedTimestamps(receiverToSpooferToProbe, cl, probeCount, processTSDestDoesNotStamp)
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
		issueTimestamps(destDoesNotStampToVerifySpooferToProbe, cl, probeCount, processTSDestDoesNotStampToVerify)
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
	if segments, ok := revHopsSrcDstToRevSeg[pair{src: revtr.Src, dst: revtr.LastHop()}]; ok {
		if revtr.AddSegments(segments) {
			return nil
		}
	}
	return nil
}

// whether this destination is repsonsive but with ts=0
var tsDstToStampsZero = make(map[string]bool)

// whether this particular src should use spoofed ts to that hop
var tsSrcToHopToSendSpoofed = make(map[string]map[string]bool)

// whether this hop is thought to be responsive at all to this src
// Since I can't intialize to true, I'm going to use an int and say 0 is true
// anythign else will be false
var tsSrcToHopToResponsive = make(map[string]map[string]int)

// nil means we issued the probe, did not get a response
var tsSrcToProbeToVPToResult = make(map[string]map[string]map[string][]string)

func issueTimestamps(issue map[string][][]string, cl client.Client, probeCount map[string]int, fn func(string, string, *datamodel.Ping)) error {

	return nil
}

func issueSpoofedTimestamps(issue map[string]map[string][][]string, cl client.Client, probeCount map[string]int, fn func(string, string, *datamodel.Ping)) error {

	return nil
}
