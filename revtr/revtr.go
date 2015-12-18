package revtr

import (
	"fmt"
	"reflect"
	"sort"
	"sync"

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
func getTimestampSpoofers(src, dst string) []*datamodel.VantagePoint {
	siteToSpoofer := chooseOneSpooferPerSite()
	var spoofers []*datamodel.VantagePoint
	for _, val := range siteToSpoofer {
		if val.Timestamp {
			spoofers = append(spoofers, val)
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

func reverseTraceroute(revtr ReverseTracerouteReq, backoffEndhost bool, cl client.Client) (*ReverseTraceroute, string, map[string]int, error) {
	probeCount := make(map[string]int)
	//rt := NewReverseTraceroute(revtr.Src, revtr.Dst)
	return nil, "", probeCount, nil
}

var trsSrcToDstToPath = make(map[string]map[string]string)
var trsMu sync.Mutex

func issueTraceroute(revtr *ReverseTraceroute, cl client.Client, deleteUnresponsive bool) {
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
	}
}

func reverseHopsAssumeSymmetric(revtr *ReverseTraceroute, cl client.Client, probeCounts map[string]int) {
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
			return
		}
	}
	trsMu.Lock()
	_, ok := trsSrcToDstToPath[revtr.Src][ipToCluster[revtr.LastHop()]]
	if !ok {
	}
}
