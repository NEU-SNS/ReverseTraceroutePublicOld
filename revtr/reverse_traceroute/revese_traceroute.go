package reversetraceroute

import (
	"bytes"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"sync"
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/util/string"

	apb "github.com/NEU-SNS/ReverseTraceroute/atlas/pb"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/clustermap"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/pb"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/types"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	vpservice "github.com/NEU-SNS/ReverseTraceroute/vpservice/client"
)

var (
	dstMustBeReachable = false
	batchInitRRVPs     = true
	maxUnresponsive    = 10
	rrVPsByCluster     bool
	tsAdjsByCluster    bool
)

// StopReason is the reason a reverse traceroute stopped
type StopReason string

const (
	// Failed is the StopReason when no technique could add a hop
	// or an unrecoverable error occured
	Failed StopReason = "FAILED"
	// Trivial is the StopReason when a revtr was trivial to accomplish
	Trivial StopReason = "TRIVIAL"
	// Canceled is the StopReason when a revtr is canceled
	Canceled StopReason = "CANCELED"
	// Reaches is the StopReason when a revtr has reached its destination
	Reaches StopReason = "REACHES"
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
	ID                      uint32
	logStr                  string
	Paths                   *[]*ReversePath
	DeadEnd                 map[string]bool
	RRHop2RateLimit         map[string]int
	RRHop2VPSLeft           map[string][]string
	TSHop2RateLimit         map[string]int
	TSHop2AdjsLeft          map[string][]string
	Src, Dst                string
	StopReason              StopReason
	StartTime, EndTime      time.Time
	Staleness               int64
	ProbeCount              map[string]int
	BackoffEndhost          bool
	mu                      sync.Mutex // protects running
	hnCacheInit             bool
	hostnameCache           map[string]string
	rttCache                map[string]float32
	Tokens                  []*apb.IntersectionResponse
	TSDstToStampsZero       map[string]bool
	TSSrcToHopToSendSpoofed map[string]map[string]bool
	// whether this hop is thought to be responsive at all to this src
	// Since I can't intialize to true, I'm going to use an int and say 0 is true
	// anythign else will be false
	tsHopResponsive         map[string]int
	errorDetails            bytes.Buffer
	lastResponsive          string
	rrSpoofRRResponsive     map[string]int
	onAdd                   OnAddFunc
	onReach                 OnReachFunc
	onFail                  OnFailFunc
	onReachOnce, onFailOnce sync.Once
}

// OnAddFunc is the type of the callback that can be called
// when a segment is added to the ReverseTraceroute
type OnAddFunc func(*ReverseTraceroute)

// OnReachFunc is called when a reverse traceroute reaches its dst
type OnReachFunc func(*ReverseTraceroute)

// OnFailFunc is called when a reverse traceroute fails
type OnFailFunc func(*ReverseTraceroute)

var initOnce sync.Once

// NewReverseTraceroute creates a new reverse traceroute
func NewReverseTraceroute(src, dst string, id, stale uint32) *ReverseTraceroute {
	if id == 0 {
		id = rand.Uint32()
	}
	ret := ReverseTraceroute{
		ID:                      id,
		logStr:                  fmt.Sprintf("ID: %d :", id),
		Src:                     src,
		Dst:                     dst,
		Paths:                   &[]*ReversePath{NewReversePath(src, dst, nil)},
		DeadEnd:                 make(map[string]bool),
		tsHopResponsive:         make(map[string]int),
		TSDstToStampsZero:       make(map[string]bool),
		TSSrcToHopToSendSpoofed: make(map[string]map[string]bool),
		RRHop2RateLimit:         make(map[string]int),
		RRHop2VPSLeft:           make(map[string][]string),
		TSHop2RateLimit:         make(map[string]int),
		TSHop2AdjsLeft:          make(map[string][]string),
		ProbeCount:              make(map[string]int),
		StartTime:               time.Now(),
		Staleness:               int64(stale),
		rrSpoofRRResponsive:     make(map[string]int),
	}
	return &ret
}

// TSSetUnresponsive sets the dst as unresponsive to ts probes
func (rt *ReverseTraceroute) TSSetUnresponsive(dst string) {
	rt.tsHopResponsive[dst] = 1
}

// TSIsResponsive checks if the dst is responsive to ts probes
func (rt *ReverseTraceroute) TSIsResponsive(dst string) bool {
	return rt.tsHopResponsive[dst] == 0
}

// AddUnresponsiveRRSpoofer marks target dst as unresponsive.
// if the pair already exists the value is incremented by cnt.
// if  the value is already marked Responsive, this will noop
func (rt *ReverseTraceroute) AddUnresponsiveRRSpoofer(dst string, cnt int) {
	if _, ok := rt.rrSpoofRRResponsive[dst]; ok {
		if rt.rrSpoofRRResponsive[dst] == -1 {
			return
		}
		rt.rrSpoofRRResponsive[dst] = rt.rrSpoofRRResponsive[dst] + cnt
		return
	}
	rt.rrSpoofRRResponsive[dst] = cnt
}

// MarkResponsiveRRSpoofer makes the dst as responsive
func (rt *ReverseTraceroute) MarkResponsiveRRSpoofer(dst string) {
	rt.rrSpoofRRResponsive[dst] = -1
}

func (rt *ReverseTraceroute) len() int {
	return len(*(rt.Paths))
}

// SymmetricAssumptions returns the number of symmetric
// assumptions of the last path
func (rt *ReverseTraceroute) SymmetricAssumptions() int {
	return (*rt.Paths)[rt.len()-1].SymmetricAssumptions()
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
	return (*rt.Paths)[rt.len()-1]
}

// Hops gets the hops from the last path
func (rt *ReverseTraceroute) Hops() []string {
	if rt.len() == 0 {
		return []string{}
	}
	return (*rt.Paths)[rt.len()-1].Hops()
}

// LastHop gets the last hop from the last path
func (rt *ReverseTraceroute) LastHop() string {
	return (*rt.Paths)[rt.len()-1].LastHop()
}

// Reaches checks if the last path reaches
func (rt *ReverseTraceroute) Reaches() bool {
	// Assume that any path reaches if and only if the last one reaches
	if len(*rt.Paths) == 0 {
		return false
	}
	reach := (*rt.Paths)[rt.len()-1].Reaches()
	if reach {
		rt.EndTime = time.Now()
		rt.StopReason = Reaches
	}
	if reach && rt.onReach != nil {
		rt.onReachOnce.Do(func() {
			log.Debug("Calling onReach")
			rt.onReach(rt)
		})
	}
	return reach
}

// Failed returns whether we have any options lefts to explore
func (rt *ReverseTraceroute) Failed() bool {
	failed := rt.len() == 0 || (rt.BackoffEndhost && rt.len() == 1 &&
		(*rt.Paths)[0].len() == 1 && reflect.TypeOf((*(*rt.Paths)[0].Path)[0]) == reflect.TypeOf(&DstRevSegment{}))
	if failed {
		rt.EndTime = time.Now()
		rt.StopReason = Failed
	}
	if failed && rt.onFail != nil {
		rt.onFailOnce.Do(func() {
			rt.onFail(rt)
		})
	}
	return failed
}

// FailCurrPath fails the current path
func (rt *ReverseTraceroute) FailCurrPath() {
	rt.DeadEnd[rt.LastHop()] = true
	// keep popping until we find something that is either on a path
	// we are assuming symmetric (we know it started at src so goes to whole way)
	// or is not known to be a deadend
	for !rt.Failed() && rt.DeadEnd[rt.LastHop()] && reflect.TypeOf(rt.CurrPath().LastSeg()) !=
		reflect.TypeOf(&DstSymRevSegment{}) {
		// Pop
		*rt.Paths = (*rt.Paths)[:rt.len()-1]
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
	basePath := rt.CurrPath().Clone()
	basePath.Pop()
	basePath.Add(s)
	*rt.Paths = append(*rt.Paths, basePath)
	if rt.onAdd != nil {
		rt.onAdd(rt)
	}
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
	basePath := rt.CurrPath().Clone()
	for _, s := range segs {
		if !rt.DeadEnd[s.LastHop()] {
			// add loop removal here
			err := s.RemoveHops(basePath.Hops())
			if err != nil {
				log.Error(err)
				return false
			}
			if s.Length(false) == 0 {
				log.Debug("Skipping loop-causing segment ", s)
				continue
			}
			*added = true
			cl := basePath.Clone()
			cl.Add(s)
			*rt.Paths = append(*rt.Paths, cl)
		}
	}
	if *added && rt.onAdd != nil {
		rt.onAdd(rt)
	}
	return *added
}

// ToStorable returns a storble form of a ReverseTraceroute
func (rt *ReverseTraceroute) ToStorable() pb.ReverseTraceroute {
	var ret pb.ReverseTraceroute
	ret.Id = rt.ID
	ret.Src = rt.Src
	ret.Dst = rt.Dst
	ret.Runtime = rt.EndTime.Sub(rt.StartTime).Nanoseconds()
	ret.RrIssued = int32(rt.ProbeCount["rr"] + rt.ProbeCount["spoof-rr"])
	ret.TsIssued = int32(rt.ProbeCount["ts"] + rt.ProbeCount["spoof-ts"])
	ret.StopReason = string(rt.StopReason)
	if rt.StopReason != "" {
		ret.Status = pb.RevtrStatus_COMPLETED
	} else {
		ret.Status = pb.RevtrStatus_RUNNING
	}
	hopsSeen := make(map[string]bool)
	if !rt.Failed() {
		for _, s := range *rt.CurrPath().Path {
			ty := s.Type()
			for _, hi := range s.Hops() {
				if hopsSeen[hi] {
					continue
				}
				hopsSeen[hi] = true
				var h pb.RevtrHop
				h.Hop = hi
				h.Type = pb.RevtrHopType(ty)
				ret.Path = append(ret.Path, &h)
			}
		}
	}
	return ret
}

// InitializeTSAdjacents ...
func (rt *ReverseTraceroute) InitializeTSAdjacents(cls string, as types.AdjacencySource) error {
	adjs, err := getAdjacenciesForIPToSrc(cls, rt.Src, as)
	if err != nil {
		return err
	}
	rt.TSHop2AdjsLeft[cls] = adjs
	return nil
}

type byCount []types.AdjacencyToDest

func (b byCount) Len() int           { return len(b) }
func (b byCount) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byCount) Less(i, j int) bool { return b[i].Cnt < b[j].Cnt }

type aByCount []types.Adjacency

func (b aByCount) Len() int           { return len(b) }
func (b aByCount) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b aByCount) Less(i, j int) bool { return b[i].Cnt < b[j].Cnt }

func getAdjacenciesForIPToSrc(ip string, src string, as types.AdjacencySource) ([]string, error) {
	ipint, _ := util.IPStringToInt32(ip)
	srcint, _ := util.IPStringToInt32(src)
	dest24 := srcint >> 8

	ips1, err := as.GetAdjacenciesByIP1(ipint)
	if err != nil {
		return nil, err
	}
	ips2, err := as.GetAdjacenciesByIP2(ipint)
	if err != nil {
		return nil, err
	}
	adjstodst, err := as.GetAdjacencyToDestByAddrAndDest24(dest24, ipint)
	if err != nil {
		return nil, err
	}
	// Sort in descending order
	sort.Sort(sort.Reverse(byCount(adjstodst)))
	var atjs []string
	for _, adj := range adjstodst {
		ip, _ := util.Int32ToIPString(adj.Adjacent)
		atjs = append(atjs, ip)
	}
	combined := append(ips1, ips2...)
	sort.Sort(sort.Reverse(aByCount(combined)))
	var combinedIps []string
	for _, a := range combined {
		if a.IP1 == ipint {
			ips, _ := util.Int32ToIPString(a.IP2)
			combinedIps = append(combinedIps, ips)
		} else {
			ips, _ := util.Int32ToIPString(a.IP1)
			combinedIps = append(combinedIps, ips)
		}
	}
	ss := stringutil.StringSliceMinus(combinedIps, atjs)
	ss = stringutil.StringSliceMinus(ss, []string{ip})
	ret := append(atjs, ss...)
	if len(ret) < 30 {
		num := len(ret)
		return ret[:num], nil
	}
	return ret[:30], nil
}

// GetTSAdjacents get the set of adjacents to try for a hop
// for revtr:s,d,r, the set of R' left to consider, or if there are none
// will return the number that we want to probe at a time
func (rt *ReverseTraceroute) GetTSAdjacents(hop string, as types.AdjacencySource) []string {
	if _, ok := rt.TSHop2AdjsLeft[hop]; !ok {
		err := rt.InitializeTSAdjacents(hop, as)
		if err != nil {
			log.Error(err)
		}
	}
	log.Debug(rt.Src, " ", rt.Dst, " ", rt.LastHop(), " ", len(rt.TSHop2AdjsLeft[hop]), " TS adjacents left to try")

	// CASES:
	// 1. no adjacents left return nil
	if len(rt.TSHop2AdjsLeft[hop]) == 0 {
		return nil
	}
	// CAN EVENTUALLY MOVE TO REUSSING PROBES TO THIS DST FROM ANOTHER
	// REVTR, BUT NOT FOR NOW
	// 2. For now, we just take the next batch and send them
	var min int
	var rate int
	if val, ok := rt.TSHop2RateLimit[hop]; ok {
		rate = val
	} else {
		rate = 1
	}
	if rate > len(rt.TSHop2AdjsLeft[hop]) {
		min = len(rt.TSHop2AdjsLeft[hop])
	} else {
		min = rate
	}
	adjacents := rt.TSHop2AdjsLeft[hop][:min]
	rt.TSHop2AdjsLeft[hop] = rt.TSHop2AdjsLeft[hop][min:]
	return adjacents
}

// GetTimestampSpoofers gets spoofers to use for timestamp probes
func (rt *ReverseTraceroute) GetTimestampSpoofers(src, dst string, vpsource vpservice.VPSource) []string {
	var spoofers []string
	vps, err := vpsource.GetTSSpoofers(0)
	if err != nil {
		log.Error(err)
		return nil
	}
	for _, vp := range vps {
		ips, _ := util.Int32ToIPString(vp.Ip)
		spoofers = append(spoofers, ips)
	}
	return spoofers
}

const (
	// RateLimit is the number of Probes to send at once
	RateLimit = 5
)

// InitializeRRVPs initializes the rr vps for a cls
func (rt *ReverseTraceroute) InitializeRRVPs(cls string, vpsource vpservice.VPSource) error {
	log.Debug("Initializing RR VPs individually for spoofers for ", cls)
	rt.RRHop2RateLimit[cls] = RateLimit
	spoofersForTarget := []string{"non_spoofed"}
	clsi, _ := util.IPStringToInt32(cls)
	vps, err := vpsource.GetRRSpoofers(clsi, 0)
	if err != nil {
		return err
	}
	for _, vp := range vps {
		ips, _ := util.Int32ToIPString(vp.Ip)
		spoofersForTarget = append(spoofersForTarget, ips)
	}
	rt.RRHop2VPSLeft[cls] = spoofersForTarget
	return nil
}

func stringSliceIndexWithClusters(ss []string, seg string, cm clustermap.ClusterMap) int {
	for i, s := range ss {
		if cm.Get(s) == cm.Get(seg) {
			return i
		}
	}
	return -1
}

// AddBackgroundTRSegment need a different function because a TR segment might intersect
// at an IP back up the TR chain, want to delete anything that has been added along the way
func (rt *ReverseTraceroute) AddBackgroundTRSegment(trSeg Segment, cm clustermap.ClusterMap) bool {
	log.Debug("Adding Background trSegment ", trSeg)
	var found *ReversePath
	// iterate through the paths, trying to find one that contains
	// intersection point, chunk is a ReversePath
	for _, chunk := range *rt.Paths {
		var index int
		log.Debug("Looking for ", trSeg.Hops()[0], " in ", chunk.Hops())
		if index = stringSliceIndexWithClusters(chunk.Hops(), trSeg.Hops()[0], cm); index != -1 {
			log.Debug("Intersected: ", trSeg.Hops()[0], " in ", chunk)
			chunk = chunk.Clone()
			found = chunk
			// Iterate through all the segments until you find the hop
			// where they intersect. After reaching the hop where
			// they intersect, delete any subsequent hops within
			// the same segment. Then delete any segments after.
			var k int // Which IP hop we're at
			var j int // Which segment we're at
			for len(chunk.Hops())-1 > index {
				// get current segment
				seg := (*chunk.Path)[j]
				// if we're past the intersection point then delete the whole segment
				if k > index {
					*chunk.Path, (*chunk.Path)[j] = append((*chunk.Path)[:j], (*chunk.Path)[j+1:]...), nil
				} else if k+len(seg.Hops())-1 > index {
					l := stringutil.StringSliceIndex(seg.Hops(), trSeg.Hops()[0]) + 1
					for k+len(seg.Hops())-1 > index {
						seg.RemoveAt(l)
					}
				} else {
					j++
					k += len(seg.Hops())
				}
			}
			break
		}
	}
	if found == nil {
		log.Debug(trSeg)
		log.Debug("Tried to add traceroute to Reverse Traceroute that didn't share an IP... what happened?!")
		return false
	}
	*rt.Paths = append(*rt.Paths, found)
	// Now that the traceroute is cleaned up, add the new segment
	// this sequence slightly breaks how add_segment normally works here
	// we append a cloned path (with any hops past the intersection trimmed).
	// then, we call add_segment. Add_segment clones the last path,
	// then adds the segemnt to it. so we end up with an extra copy of found,
	// that might have soem hops trimmed off it. not the end of the world,
	// but something to be aware of
	success := rt.AddSegments([]Segment{trSeg})
	if !success {
		for i, s := range *rt.Paths {
			if found == s {
				*rt.Paths, (*rt.Paths)[rt.len()-1] = append((*rt.Paths)[:i], (*rt.Paths)[i+1:]...), nil
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
func (rt *ReverseTraceroute) GetRRVPs(dst string, vps vpservice.VPSource) ([]string, string) {
	log.Debug("GettingRRVPs for ", dst)
	// we either use destination or cluster, depending on how flag is set
	hops := rt.CurrPath().LastSeg().Hops()
	for _, hop := range hops {
		if _, ok := rt.RRHop2VPSLeft[hop]; !ok {
			rt.InitializeRRVPs(hop, vps)
		}
	}
	// CASES:
	segHops := stringutil.CloneStringSlice(rt.CurrPath().LastSeg().Hops())
	log.Debug("segHops: ", segHops)
	var target, cls *string
	target = new(string)
	cls = new(string)
	var foundVPs bool
	for !foundVPs && len(segHops) > 0 {
		*target, segHops = segHops[len(segHops)-1], segHops[:len(segHops)-1]
		*cls = *target
		log.Debug("Sending RR probes to: ", *cls)
		log.Debug("RR VPS: ", rt.RRHop2VPSLeft[*cls])
		// 0. destination seems to be unresponsive
		if rt.rrSpoofRRResponsive[*cls] != -1 &&
			rt.rrSpoofRRResponsive[*cls] >= maxUnresponsive {
			// this may not match exactly but I think it does
			log.Debug("GetRRVPs: unresponsive for: ", *cls)
			continue
		}
		// 1. no VPs left, return nil
		if len(rt.RRHop2VPSLeft[*cls]) == 0 {
			log.Debug("GetRRVPs: No VPs left for: ", *cls)
			continue
		}
		foundVPs = true
	}
	if !foundVPs {
		return nil, ""
	}
	log.Debug(rt.Src, " ", rt.Dst, " ", *target, " ", len(rt.RRHop2VPSLeft[*cls]), " RR VPs left to try")
	// 2. send non-spoofed version if it is in the next batch
	min := rt.RRHop2RateLimit[*cls]
	if len(rt.RRHop2VPSLeft[*cls]) < min {
		min = len(rt.RRHop2VPSLeft[*cls])
	}
	log.Debug("Getting vps for: ", *cls, " min: ", min)
	if stringutil.InArray(rt.RRHop2VPSLeft[*cls][0:min], "non_spoofed") {
		rt.RRHop2VPSLeft[*cls] = rt.RRHop2VPSLeft[*cls][1:]
		return []string{"non_spoofed"}, *target
	}

	// 3. use unused spoofing VPs
	// if the current last hop was discovered with spoofed, and it
	// hasn't been used yet, use it
	notEmpty := rt.len() > 0
	var isRRRev *bool
	isRRRev = new(bool)
	spoofer := new(string)
	if rrev, ok := rt.CurrPath().LastSeg().(*SpoofRRRevSegment); ok {
		*isRRRev = true
		*spoofer = rrev.SpoofSource
	}
	if notEmpty && *isRRRev {
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
	touse := rt.RRHop2VPSLeft[*cls][:min]
	rt.RRHop2VPSLeft[*cls] = rt.RRHop2VPSLeft[*cls][min:]
	log.Debug("Returning VPS for spoofing: ", touse)
	return touse, *target
}

var (
	ipToCluster clustermap.ClusterMap
)

// CreateReverseTraceroute creates a reverse traceroute for the web interface
func CreateReverseTraceroute(revtr pb.RevtrMeasurement, cs types.ClusterSource,
	onAdd OnAddFunc, onFail OnFailFunc, onReach OnReachFunc) *ReverseTraceroute {
	initOnce.Do(func() {
		ipToCluster = clustermap.New(cs)
	})
	rt := NewReverseTraceroute(revtr.Src, revtr.Dst, revtr.Id, revtr.Staleness)
	rt.BackoffEndhost = revtr.BackoffEndhost
	rt.onAdd = onAdd
	rt.onFail = onFail
	rt.onReach = onReach
	return rt
}
