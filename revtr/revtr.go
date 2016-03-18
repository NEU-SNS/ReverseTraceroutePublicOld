package revtr

import (
	"bytes"
	"encoding/json"
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
	"github.com/NEU-SNS/ReverseTraceroute/controller/client"
	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	vpservice "github.com/NEU-SNS/ReverseTraceroute/vpservice/client"
	"github.com/gorilla/websocket"
)

var plHost2IP map[string]string

type clusterMap struct {
	ipc map[string]cmItem
	mu  *sync.Mutex
	cs  ClusterSource
}

type cmItem struct {
	fetched time.Time
	val     string
}

func (cm clusterMap) fetchCluster(s string) string {
	ipint, _ := util.IPStringToInt32(s)
	cluster, err := cm.cs.GetClusterIDByIP(ipint)
	if err != nil {
		cm.ipc[s] = cmItem{
			fetched: time.Now(),
			val:     s,
		}
		return s
	}
	clusters := fmt.Sprintf("%d", cluster)
	cm.ipc[s] = cmItem{
		fetched: time.Now(),
		val:     clusters,
	}
	return clusters
}

func (cm clusterMap) Get(s string) string {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if cl, ok := cm.ipc[s]; ok {
		if time.Since(cl.fetched) < time.Hour*2 {
			return cl.val
		}
	}
	return cm.fetchCluster(s)
}

func newClusterMap(cs ClusterSource) clusterMap {
	i := make(map[string]cmItem)
	return clusterMap{
		ipc: i,
		mu:  &sync.Mutex{},
		cs:  cs,
	}
}

var ipToCluster clusterMap
var clusterToIps map[string][]string
var tsAdjsByCluster bool
var vps []*datamodel.VantagePoint
var rrVPsByCluster bool

// ReversePath is a reverse path
type ReversePath struct {
	Src, Dst string
	Path     *[]Segment
}

func (rp *ReversePath) len() int {
	return len(*rp.Path)
}

// NewReversePath creates a reverse path
func NewReversePath(src, dst string, path []Segment) *ReversePath {
	ret := ReversePath{
		Src: src,
		Dst: dst,
	}
	if len(path) == 0 {
		ret.Path = &[]Segment{NewDstRevSegment([]string{dst}, src, dst)}
	} else {
		ret.Path = &path
	}
	return &ret
}

// Clone clones a ReversePath
func (rp *ReversePath) Clone() *ReversePath {
	ret := ReversePath{
		Src:  rp.Src,
		Dst:  rp.Dst,
		Path: new([]Segment),
	}
	for _, seg := range *rp.Path {
		*ret.Path = append(*ret.Path, seg.Clone())
	}
	return &ret
}

// Hops gets the hops from each segment
func (rp *ReversePath) Hops() []string {
	var segs [][]string
	for _, p := range *rp.Path {
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
	for _, seg := range *rp.Path {
		length += seg.Length(false)
	}
	return length
}

// LastHop gets the last hop of the last segment
func (rp *ReversePath) LastHop() string {
	return rp.LastSeg().LastHop()
}

// LastSeg gets the last segment
func (rp *ReversePath) LastSeg() Segment {
	return (*rp.Path)[rp.len()-1]
}

// Pop pops a segment off of the path
func (rp *ReversePath) Pop() Segment {
	length := rp.len()
	last := (*rp.Path)[length-1]
	*rp.Path = (*rp.Path)[:length-1]
	return last
}

// Reaches returns weather or not the last segment reaches
func (rp *ReversePath) Reaches() bool {
	return rp.LastSeg().Reaches()
}

// SymmetricAssumptions returns the number of symmetric assumptions
func (rp *ReversePath) SymmetricAssumptions() int {
	var total int
	for _, seg := range *rp.Path {
		total += seg.SymmetricAssumptions()
	}
	return total
}

// Add adds a segment to the path
func (rp *ReversePath) Add(s Segment) {
	*rp.Path = append(*rp.Path, s)
}

func (rp *ReversePath) String() string {
	return fmt.Sprintf("RevPath_D%s_S%s_%v", rp.Dst, rp.Src, rp.Path)
}

const (
	// RateLimit is the ReverseTraceroute Rate Limit
	RateLimit int = 5
)

type wsConnection struct {
	c *websocket.Conn
}

func (ws wsConnection) Close() error {
	if ws.c == nil {
		return nil
	}
	return ws.c.Close()
}

func (ws wsConnection) Write(in []byte) error {
	if ws.c == nil {
		return nil
	}
	err := ws.c.SetWriteDeadline(time.Now().Add(time.Second * 10))
	if err != nil {
		return err
	}
	return ws.c.WriteMessage(websocket.TextMessage, in)

}

type wsConns []wsConnection

type multiError []error

func (m multiError) Error() string {
	var buf bytes.Buffer
	for _, e := range m {
		buf.WriteString(e.Error() + "\n")
	}
	return buf.String()
}

func (wc wsConns) Close() error {
	var err multiError
	for _, w := range wc {
		e2 := w.Close()
		if e2 != nil {
			err = append(err, e2)
		}
	}
	if len(err) == 0 {
		return nil
	}
	return err
}

func (wc wsConns) Write(in []byte) error {
	var err multiError
	for _, w := range wc {
		e2 := w.c.SetWriteDeadline(time.Now().Add(time.Second * 10))
		if e2 != nil {
			err = append(err, e2)
		}
		e2 = w.c.WriteMessage(websocket.TextMessage, []byte(in))
		if e2 != nil {
			err = append(err, e2)
		}
	}
	if len(err) == 0 {
		return nil
	}
	return err
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
	as                       AdjacencySource
	ws                       wsConns
	backoffEndhost           bool
	cl                       client.Client
	at                       at.Atlas
	print                    bool
	running                  bool
	mu                       sync.Mutex // protects running
	hostnameCache            map[string]string
	rttCache                 map[string]float32
	tokens                   []*datamodel.IntersectionResponse
	rrsSrcToDstToVPToRevHops map[string]map[string]map[string][]string
	trsSrcToDstToPath        map[string]map[string][]string
	tsSrcToProbeToVPToResult map[string]map[string]map[string][]string
	errorDetails             bytes.Buffer
	lastResponsive           string
}

// NewReverseTraceroute creates a new reverse traceroute
func NewReverseTraceroute(src, dst string, id, stale uint32, as AdjacencySource) *ReverseTraceroute {
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
	}
	return &ret
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

type wsMessage struct {
	HTML   string
	Status bool
	Error  string
}

func (rt *ReverseTraceroute) output() error {

	st := fmt.Sprintf("%s\n%s", rt.HTML(), rt.StopReason)
	var showError = !rt.Reaches()
	var errorText string
	if showError {
		errorText = strings.Replace(rt.errorDetails.String(), "\n", "<br>", -1)
	}
	res, err := json.Marshal(&wsMessage{
		HTML: strings.Replace(st, "\n", "<br>", -1),
		// Either we're done because we reached or we're done for some other reason
		// so signal the brower that we're gunna disconnect
		Status: rt.Reaches() || rt.StopReason != "",
		Error:  errorText,
	})
	if err != nil {
		return err
	}
	return rt.ws.Write(res)
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
func (rt *ReverseTraceroute) ToStorable() datamodel.ReverseTraceroute {
	var ret datamodel.ReverseTraceroute
	ret.Id = rt.ID
	ret.Src = rt.Src
	ret.Dst = rt.Dst
	ret.Runtime = rt.EndTime.Sub(rt.StartTime).Nanoseconds()
	ret.RrIssued = int32(rt.ProbeCount["rr"] + rt.ProbeCount["spoof-rr"])
	ret.TsIssued = int32(rt.ProbeCount["ts"] + rt.ProbeCount["spoof-ts"])
	ret.StopReason = rt.StopReason
	if rt.StopReason != "" {
		ret.Status = datamodel.RevtrStatus_COMPLETED
	} else {
		ret.Status = datamodel.RevtrStatus_RUNNING
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
				var h datamodel.RevtrHop
				h.Hop = hi
				h.Type = datamodel.RevtrHopType(ty)
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
	out.WriteString(fmt.Sprintf("%s (%s) back to ", rt.Hops()[0], rt.resolveHostname(rt.Hops()[0])))
	out.WriteString(rt.Src)
	out.WriteString(fmt.Sprintf(" (%s)", rt.resolveHostname(rt.Src)))
	out.WriteString("</caption>")
	out.WriteString(`<tbody>`)
	first := true
	var i int
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

type byCount []datamodel.AdjacencyToDest

func (b byCount) Len() int           { return len(b) }
func (b byCount) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byCount) Less(i, j int) bool { return b[i].Cnt < b[j].Cnt }

type aByCount []datamodel.Adjacency

func (b aByCount) Len() int           { return len(b) }
func (b aByCount) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b aByCount) Less(i, j int) bool { return b[i].Cnt < b[j].Cnt }

func getAdjacenciesForIPToSrc(ip string, src string, as AdjacencySource, settings *adjSettings) ([]string, error) {
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

func chooseOneSpooferPerSite() map[string]*datamodel.VantagePoint {
	ret := make(map[string]*datamodel.VantagePoint)
	for _, vp := range vps {
		if vp.CanSpoof {
			ret[vp.Site] = vp
		}
	}
	return ret
}

func getRRSpoofers() map[string]*datamodel.VantagePoint {
	ret := make(map[string]*datamodel.VantagePoint)
	for _, vp := range vps {
		if vp.CanSpoof && vp.RecordRoute {
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
		if val.Timestamp && val.CanSpoof {
			ips, _ := util.Int32ToIPString(val.Ip)
			spoofers = append(spoofers, ips)
		}
	}
	return spoofers
}

// InitializeRRVPs initializes the rr vps for a cls
func (rt *ReverseTraceroute) InitializeRRVPs(cls string) error {
	rt.debug("Initializing RR VPs individually for spoofers for ", cls)
	rt.RRHop2RateLimit[cls] = RateLimit
	siteToSpoofer := getRRSpoofers()
	var sitesForTarget []*datamodel.VantagePoint
	sitesForTarget = nil
	spoofersForTarget := []string{"non_spoofed"}
	var tempSpoofers []string
	if sitesForTarget == nil {
		for _, val := range siteToSpoofer {
			ipsrc, _ := util.Int32ToIPString(val.Ip)
			if ipsrc == rt.Src {
				continue
			}
			tempSpoofers = append(tempSpoofers, ipsrc)
		}
		random := rand.Perm(len(tempSpoofers))
		for _, r := range random {
			spoofersForTarget = append(spoofersForTarget, tempSpoofers[r])
		}
	} else {
		// TODO
		// This is the case for using smarter results for vp selection
		// currently we don't have this so nothing is gunna happen
	}
	if len(spoofersForTarget) > 10 {
		rt.RRHop2VPSLeft[cls] = spoofersForTarget[:10]
	} else {
		rt.RRHop2VPSLeft[cls] = spoofersForTarget
	}
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
func CreateReverseTraceroute(revtr datamodel.RevtrMeasurement, backoffEndhost, print bool, cl client.Client, at at.Atlas, vpserv vpservice.VPSource, as AdjacencySource, cs ClusterSource) *ReverseTraceroute {
	once.Do(func() {
		initialize(vpserv, cs)
	})
	rt := NewReverseTraceroute(revtr.Src, revtr.Dst, revtr.Id, revtr.Staleness, as)
	rt.backoffEndhost = backoffEndhost
	rt.print = print
	rt.cl = cl
	rt.at = at
	return rt
}

func (rt *ReverseTraceroute) isRunning() bool {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.running
}

func (rt *ReverseTraceroute) run() error {
	rt.mu.Lock()
	rt.running = true
	rt.mu.Unlock()
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
			continue
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
			continue
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
			continue
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

// ReverseTracerouteReq is a revtr req
type ReverseTracerouteReq struct {
	Src, Dst  uint32
	Staleness uint32
}

type stringInt int

func (si stringInt) String() string {
	return fmt.Sprintf("%d", int(si))
}

func initialize(cl vpservice.VPSource, cs ClusterSource) {
	clusterToIps = make(map[string][]string)
	vpr, err := cl.GetVPs()
	if err != nil {
		panic(err)
	}
	vps = vpr.GetVps()
	ipToCluster = newClusterMap(cs)
}

// AdjacencySource is the interface for something that provides adjacnecies
type AdjacencySource interface {
	GetAdjacenciesByIP1(uint32) ([]datamodel.Adjacency, error)
	GetAdjacenciesByIP2(uint32) ([]datamodel.Adjacency, error)
	GetAdjacencyToDestByAddrAndDest24(uint32, uint32) ([]datamodel.AdjacencyToDest, error)
}

// ClusterSource is the interface for something that provides cluster data
type ClusterSource interface {
	GetClusterIDByIP(uint32) (int, error)
	GetIPsForClusterID(int) ([]uint32, error)
}

var once sync.Once

// RunReverseTraceroute runs a reverse traceroute
func RunReverseTraceroute(revtr datamodel.RevtrMeasurement, cl client.Client, at at.Atlas, vpserv vpservice.VPSource, as AdjacencySource, cs ClusterSource) (*ReverseTraceroute, error) {
	once.Do(func() {
		initialize(vpserv, cs)
	})
	rt := NewReverseTraceroute(revtr.Src, revtr.Dst, revtr.Id, revtr.Staleness, as)
	rt.backoffEndhost = revtr.BackoffEndhost
	rt.cl = cl
	rt.at = at
	return rt, rt.run()
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
		if !isInPrivatePrefix(net.ParseIP(target)) {
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
				if isInPrivatePrefix(net.ParseIP(target)) {
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
		if len(pr) > 0 {
			ssrc, _ := util.Int32ToIPString(p.Src)
			sdst, _ := util.Int32ToIPString(p.Dst)
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

			}
		}
	}
	return nil
}

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
				rt.debug("Fixing up hop: ", h)
				hss, _ := util.Int32ToIPString(h.Ip)
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
	if isInPrivatePrefix(net.ParseIP(rt.LastHop())) {
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

		if _, ok := tsSrcToHopToSendSpoofed[f]; !ok {
			tsSrcToHopToSendSpoofed[f] = make(map[string]bool)
		}
	}
	initTsSrcToHopToResponseive(rt.Sr)
	if tsSrcToHopToResponsive[rt.Src][rt.LastHop()] != 0 {
		rt.debug("No VPS found for ", rt.Src, " last hop: ", rt.LastHop())
		return ErrNoVPs
	}
	adjacents := rt.GetTSAdjacents(ipToCluster.Get(rt.LastHop()))
	rt.debug("Adjacents: ", adjacents)
	if len(adjacents) == 0 {
		rt.debug("No adjacents found")
		return ErrNoAdj
	}
	if tsDstToStampsZero[rt.LastHop()] {
		rt.debug("tsDstToStampsZero wtf")
		for _, adj := range adjacents {
			dstsDoNotStamp = append(dstsDoNotStamp, []string{rt.Src, rt.LastHop(), adj})
		}
	} else if !tsSrcToHopToSendSpoofed[rt.Src][rt.LastHop()] {
		rt.debug("Adding Spoofed TS to send")
		for _, adj := range adjacents {
			tsToIssueSrcToProbe[rt.Src] = append(tsToIssueSrcToProbe[rt.Src], []string{rt.LastHop(), rt.LastHop(), adj, adj, dummyIP})
		}
	} else {
		rt.debug("TS Non of the above")
		spfs := getTimestampSpoofers(rt.Src, rt.LastHop())
		for _, adj := range adjacents {
			for _, spf := range spfs {
				checkMapMagic(rt.Src, spf)
				receiverToSpooferToProbe[rt.Src][spf] = append(receiverToSpooferToProbe[rt.Src][spf], []string{rt.LastHop(), rt.LastHop(), adj, adj, dummyIP})
			}
		}
		// if we haven't already decided whether it is responsive,
		// we'll set it to false, then change to true if we get one
		initTsSrcToHopToResponseive(rt.Src)
		if _, ok := tsSrcToHopToResponsive[rt.Src][rt.LastHop()]; !ok {
			tsSrcToHopToResponsive[rt.Src][rt.LastHop()] = 1
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
			tsSrcToHopToSendSpoofed[src][dsts] = false
			segClass = "TSAdjRevSegment"
		}
		initTsSrcToHopToResponseive(src)
		tsSrcToHopToResponsive[src][dsts] = 1
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
				tsDstToStampsZero[dsts] = true
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
				if _, ok := tsSrcToHopToSendSpoofed[src][probe[0]]; ok {
					continue
				}
				// set it to true, then change it to false if we get a response
				tsSrcToHopToSendSpoofed[src][probe[0]] = true
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
				if tsSrcToHopToSendSpoofed[src][probe[0]] {
					mySpoofers := getTimestampSpoofers(src, probe[0])
					for _, sp := range mySpoofers {
						rt.debug("Adding spoofed TS probe to send")
						checkMapMagic(src, sp)
						receiverToSpooferToProbe[src][sp] = append(receiverToSpooferToProbe[src][sp], probe)
					}
					// if we haven't already decided whether it is responsive
					// we'll set it to false, then change to true if we get one
					if _, ok := tsSrcToHopToResponsive[src][probe[0]]; !ok {
						initTsSrcToHopToResponseive(src)
						tsSrcToHopToResponsive[src][probe[0]] = 1
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
				tsSrcToHopToSendSpoofed[src][dsts] = false
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
		spoofers := getTimestampSpoofers(probe.src, probe.dst)
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

// whether this destination is repsonsive but with ts=0
var tsDstToStampsZero = make(map[string]bool)

// whether this particular src should use spoofed ts to that hop
var tsSrcToHopToSendSpoofed = make(map[string]map[string]bool)

// whether this hop is thought to be responsive at all to this src
// Since I can't intialize to true, I'm going to use an int and say 0 is true
// anythign else will be false
var tsSrcToHopToResponsive = make(map[string]map[string]int)

func initTsSrcToHopToResponseive(s string) {
	if _, ok := tsSrcToHopToResponsive[s]; ok {
		return
	}
	tsSrcToHopToResponsive[s] = make(map[string]int)
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
			if isInPrivatePrefix(net.ParseIP(probe[0])) {
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
				if isInPrivatePrefix(net.ParseIP(probe[0])) {
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
