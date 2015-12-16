package revtr

import (
	"fmt"
	"reflect"

	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
)

var plHost2IP map[string]string
var ipToCluster map[string]int

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
	RRHop2VPSLeft   map[string][]*datamodel.VantagePoint
	TSHop2RateLimit map[string]int
	TSHop2AdjsLeft  map[string][]Adjacency
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

func (rt *ReverseTraceroute) setRRVPSForHop(hop string, vps []*datamodel.VantagePoint) {
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
	// Original code has a whole lot of logic here just to print some logs
	// For now i'm going to ignore
}

// AddAndReplaceSegment adds a new path, equal to the current one but with the last
// segment replaced by the new one
// returns a bool of whether it was added
// might not be added if it is a deadend
func (rt *ReverseTraceroute) AddAndReplaceSegment(s Segment) bool {
	if rt.DeadEnd[s.LastHop()] {
		return false
	}
}
