package atlas

import (
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	cclient "github.com/NEU-SNS/ReverseTraceroute/controller/client"
	"github.com/NEU-SNS/ReverseTraceroute/dataaccess"
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice/client"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"
)

var (
	procCollector = prometheus.NewProcessCollectorPIDFn(func() (int, error) {
		return os.Getpid(), nil
	}, getName())
)

var id = rand.Uint32()

func getName() string {
	name, err := os.Hostname()
	if err != nil {
		return fmt.Sprintf("atlas_%d", id)
	}
	return fmt.Sprintf("atlas_%s", strings.Replace(name, ".", "_", -1))
}

func init() {
	prometheus.MustRegister(procCollector)
}

// Atlas is the atlas
type Atlas struct {
	da     *dataaccess.DataAccess
	donec  chan struct{}
	rootCA string
	curr   runningTraces
	ccon   cclient.Client
	vpcon  client.VPSource
	tc     *tokenCache
}

// NewAtlasService creates a new Atlas
func NewAtlasService(da *dataaccess.DataAccess, rootCA string) *Atlas {
	log.Debug("Creating New Atlas Service")
	ret := &Atlas{
		da:     da,
		rootCA: rootCA,
		curr:   newRunningTraces(),
		tc:     newTokenCache(),
	}
	ret.startHTTP()
	return ret
}

func startHTTP(addr string) {
	for {
		log.Error(http.ListenAndServe(addr, nil))
	}
}

func (a *Atlas) startHTTP() {
	http.Handle("/metrics", prometheus.Handler())
	go startHTTP(":8080")
}

// GetPathsWithToken satisfies the server interface
func (a *Atlas) GetPathsWithToken(ctx context.Context, in *dm.TokenRequest) ([]*dm.TokenResponse, error) {
	out := make(chan *dm.TokenResponse, 1)
	go func() {
		log.Debug("Looking for intersection from token: ", in)
		req := a.tc.Get(in.Token)
		if req == nil {
			select {
			case out <- &dm.TokenResponse{
				Token: in.Token,
				Type:  dm.IResponseType_ERROR,
				Error: fmt.Sprintf("No request found matching: %d", in.Token),
			}:
			case <-ctx.Done():
				log.Error(ctx.Err())
			}
			close(out)
			return
		}
		a.tc.Remove(in.Token)
		pair := []dm.SrcDst{
			dm.SrcDst{
				Addr:         req.Address,
				Dst:          req.Dest,
				Src:          req.Src,
				Stale:        time.Duration(req.Staleness) * time.Minute,
				IgnoreSource: req.IgnoreSource,
				Alias:        req.UseAliases,
			},
		}
		log.Debug("Looking for intesection for: ", req)
		path, err := a.da.FindIntersectingTraceroute(pair)
		log.Debug("FindIntersectingTraceroute resp: ", path)
		if err != nil {
			log.Debug("Found no intersection")
			if err != dataaccess.ErrNoIntFound {
				log.Error(err)
				select {
				case out <- &dm.TokenResponse{
					Type:  dm.IResponseType_ERROR,
					Error: err.Error(),
				}:
				case <-ctx.Done():
					log.Error(ctx.Err())
				}
			} else {
				intr := &dm.TokenResponse{
					Token: in.Token,
					Type:  dm.IResponseType_NONE_FOUND,
				}
				select {
				case out <- intr:
				case <-ctx.Done():
					log.Error(ctx.Err())
				}
			}
			close(out)
			return
		}
		if len(path) == 0 {
			log.Debug("Found no path")
			intr := &dm.TokenResponse{
				Token: in.Token,
				Type:  dm.IResponseType_NONE_FOUND,
			}
			select {
			case out <- intr:
			case <-ctx.Done():
				log.Error(ctx.Err())
			}
			close(out)
			return
		}
		for _, resp := range path {
			log.Debug("Got path: ", path, " for token ", in.Token)
			intr := &dm.TokenResponse{
				Token: in.Token,
				Type:  dm.IResponseType_PATH,
				Path:  resp,
			}
			select {
			case out <- intr:
			case <-ctx.Done():
				log.Error(ctx.Err())
				break
			}
		}
		close(out)
		return
	}()
	var results []*dm.TokenResponse
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case r, ok := <-out:
			log.Debug("Token intersection get result: ", r)
			if !ok {
				log.Debug("Out was closed")
				return results, nil
			}
			results = append(results, r)
		}
	}
}

// GetIntersectingPath satisfies the server interface
func (a *Atlas) GetIntersectingPath(ctx context.Context, ir *dm.IntersectionRequest) ([]*dm.IntersectionResponse, error) {
	in := make(chan *dm.IntersectionResponse, 1)
	go func() {
		log.Debug("Looing for intersect for ", ir)
		req := []dm.SrcDst{
			dm.SrcDst{
				Addr:         ir.Address,
				Dst:          ir.Dest,
				Src:          ir.Src,
				Stale:        time.Duration(ir.Staleness) * time.Minute,
				Alias:        ir.UseAliases,
				IgnoreSource: ir.IgnoreSource,
			},
		}
		if ir.Staleness == 0 {
			ir.Staleness = 60
		}
		res, err := a.da.FindIntersectingTraceroute(req)
		log.Debug("FindIntersectingTraceroute resp ", res)
		if err != nil {
			if err != dataaccess.ErrNoIntFound {
				log.Error(err)
				iresp := &dm.IntersectionResponse{
					Type:  dm.IResponseType_ERROR,
					Error: err.Error(),
				}
				log.Error(err)
				select {
				case in <- iresp:
				case <-ctx.Done():
					close(in)
					return
				}
			} else {
				iresp := &dm.IntersectionResponse{
					Type:  dm.IResponseType_TOKEN,
					Token: a.tc.Add(ir),
				}
				select {
				case in <- iresp:
				case <-ctx.Done():
					close(in)
					return
				}
				go a.fillAtlas(ir.Address, ir.Dest, ir.Staleness)
			}

			close(in)
			return
		}
		if len(res) == 0 {
			intr := &dm.IntersectionResponse{
				Type: dm.IResponseType_NONE_FOUND,
			}
			select {
			case in <- intr:
			case <-ctx.Done():
			}
			close(in)
			return
		}
		for _, resp := range res {
			intr := &dm.IntersectionResponse{
				Type: dm.IResponseType_PATH,
				Path: resp,
			}
			select {
			case in <- intr:
			case <-ctx.Done():
				break
			}
		}
		close(in)
		return
	}()
	var ret []*dm.IntersectionResponse
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case ir, ok := <-in:
			log.Debug("Got: ", ir, " ", ok)
			if !ok {
				return ret, nil
			}
			ret = append(ret, ir)
		}
	}
}

func (a *Atlas) connect() bool {
	if a.ccon == nil {
		_, srvs, err := net.LookupSRV("controller", "tcp", "revtr.ccs.neu.edu")
		if err != nil {
			log.Error(err)
			return false
		}
		connstr := fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port)
		creds, err := credentials.NewClientTLSFromFile(a.rootCA, srvs[0].Target)
		if err != nil {
			log.Error(err)
			return false
		}
		cc, err := grpc.Dial(connstr, grpc.WithTransportCredentials(creds))
		if err != nil {
			log.Error(err)
			return false
		}
		a.ccon = cclient.New(context.Background(), cc)
	}
	if a.vpcon == nil {
		_, srvs, err := net.LookupSRV("vpservice", "tcp", "revtr.ccs.neu.edu")
		if err != nil {
			log.Error(err)
			return false
		}
		connstr := fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port)
		creds, err := credentials.NewClientTLSFromFile(a.rootCA, srvs[0].Target)
		if err != nil {
			log.Error(err)
			return false
		}
		cc, err := grpc.Dial(connstr, grpc.WithTransportCredentials(creds))
		if err != nil {
			log.Error(err)
			return false
		}
		a.vpcon = client.New(context.Background(), cc)
	}
	return true
}

func (a *Atlas) fillAtlas(hop, dest uint32, stale int64) {
	if !a.connect() {
		return
	}
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
	st, err := a.ccon.Traceroute(ctx, &dm.TracerouteArg{Traceroutes: traces})
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
			err = a.da.StoreAtlasTraceroute(tr)
			if err != nil {
				log.Error(err)
			}
		}(t)
		finished = append(finished, t.Src)
	}
	a.curr.Remove(dest, finished)
}

func (a *Atlas) getSrcs(hop, dest uint32, stale int64) []uint32 {
	vps, err := a.vpcon.GetVPs()
	if err != nil {
		return nil
	}
	oldsrcs, err := a.da.GetAtlasSources(dest, time.Minute*time.Duration(stale))
	os := make(map[uint32]bool)
	for _, o := range oldsrcs {
		os[o] = true
	}
	sites := make(map[string]*dm.VantagePoint)
	var srcIsVP *dm.VantagePoint
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
	cache map[uint32]*dm.IntersectionRequest
	// Should only be accessed atomicaly
	nextID uint32
}

func (tc *tokenCache) Add(ir *dm.IntersectionRequest) uint32 {
	new := atomic.AddUint32(&tc.nextID, 1)
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.cache[new] = ir
	return new
}

func (tc *tokenCache) Get(id uint32) *dm.IntersectionRequest {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	return tc.cache[id]
}

func (tc *tokenCache) Remove(id uint32) error {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	if _, ok := tc.cache[id]; ok {
		delete(tc.cache, id)
		return nil
	}
	return fmt.Errorf("No token registerd for id: %d", id)
}

func newTokenCache() *tokenCache {
	tc := &tokenCache{
		cache: make(map[uint32]*dm.IntersectionRequest),
	}
	return tc
}
