package server

import (
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/atlas/pb"
	"github.com/NEU-SNS/ReverseTraceroute/atlas/repo"
	"github.com/NEU-SNS/ReverseTraceroute/atlas/types"
	cclient "github.com/NEU-SNS/ReverseTraceroute/controller/client"
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice/client"
	vppb "github.com/NEU-SNS/ReverseTraceroute/vpservice/pb"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"
)

var (
	nameSpace     = "atlas"
	procCollector = prometheus.NewProcessCollectorPIDFn(func() (int, error) {
		return os.Getpid(), nil
	}, nameSpace)
)

func init() {
	prometheus.MustRegister(procCollector)
}

// AtlasServer is the interface for the atlas
type AtlasServer interface {
	GetIntersectingPath(*pb.IntersectionRequest) (*pb.IntersectionResponse, error)
	GetPathsWithToken(*pb.TokenRequest) (*pb.TokenResponse, error)
}

type server struct {
	donec chan struct{}
	curr  runningTraces
	tc    *tokenCache
	opts  serverOptions
}

type serverOptions struct {
	cl  cclient.Client
	vps client.VPSource
	trs types.TRStore
}

// Option sets an option to configure the server
type Option func(*serverOptions)

// WithClient configures the server with client c
func WithClient(c cclient.Client) Option {
	return func(opts *serverOptions) {
		opts.cl = c
	}
}

// WithVPS configures the server with the given VPSource
func WithVPS(vps client.VPSource) Option {
	return func(opts *serverOptions) {
		opts.vps = vps
	}
}

// WithTRS configures the server with the given TRStore
func WithTRS(trs types.TRStore) Option {
	return func(opts *serverOptions) {
		opts.trs = trs
	}
}

// NewServer creates a server
func NewServer(opts ...Option) AtlasServer {
	atlas := &server{
		curr: newRunningTraces(),
		tc:   newTokenCache(),
	}
	for _, opt := range opts {
		opt(&atlas.opts)
	}
	return atlas
}

// GetPathsWithToken satisfies the server interface
func (a *server) GetPathsWithToken(tr *pb.TokenRequest) (*pb.TokenResponse, error) {
	log.Debug("Looking for intersection from token: ", tr)
	req := a.tc.Get(tr.Token)
	if req == nil {
		return &pb.TokenResponse{
			Token: tr.Token,
			Type:  pb.IResponseType_ERROR,
			Error: fmt.Sprintf("No request found matching: %d", tr.Token),
		}, nil
	}
	a.tc.Remove(tr.Token)

	ir := types.IntersectionQuery{
		Addr:         req.Address,
		Dst:          req.Dest,
		Src:          req.Src,
		Stale:        time.Duration(req.Staleness) * time.Minute,
		IgnoreSource: req.IgnoreSource,
		Alias:        req.UseAliases,
	}
	log.Debug("Looking for intesection for: ", req)
	path, err := a.opts.trs.FindIntersectingTraceroute(ir)
	log.Debug("FindIntersectingTraceroute resp: ", path)
	if err != nil {
		log.Debug("Found no intersection")
		if err != repo.ErrNoIntFound {
			log.Error(err)
			return nil, err
		}
		return &pb.TokenResponse{
			Token: tr.Token,
			Type:  pb.IResponseType_NONE_FOUND,
		}, nil
	}
	log.Debug("Got path: ", path, " for token ", tr.Token)
	intr := &pb.TokenResponse{
		Token: tr.Token,
		Type:  pb.IResponseType_PATH,
		Path:  path,
	}
	return intr, nil
}

// GetIntersectingPath satisfies the server interface
func (a *server) GetIntersectingPath(ir *pb.IntersectionRequest) (*pb.IntersectionResponse, error) {
	log.Debug("Looing for intersection for ", ir)
	if ir.Staleness == 0 {
		ir.Staleness = 60
	}
	iq := types.IntersectionQuery{
		Addr:         ir.Address,
		Dst:          ir.Dest,
		Src:          ir.Src,
		Stale:        time.Duration(ir.Staleness) * time.Minute,
		Alias:        ir.UseAliases,
		IgnoreSource: ir.IgnoreSource,
	}
	res, err := a.opts.trs.FindIntersectingTraceroute(iq)
	log.Debug("FindIntersectingTraceroute resp ", res)
	if err != nil {
		if err != repo.ErrNoIntFound {
			log.Error(err)
			return nil, err
		}
		iresp := &pb.IntersectionResponse{
			Type:  pb.IResponseType_TOKEN,
			Token: a.tc.Add(ir),
		}
		go a.fillAtlas(ir.Address, ir.Dest, ir.Staleness)
		return iresp, nil
	}
	intr := &pb.IntersectionResponse{
		Type: pb.IResponseType_PATH,
		Path: res,
	}
	return intr, nil
}

func (a *server) fillAtlas(hop, dest uint32, stale int64) {
	srcs := a.getSrcs(hop, dest, stale)
	var traces []*dm.TracerouteMeasurement
	for _, src := range srcs {
		curr := &dm.TracerouteMeasurement{
			Src:        src,
			Dst:        dest,
			Timeout:    20,
			Wait:       "2",
			Attempts:   "1",
			LoopAction: "1",
			Loops:      "3",
			CheckCache: true,
			CheckDb:    true,
			Staleness:  stale,
		}
		traces = append(traces, curr)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	st, err := a.opts.cl.Traceroute(ctx, &dm.TracerouteArg{Traceroutes: traces})
	if err != nil {
		log.Error(err)
		a.curr.Remove(dest, srcs)
		return
	}
	var finished []uint32
	for {
		t, err := st.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			break
		}
		go func(tr *dm.Traceroute) {
			hops := tr.GetHops()
			if len(hops) == 0 {
				return
			}
			if hops[len(hops)-1].Addr != tr.Dst {
				log.Error("Traceroute did not reach destination")
				return
			}
			err = a.opts.trs.StoreAtlasTraceroute(tr)
			if err != nil {
				log.Error(err)
			}
		}(t)
		finished = append(finished, t.Src)
	}
	a.curr.Remove(dest, finished)
}

func (a *server) getSrcs(hop, dest uint32, stale int64) []uint32 {
	vps, err := a.opts.vps.GetVPs()
	if err != nil {
		return nil
	}
	oldsrcs, err := a.opts.trs.GetAtlasSources(dest, time.Minute*time.Duration(stale))
	os := make(map[uint32]bool)
	for _, o := range oldsrcs {
		os[o] = true
	}
	sites := make(map[string]*vppb.VantagePoint)
	var srcIsVP *vppb.VantagePoint
	for _, vp := range vps.GetVps() {
		if os[vp.Ip] {
			// if the src has been used in interval [now, stale], skip it
			continue
		}
		if vp.Ip == hop {
			srcIsVP = vp
		}
		sites[vp.Site] = vp
	}
	//overwrite site to use the src
	if srcIsVP != nil {
		sites[srcIsVP.Site] = srcIsVP
	}
	var srcs []uint32
	for _, vp := range sites {
		srcs = append(srcs, vp.Ip)
	}
	return a.curr.TryAdd(dest, srcs)
}

type runningTraces struct {
	mu        *sync.Mutex
	dstToSrcs map[uint32][]uint32
}

func newRunningTraces() runningTraces {
	return runningTraces{
		mu:        &sync.Mutex{},
		dstToSrcs: make(map[uint32][]uint32),
	}
}

func (rt runningTraces) Check(ip uint32) ([]uint32, bool) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	srcs, ok := rt.dstToSrcs[ip]
	return srcs, ok
}

func (rt runningTraces) Remove(ip uint32, done []uint32) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if running, ok := rt.dstToSrcs[ip]; ok {
		checked := make(map[uint32]bool)
		for _, r := range running {
			checked[r] = true
		}
		for _, d := range done {
			checked[d] = false
		}
		var new []uint32
		for k, v := range checked {
			if v {
				new = append(new, k)
			}
		}
		if len(new) > 0 {
			rt.dstToSrcs[ip] = new
		} else {
			delete(rt.dstToSrcs, ip)
		}
	}
}

// UInt32Slice is for sorting uint32s
type UInt32Slice []uint32

func (u UInt32Slice) Len() int           { return len(u) }
func (u UInt32Slice) Less(i, j int) bool { return u[i] < u[j] }
func (u UInt32Slice) Swap(i, j int)      { u[i], u[j] = u[j], u[i] }

func (rt runningTraces) TryAdd(ip uint32, dsts []uint32) []uint32 {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	var merged []uint32
	var added []uint32
	if old, ok := rt.dstToSrcs[ip]; ok {
		sort.Sort(UInt32Slice(old))
		sort.Sort(UInt32Slice(dsts))
		var i, j int
		for i < len(old) && j < len(dsts) {
			switch {
			case old[i] < dsts[j]:
				merged = append(merged, old[i])
				i++
			case old[i] > dsts[j]:
				merged = append(merged, dsts[j])
				added = append(added, dsts[j])
				j++
			default:
				merged = append(merged, old[i])
				i++
				j++
			}
		}
		for i < len(old) {
			merged = append(merged, old[i])
			i++
		}
		for j < len(dsts) {
			merged = append(merged, dsts[j])
			added = append(added, dsts[j])
			j++
		}
		rt.dstToSrcs[ip] = added
		return added
	}
	rt.dstToSrcs[ip] = dsts
	return dsts
}

type tokenCache struct {
	mu    sync.Mutex
	cache map[uint32]*pb.IntersectionRequest
	// Should only be accessed atomicaly
	nextID uint32
}

func (tc *tokenCache) Add(ir *pb.IntersectionRequest) uint32 {
	new := atomic.AddUint32(&tc.nextID, 1)
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.cache[new] = ir
	return new
}

func (tc *tokenCache) Get(id uint32) *pb.IntersectionRequest {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	return tc.cache[id]
}

type cacheError struct {
	id uint32
}

func (ce cacheError) Error() string {
	return fmt.Sprintf("No token registered for id: %d", ce.id)
}

func (tc *tokenCache) Remove(id uint32) error {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	if _, ok := tc.cache[id]; ok {
		delete(tc.cache, id)
		return nil
	}
	return cacheError{id: id}
}

func newTokenCache() *tokenCache {
	tc := &tokenCache{
		cache: make(map[uint32]*pb.IntersectionRequest),
	}
	return tc
}
