package reversetraceroute

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"net"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/context"

	at "github.com/NEU-SNS/ReverseTraceroute/atlas/client"
	apb "github.com/NEU-SNS/ReverseTraceroute/atlas/pb"
	"github.com/NEU-SNS/ReverseTraceroute/controller/client"
	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/ip_utils"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/pb"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/types"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	vpservice "github.com/NEU-SNS/ReverseTraceroute/vpservice/client"
)

var plHost2IP map[string]string

var ipToCluster clusterMap
var clusterToIps map[string][]string
var tsAdjsByCluster bool
var vps []*datamodel.VantagePoint

var rrVPsByCluster bool

const (
	// RateLimit is the ReverseTraceroute Rate Limit
	RateLimit int = 5
)

type multiError []error

func (m multiError) Error() string {
	var buf bytes.Buffer
	for _, e := range m {
		buf.WriteString(e.Error() + "\n")
	}
	return buf.String()
}

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
	ID                       uint32
	logStr                   string
	Paths                    *[]*ReversePath
	DeadEnd                  map[string]bool
	RRHop2RateLimit          map[string]int
	RRHop2VPSLeft            map[string][]string
	TSHop2RateLimit          map[string]int
	TSHop2AdjsLeft           map[string][]string
	Src, Dst                 string
	StopReason               string
	StartTime, EndTime       time.Time
	Staleness                int64
	ProbeCount               map[string]int
	as                       types.AdjacencySource
	backoffEndhost           bool
	cl                       client.Client
	at                       at.Atlas
	print                    bool
	running                  bool
	mu                       sync.Mutex // protects running
	hostnameCache            map[string]string
	rttCache                 map[string]float32
	tokens                   []*apb.IntersectionResponse
	rrsSrcToDstToVPToRevHops map[string]map[string]map[string][]string
	trsSrcToDstToPath        map[string]map[string][]string
	tsSrcToProbeToVPToResult map[string]map[string]map[string][]string
	tsDstToStampsZero        map[string]bool
	tsSrcToHopToSendSpoofed  map[string]map[string]bool
	// whether this hop is thought to be responsive at all to this src
	// Since I can't intialize to true, I'm going to use an int and say 0 is true
	// anythign else will be false
	tsSrcToHopToResponsive map[string]map[string]int
	errorDetails           bytes.Buffer
	lastResponsive         string
	vps                    vpservice.VPSource
	opc                    chan Status
}

var initOnce sync.Once

// NewReverseTraceroute creates a new reverse traceroute
func NewReverseTraceroute(src, dst string, id, stale uint32, as types.AdjacencySource) *ReverseTraceroute {
	if id == 0 {
		id = rand.Uint32()
	}
	ret := ReverseTraceroute{
		ID:                       id,
		logStr:                   fmt.Sprintf("ID: %d :", id),
		Src:                      src,
		Dst:                      dst,
		Paths:                    &[]*ReversePath{NewReversePath(src, dst, nil)},
		DeadEnd:                  make(map[string]bool),
		tsSrcToHopToResponsive:   make(map[string]map[string]int),
		tsDstToStampsZero:        make(map[string]bool),
		tsSrcToHopToSendSpoofed:  make(map[string]map[string]bool),
		RRHop2RateLimit:          make(map[string]int),
		RRHop2VPSLeft:            make(map[string][]string),
		TSHop2RateLimit:          make(map[string]int),
		TSHop2AdjsLeft:           make(map[string][]string),
		ProbeCount:               make(map[string]int),
		as:                       as,
		StartTime:                time.Now(),
		Staleness:                int64(stale),
		hostnameCache:            make(map[string]string),
		rttCache:                 make(map[string]float32),
		rrsSrcToDstToVPToRevHops: make(map[string]map[string]map[string][]string),
		trsSrcToDstToPath:        make(map[string]map[string][]string),
		tsSrcToProbeToVPToResult: make(map[string]map[string]map[string][]string),
		opc: make(chan Status, 5),
	}
	return &ret
}

// GetOutputChan retreives the output channel of the ReverseTraceroute
func (rt *ReverseTraceroute) GetOutputChan() <-chan Status {
	return rt.opc
}

func (rt *ReverseTraceroute) debug(args ...interface{}) {
	var nargs []interface{}
	nargs = append(nargs, rt.logStr)
	nargs = append(nargs, args...)
	log.Debug(nargs...)
}
func (rt *ReverseTraceroute) debugf(s string, args ...interface{}) {
	news := rt.logStr + s
	log.Debugf(news, args...)
}
func (rt *ReverseTraceroute) info(args ...interface{}) {
	var nargs []interface{}
	nargs = append(nargs, rt.logStr)
	nargs = append(nargs, args...)
	log.Info(nargs...)
}
func (rt *ReverseTraceroute) infof(s string, args ...interface{}) {
	news := rt.logStr + s
	log.Infof(news, args...)
}
func (rt *ReverseTraceroute) warn(args ...interface{}) {
	var nargs []interface{}
	nargs = append(nargs, rt.logStr)
	nargs = append(nargs, args...)
	log.Warn(nargs...)
}
func (rt *ReverseTraceroute) warnf(s string, args ...interface{}) {
	news := rt.logStr + s
	log.Warnf(news, args...)
}
func (rt *ReverseTraceroute) error(args ...interface{}) {
	var nargs []interface{}
	nargs = append(nargs, rt.logStr)
	nargs = append(nargs, args...)
	log.Error(nargs...)
}
func (rt *ReverseTraceroute) errorf(s string, args ...interface{}) {
	news := rt.logStr + s
	log.Errorf(news, args...)
}

// Status represents the current running state of a reverse traceroute
// it is use for the web interface. Something better is probably needed
type Status struct {
	Rep    string
	Status bool
	Error  string
}

func (rt *ReverseTraceroute) output() {
	st := fmt.Sprintf("%s\n%s", rt.HTML(), rt.StopReason)
	var showError = !rt.Reaches()
	var errorText string
	if showError {
		errorText = strings.Replace(rt.errorDetails.String(), "\n", "<br>", -1)
	}
	var stat Status
	stat.Rep = strings.Replace(st, "\n", "<br>", -1)
	stat.Status = rt.Reaches() || rt.StopReason != ""
	stat.Error = errorText
	select {
	case rt.opc <- stat:
	default:
	}
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
	return (*rt.Paths)[rt.len()-1].Reaches()
}

// Failed returns whether we have any options lefts to explore
func (rt *ReverseTraceroute) Failed(backoffEndhost bool) bool {
	return rt.len() == 0 || (backoffEndhost && rt.len() == 1 &&
		(*rt.Paths)[0].len() == 1 && reflect.TypeOf((*(*rt.Paths)[0].Path)[0]) == reflect.TypeOf(&DstRevSegment{}))
}

// FailCurrPath fails the current path
func (rt *ReverseTraceroute) FailCurrPath() {
	rt.DeadEnd[rt.LastHop()] = true
	// keep popping until we find something that is either on a path
	// we are assuming symmetric (we know it started at src so goes to whole way)
	// or is not known to be a deadend
	for !rt.Failed(rt.backoffEndhost) && rt.DeadEnd[rt.LastHop()] && reflect.TypeOf(rt.CurrPath().LastSeg()) !=
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

	if rt.print {
		rt.output()
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
				rt.error(err)
				return false
			}
			if s.Length(false) == 0 {
				rt.debug("Skipping loop-causing segment ", s)
				continue
			}
			*added = true
			cl := basePath.Clone()
			cl.Add(s)
			*rt.Paths = append(*rt.Paths, cl)
		}
	}
	if *added && rt.print {
		rt.output()
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
	ret.StopReason = rt.StopReason
	if rt.StopReason != "" {
		ret.Status = pb.RevtrStatus_COMPLETED
	} else {
		ret.Status = pb.RevtrStatus_RUNNING
	}
	hopsSeen := make(map[string]bool)
	if !rt.Failed(rt.backoffEndhost) {
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

// HTML creates the html output for a ReverseTraceroute
func (rt *ReverseTraceroute) HTML() string {
	//need to find hostnames and rtts
	hopsSeen := make(map[string]bool)
	var out bytes.Buffer
	out.WriteString(`<table class="table">`)
	out.WriteString(`<caption class="text-center">Reverse Traceroute from `)
	if len(rt.Hops()) >= 1 {
		out.WriteString(fmt.Sprintf("%s (%s) back to ", rt.Hops()[0], rt.resolveHostname(rt.Hops()[0])))
	}
	out.WriteString(rt.Src)
	out.WriteString(fmt.Sprintf(" (%s)", rt.resolveHostname(rt.Src)))
	out.WriteString("</caption>")
	out.WriteString(`<tbody>`)
	first := true
	var i int
	if len(*rt.Paths) > 0 {

		for _, segment := range *rt.CurrPath().Path {
			symbol := new(string)
			switch segment.(type) {
			case *DstSymRevSegment:
				*symbol = "sym"
			case *DstRevSegment:
				*symbol = "dst"
			case *TRtoSrcRevSegment:
				*symbol = "tr"
			case *SpoofRRRevSegment:
				*symbol = "rr"
			case *RRRevSegment:
				*symbol = "rr"
			case *SpoofTSAdjRevSegmentTSZeroDoubleStamp:
				*symbol = "ts"
			case *SpoofTSAdjRevSegmentTSZero:
				*symbol = "ts"
			case *SpoofTSAdjRevSegment:
				*symbol = "ts"
			case *TSAdjRevSegment:
				*symbol = "ts"
			}
			for _, hop := range segment.Hops() {
				if hopsSeen[hop] {
					continue
				}
				hopsSeen[hop] = true
				tech := new(string)
				if first {
					*tech = *symbol
					first = false
				} else {
					*tech = "-" + *symbol
				}
				if hop == "0.0.0.0" || hop == "*" {
					out.WriteString(fmt.Sprintf("<tr><td>%-2d</td><td>%-80s</td><td></td><td>%s</td></tr>", i, "* * *", *tech))
				} else {
					out.WriteString(fmt.Sprintf("<tr><td>%-2d</td><td>%-80s (%s)</td><td>%.3fms</td><td>%s</td></tr>", i, hop, rt.resolveHostname(hop), rt.getRTT(hop), *tech))
				}
				i++
			}
		}
	}
	out.WriteString("</tbody></table>")
	return out.String()
}

func (rt *ReverseTraceroute) resolveHostname(ip string) string {
	hn, ok := rt.hostnameCache[ip]
	if !ok {
		for _, vp := range vps {
			ips, _ := util.Int32ToIPString(vp.Ip)
			if ips == ip {
				rt.hostnameCache[ip] = vp.Hostname
				return vp.Hostname
			}
		}
		hns, err := net.LookupAddr(ip)
		if err != nil {
			rt.error(err)
			rt.hostnameCache[ip] = ""
			hn = ""
		} else {
			if len(hns) == 0 {
				rt.hostnameCache[ip] = ""
			} else {
				rt.hostnameCache[ip] = hns[0]
				hn = hns[0]
			}
		}
	}
	return hn
}

func (rt *ReverseTraceroute) getRTT(ip string) float32 {
	rtt, ok := rt.rttCache[ip]
	if ok {
		return rtt
	}
	targ, _ := util.IPStringToInt32(ip)
	src, _ := util.IPStringToInt32(rt.Src)
	ping := &datamodel.PingMeasurement{
		Src:     src,
		Dst:     targ,
		Count:   "1",
		Timeout: 10,
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	st, err := rt.cl.Ping(ctx, &datamodel.PingArg{Pings: []*datamodel.PingMeasurement{ping}})
	if err != nil {
		rt.error(err)
		rt.rttCache[ip] = 0
		return 0
	}
	for {
		p, err := st.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			rt.error(err)
			rt.rttCache[ip] = 0
			break
		}
		if len(p.Responses) == 0 {
			rt.rttCache[ip] = 0
			break
		}
		rt.debug(p)
		rt.rttCache[ip] = float32(p.Responses[0].Rtt) / 1000
	}
	return rt.rttCache[ip]
}

type adjSettings struct {
	timeout      int
	maxnum       int
	maxalert     string
	retryCommand bool
}

func defaultAdjSettings() *adjSettings {
	ret := adjSettings{
		maxnum: 30,
	}
	return &ret
}

type byCount []types.AdjacencyToDest

func (b byCount) Len() int           { return len(b) }
func (b byCount) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byCount) Less(i, j int) bool { return b[i].Cnt < b[j].Cnt }

type aByCount []types.Adjacency

func (b aByCount) Len() int           { return len(b) }
func (b aByCount) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b aByCount) Less(i, j int) bool { return b[i].Cnt < b[j].Cnt }

func getAdjacenciesForIPToSrc(ip string, src string, as types.AdjacencySource, settings *adjSettings) ([]string, error) {
	if settings == nil {
		settings = defaultAdjSettings()
	}
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
	ss := stringSliceMinus(combinedIps, atjs)
	ss = stringSliceMinus(ss, []string{ip})
	ret := append(atjs, ss...)
	var mi = &settings.maxnum
	if len(ret) < settings.maxnum {
		*mi = len(ret)
	}
	return ret[:*mi], nil
}

// InitializeTSAdjacents ...
func (rt *ReverseTraceroute) InitializeTSAdjacents(cls string) error {
	adjs, err := getAdjacenciesForIPToSrc(cls, rt.Src, rt.as, nil)
	if err != nil {
		return err
	}
	var cleaned []string
	for _, ip := range adjs {
		if cls != ipToCluster.Get(ip) {
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
		cls = ipToCluster.Get(hop)
	} else {
		cls = hop
	}
	if _, ok := rt.TSHop2AdjsLeft[cls]; !ok {
		rt.InitializeTSAdjacents(cls)
	}
	rt.debug(rt.Src, " ", rt.Dst, " ", rt.LastHop(), " ", len(rt.TSHop2AdjsLeft[cls]), " TS adjacents left to try")

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
		rt.debug("vps: ", vps)
	} else {
		min = rate
	}
	adjacents := rt.TSHop2AdjsLeft[cls][:min]
	rt.TSHop2AdjsLeft[cls] = rt.TSHop2AdjsLeft[cls][min:]
	return adjacents
}

// for now we're just ignoring the src dst and choosing randomly
func (rt *ReverseTraceroute) getTimestampSpoofers(src, dst string) []string {
	var spoofers []string
	vps, err := rt.vps.GetTSSpoofers(0)
	if err != nil {
		rt.error(err)
		return nil
	}
	for _, vp := range vps {
		ips, _ := util.Int32ToIPString(vp.Ip)
		spoofers = append(spoofers, ips)
	}
	return spoofers
}

// InitializeRRVPs initializes the rr vps for a cls
func (rt *ReverseTraceroute) InitializeRRVPs(cls string) error {
	rt.debug("Initializing RR VPs individually for spoofers for ", cls)
	rt.RRHop2RateLimit[cls] = RateLimit
	spoofersForTarget := []string{"non_spoofed"}
	clsi, _ := util.IPStringToInt32(cls)
	vps, err := rt.vps.GetRRSpoofers(clsi, 0)
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

func cloneStringSlice(ss []string) []string {
	var ret []string
	for _, s := range ss {
		ret = append(ret, s)
	}
	return ret
}

var batchInitRRVPs = true
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
	foundSpoofed := new(bool)
	for key := range old {
		if key == "non_spoofed" {
			*foundSpoofed = true
			continue
		}
		ret = append(ret, key)
	}
	if *foundSpoofed {
		ret = append([]string{"non_spoofed"}, ret...)
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

func stringSliceIndexWithClusters(ss []string, seg string) int {
	for i, s := range ss {
		if ipToCluster.Get(s) == ipToCluster.Get(seg) {
			return i
		}
	}
	return -1
}

// AddBackgroundTRSegment need a different function because a TR segment might intersect
// at an IP back up the TR chain, want to delete anything that has been added along the way
// THIS IS ALMOST DEFINITLY WRONG AND WILL NEED DEBUGGING
func (rt *ReverseTraceroute) AddBackgroundTRSegment(trSeg Segment) bool {
	rt.debug("Adding Background trSegment ", trSeg)
	var found *ReversePath
	// iterate through the paths, trying to find one that contains
	// intersection point, chunk is a ReversePath
	for _, chunk := range *rt.Paths {
		var index int
		rt.debug("Looking for ", trSeg.Hops()[0], " in ", chunk.Hops())
		if index = stringSliceIndexWithClusters(chunk.Hops(), trSeg.Hops()[0]); index != -1 {
			rt.debug("Intersected: ", trSeg.Hops()[0], " in ", chunk)
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
					l := stringSliceIndex(seg.Hops(), trSeg.Hops()[0]) + 1
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
		rt.debug(trSeg)
		rt.debug("Tried to add traceroute to Reverse Traceroute that didn't share an IP... what happened?!")
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
func (rt *ReverseTraceroute) GetRRVPs(dst string) ([]string, string) {
	rt.debug("GettingRRVPs for ", dst)
	// we either use destination or cluster, depending on how flag is set
	hops := rt.CurrPath().LastSeg().Hops()
	for _, hop := range hops {
		cls := &hop
		if rrVPsByCluster {
			*cls = ipToCluster.Get(hop)
		}
		if _, ok := rt.RRHop2VPSLeft[*cls]; !ok {
			rt.InitializeRRVPs(*cls)
		}
	}
	// CASES:
	segHops := cloneStringSlice(rt.CurrPath().LastSeg().Hops())
	rt.debug("segHops: ", segHops)
	var target, cls *string
	target = new(string)
	cls = new(string)
	var foundVPs bool
	for !foundVPs && len(segHops) > 0 {
		*target, segHops = segHops[len(segHops)-1], segHops[:len(segHops)-1]
		*cls = *target
		if rrVPsByCluster {
			*cls = ipToCluster.Get(*target)
		}
		rt.debug("Sending RR probes to: ", *cls)
		rt.debug("RR VPS: ", rt.RRHop2VPSLeft[*cls])
		var vals [][]string
		for _, val := range rt.rrsSrcToDstToVPToRevHops[rt.Src][*cls] {
			rt.debug("Found old values: ", val)
			if len(val) > 0 {
				vals = append(vals, val)
			}
		}
		// 0. destination seems to be unresponsive
		if len(rt.rrsSrcToDstToVPToRevHops[rt.Src][*cls]) >= maxUnresponsive &&
			// this may not match exactly but I think it does
			len(vals) == 0 {
			rt.debug("GetRRVPs: unresponsive for: ", *cls)
			continue
		}
		// 1. no VPs left, return nil
		if len(rt.RRHop2VPSLeft[*cls]) == 0 {
			rt.debug("GetRRVPs: No VPs left for: ", *cls)
			continue
		}
		foundVPs = true
	}
	if !foundVPs {
		return nil, ""
	}
	rt.debug(rt.Src, " ", rt.Dst, " ", *target, " ", len(rt.RRHop2VPSLeft[*cls]), " RR VPs left to try")
	// 2. probes to this dst that were already issues for other reverse
	// traceroutes, but not in this reverse traceroute
	var keys []string
	tmp := rt.rrsSrcToDstToVPToRevHops[rt.Src][*cls]
	for k := range tmp {
		keys = append(keys, k)
	}
	rt.debug("Keys: ", keys, " vpsleft: ", rt.RRHop2VPSLeft[*cls])
	usedVps := stringSet(keys).union(stringSet(rt.RRHop2VPSLeft[*cls]))
	rt.RRHop2VPSLeft[*cls] = stringSliceMinus(rt.RRHop2VPSLeft[*cls], usedVps)
	var finalUsedVPs []string
	for _, uvp := range usedVps {
		idk, ok := rt.rrsSrcToDstToVPToRevHops[rt.Src][*cls][uvp]
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
	rt.debug("Getting vps for: ", *cls, " min: ", min)
	if stringInSlice(rt.RRHop2VPSLeft[*cls][0:min], "non_spoofed") {
		rt.RRHop2VPSLeft[*cls] = rt.RRHop2VPSLeft[*cls][1:]
		return []string{"non_spoofed"}, *target
	}

	// 4. use unused spoofing VPs
	// if the current last hop was discovered with spoofed, and it
	// hasn't been used yet, use it
	notEmpty := rt.len() > 0
	var isRRRev, containsKey *bool
	isRRRev = new(bool)
	containsKey = new(bool)
	spoofer := new(string)
	if rrev, ok := rt.CurrPath().LastSeg().(*SpoofRRRevSegment); ok {
		*isRRRev = true
		*spoofer = rrev.SpoofSource
		if _, ok := rt.rrsSrcToDstToVPToRevHops[rt.Src][*cls][rrev.SpoofSource]; ok {
			*containsKey = true
		}
	}
	if notEmpty && *isRRRev && !*containsKey {
		rt.debug("Found recent spoofer to use ", *spoofer)
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
	rt.debug("Returning VPS for spoofing: ", vps)
	return vps, *target
}

// CreateReverseTraceroute creates a reverse traceroute for the web interface
func CreateReverseTraceroute(revtr pb.RevtrMeasurement, backoffEndhost, print bool, cl client.Client, at at.Atlas, vpserv vpservice.VPSource, as types.AdjacencySource, cs types.ClusterSource) *ReverseTraceroute {
	initOnce.Do(func() {
		ipToCluster = newClusterMap(cs)
	})
	rt := NewReverseTraceroute(revtr.Src, revtr.Dst, revtr.Id, revtr.Staleness, as)
	rt.backoffEndhost = backoffEndhost
	rt.print = print
	rt.cl = cl
	rt.at = at
	rt.vps = vpserv
	return rt
}

// IsRunning returns true if the ReverseTraceroute is running
func (rt *ReverseTraceroute) IsRunning() bool {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.running
}

// Run runs the ReverseTraceroute
func (rt *ReverseTraceroute) Run() error {
	rt.mu.Lock()
	rt.running = true
	rt.mu.Unlock()
	defer func() {
		rt.output()
		close(rt.opc)
	}()
	rt.output()
	if rt.backoffEndhost || dstMustBeReachable {
		err := rt.reverseHopsAssumeSymmetric()
		if err != nil {
			rt.debug("Backoff Endhost failed")
			rt.StopReason = "FAILED"
			rt.EndTime = time.Now()
			rt.debug(rt.StopReason)
			return err
		}
		if rt.Reaches() {
			rt.StopReason = "TRIVIAL"
			rt.EndTime = time.Now()
			return nil
		}
		rt.debug("Done backing off")
	}
	for {
		rt.debug(*rt.Paths)
		rt.debug(rt.CurrPath())
		rt.debug("Attempting to find TR")
		err := rt.reverseHopsTRToSrc()
		if rt.Reaches() {
			rt.EndTime = time.Now()
			rt.StopReason = "REACHES"
			return nil
		}
		if err == nil {
			err = rt.revtreiveBackgroundTRS()
			if rt.Reaches() {
				rt.EndTime = time.Now()
				rt.StopReason = "REACHES"
				return nil
			}
			if err == nil {
				continue
			}
			log.Error(err)
		}
		rt.debug(err)
		rt.debug("Attempting RR")
		err = nil
		for err != ErrNoVPs {
			err = rt.reverseHopsRR()
			if rt.Reaches() {
				rt.EndTime = time.Now()
				rt.StopReason = "REACHES"
				return nil
			}
			if err == nil {
				break
			}
		}
		if err == nil {
			err = rt.revtreiveBackgroundTRS()
			if rt.Reaches() {
				rt.EndTime = time.Now()
				rt.StopReason = "REACHES"
				return nil
			}
			if err == nil {
				continue
			}
			log.Error(err)
		}
		rt.debug("Attempting TS")
		err = nil
		for err != ErrNoVPs && err != ErrNoAdj {
			err = rt.reverseHopsTS()
			if rt.Reaches() {
				rt.StopReason = "REACHES"
				rt.EndTime = time.Now()
				return nil
			}
			if err == nil {
				break
			}
		}
		if err == nil {
			err = rt.revtreiveBackgroundTRS()
			if rt.Reaches() {
				rt.EndTime = time.Now()
				rt.StopReason = "REACHES"
				return nil
			}
			if err == nil {
				continue
			}
			log.Error(err)
		}
		rt.debug("Attempting to add from background traceroute")
		err = rt.revtreiveBackgroundTRS()
		if rt.Reaches() {
			rt.EndTime = time.Now()
			rt.StopReason = "REACHES"
			return nil
		}
		if err == nil {
			continue
		}
		rt.debug(err)
		rt.debug("Assuming Symmetric")
		err = rt.reverseHopsAssumeSymmetric()
		if rt.Reaches() {
			rt.StopReason = "REACHES"
			rt.EndTime = time.Now()
			return nil
		}
		if err == ErrNoHopFound {
			rt.FailCurrPath()
			if rt.Failed(rt.backoffEndhost) {
				rt.StopReason = "FAILED"
				rt.EndTime = time.Now()
				_, err := rt.errorDetails.WriteString("All techniques failed to find a hop.\n")
				if err != nil {
					rt.error(err)
				}
				return err
			}
		}
	}
}

var dstMustBeReachable = false

type stringInt int

func (si stringInt) String() string {
	return fmt.Sprintf("%d", int(si))
}

// RunReverseTraceroute runs a reverse traceroute
func RunReverseTraceroute(revtr pb.RevtrMeasurement, cl client.Client, at at.Atlas, vpserv vpservice.VPSource, as types.AdjacencySource, cs types.ClusterSource) (*ReverseTraceroute, error) {
	initOnce.Do(func() {
		ipToCluster = newClusterMap(cs)
	})
	rt := NewReverseTraceroute(revtr.Src, revtr.Dst, revtr.Id, revtr.Staleness, as)
	rt.backoffEndhost = revtr.BackoffEndhost
	rt.cl = cl
	rt.at = at
	return rt, rt.Run()
}

func stringInSlice(ss []string, s string) bool {
	for _, item := range ss {
		if s == item {
			return true
		}
	}
	return false
}

var (
	// ErrNoVPs is used when there are no vps left
	ErrNoVPs = fmt.Errorf("No VPs")
	// ErrNoHopFound is used when no hop is found from a measurement round
	ErrNoHopFound = fmt.Errorf("No Hop Found")
	// ErrNoAdj is used when there are no adjacents found
	ErrNoAdj = fmt.Errorf("No Adjacents found")
	// ErrPrivateIP is used what the requested target is a private IP addr
	ErrPrivateIP = fmt.Errorf("The target is a private IP addr")
)

func (rt *ReverseTraceroute) reverseHopsRR() error {
	vps, target := rt.GetRRVPs(rt.LastHop())
	rt.debug("reverseHopsRR vps: ", vps)
	receiverToSpooferToTarget := make(map[string]map[string][]string)
	init := func(s string) {
		if _, ok := receiverToSpooferToTarget[s]; ok {
			return
		}
		receiverToSpooferToTarget[s] = make(map[string][]string)
	}
	var pings []*datamodel.PingMeasurement
	if len(vps) == 0 {
		return ErrNoVPs
	}
	var cls *string
	cls = new(string)
	if rrVPsByCluster {
		*cls = ipToCluster.Get(target)
	} else {
		*cls = target
	}
	var keys []string
	for k := range rt.rrsSrcToDstToVPToRevHops[rt.Src][*cls] {
		keys = append(keys, k)
	}
	rt.debug("VPS ", vps)
	vps = stringSliceMinus(vps, keys)
	vpsc := cloneStringSlice(vps)
	if stringInSlice(vps, "non_spoofed") {
		var nvps []string
		for _, vp := range vps {
			if vp != "non_spoofed" {
				nvps = append(nvps)
			}
		}
		vps = nvps
		srcs, _ := util.IPStringToInt32(rt.Src)
		dsts, _ := util.IPStringToInt32(target)

		if !iputil.IsPrivate(net.ParseIP(target)) {
			pings = append(pings, &datamodel.PingMeasurement{
				Src:        srcs,
				Dst:        dsts,
				RR:         true,
				Timeout:    30,
				Count:      "1",
				CheckDb:    true,
				CheckCache: true,
				Staleness:  rt.Staleness,
			})
		}
	}
	for _, vp := range vps {
		init(rt.Src)
		receiverToSpooferToTarget[rt.Src][vp] = append(receiverToSpooferToTarget[rt.Src][vp], target)
	}
	if len(pings) == 1 {
		rt.issueRecordRoutes(pings[0])
	}
	if len(receiverToSpooferToTarget) > 0 {
		rt.issueSpoofedRecordRoutes(receiverToSpooferToTarget, true)
	}
	var segs []Segment
	if rrVPsByCluster {
		target = ipToCluster.Get(target)
	}
	for _, vp := range vpsc {
		rt.debug("Creating Segs for RR hops for src ", rt.Src, " target ", target, " vp ", vp)
		hops := rt.rrsSrcToDstToVPToRevHops[rt.Src][target][vp]
		rt.debug("Trying to use hops ", hops)
		if len(hops) > 0 {
			// for every non-zero hop, build a revsegment
			for i, hop := range hops {
				if hop == "0.0.0.0" {
					continue
				}
				// i+1 otherwise the last hop will never be considered?
				if vp == "non_spoofed" {
					segs = append(segs, NewRRRevSegment(hops[:i+1], rt.Src, target))
				} else {
					segs = append(segs, NewSpoofRRRevSegment(hops[:i+1], rt.Src, target, vp))
				}
			}
		}
	}
	if !rt.AddSegments(segs) {
		return ErrNoHopFound
	}
	return nil
}

func (rt *ReverseTraceroute) issueSpoofedRecordRoutes(recvToSpooferToTarget map[string]map[string][]string, deleteUnresponsive bool) error {
	rt.debug("Issuing Spoofed RR probes ", recvToSpooferToTarget)
	var pings []*datamodel.PingMeasurement
	for rec, spoofToTarg := range recvToSpooferToTarget {
		for spoofer, targets := range spoofToTarg {
			for _, target := range targets {
				if iputil.IsPrivate(net.ParseIP(target)) {
					continue
				}
				sspoofer, _ := util.IPStringToInt32(spoofer)
				sdst, _ := util.IPStringToInt32(target)
				pings = append(pings, &datamodel.PingMeasurement{
					Spoof:     true,
					RR:        true,
					SAddr:     rec,
					Src:       sspoofer,
					Dst:       sdst,
					Timeout:   10,
					Count:     "1",
					Staleness: rt.Staleness,
				})
			}
		}
	}
	rt.ProbeCount["spoof-rr"] += len(pings)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	st, err := rt.cl.Ping(ctx, &datamodel.PingArg{
		Pings: pings,
	})
	if err != nil {
		return err
	}
	// initalize all entires so logic for max unresponsive works
	for rec, spoofToTarg := range recvToSpooferToTarget {
		inner, ok := rt.rrsSrcToDstToVPToRevHops[rec]
		if ok {
			for spoof, targs := range spoofToTarg {
				for _, targ := range targs {
					in, ok := inner[targ]
					if ok {
						in[spoof] = nil
					} else {
						inner[targ] = make(map[string][]string)
						inner[targ][spoof] = nil
					}
				}
			}
		} else {
			tmp := make(map[string]map[string][]string)
			for spoof, targs := range spoofToTarg {
				for _, targ := range targs {
					in, ok := tmp[targ]
					if ok {
						in[spoof] = nil
					} else {
						tmp[targ] = make(map[string][]string)
						tmp[targ][spoof] = nil
					}
				}
			}
			rt.rrsSrcToDstToVPToRevHops[rec] = tmp
		}
	}
	rt.debug(rt.rrsSrcToDstToVPToRevHops)
	for {
		p, err := st.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		rt.debug("Got spoofed RR response: ", p)
		pr := p.GetResponses()
		ssrc, _ := util.Int32ToIPString(p.Src)
		sdst, _ := util.Int32ToIPString(p.Dst)
		if len(pr) > 0 {
			sspoofer, _ := util.Int32ToIPString(p.SpoofedFrom)
			rrs := pr[0].RR
			cls := new(string)
			if rrVPsByCluster {
				*cls = ipToCluster.Get(sdst)
			} else {
				*cls = sdst
			}

			inner, ok := rt.rrsSrcToDstToVPToRevHops[ssrc]
			if ok {
				f, ok := inner[sdst]
				if ok {
					f[sspoofer] = processRR(ssrc, sdst, rrs, true)
				} else {
					inner[sdst] = make(map[string][]string)
					inner[sdst][sspoofer] = processRR(ssrc, sdst, rrs, true)
				}
			} else {
				tmp := make(map[string]map[string][]string)
				tmp[sdst] = make(map[string][]string)
				tmp[sdst][sspoofer] = processRR(ssrc, sdst, rrs, true)
				rt.rrsSrcToDstToVPToRevHops[ssrc] = tmp

			}
		} else {
			inner, ok := rt.rrsSrcToDstToVPToRevHops[ssrc]
			if ok {
				_, ok := inner[sdst]
				if !ok {
					inner[sdst] = make(map[string][]string)
				} else {
				}
			} else {
				tmp := make(map[string]map[string][]string)
				tmp[sdst] = make(map[string][]string)

			}
		}
	}
	return nil
}
