package atlas

import (
	"fmt"
	"io"
	"net"
	"sort"
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
	"golang.org/x/net/context"
)

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
	return ret
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
				Src: req.Address,
				Dst: req.Dest,
			},
		}
		st := time.Duration(req.Staleness)
		log.Debug("Looking for intesection for: ", req)
		path, err := a.da.FindIntersectingTraceroute(pair, req.UseAliases, st*time.Minute)
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
				Src: ir.Address,
				Dst: ir.Dest,
			},
		}
		if ir.Staleness == 0 {
			ir.Staleness = 60
		}
		st := time.Duration(ir.Staleness)
		res, err := a.da.FindIntersectingTraceroute(req, ir.UseAliases, st*time.Minute)
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
				go a.fillAtlas(ir.Dest)
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

func (a *Atlas) fillAtlas(dest uint32) {
	if !a.connect() {
		return
	}
	srcs := a.getSrcs(dest)
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
		go func() {
			err = a.da.StoreAtlasTraceroute(t)
			if err != nil {
				log.Error(err)
			}
		}()
		finished = append(finished, t.Src)
	}
	a.curr.Remove(dest, finished)
}

func (a *Atlas) getSrcs(dest uint32) []uint32 {
	vps, err := a.vpcon.GetVPs()
	if err != nil {
		return nil
	}
	sites := make(map[string]*dm.VantagePoint)
	var site string
	for _, vp := range vps.GetVps() {
		if vp.Ip == dest {
			site = vp.Site
		}
		sites[vp.Site] = vp
	}
	delete(sites, site)
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
