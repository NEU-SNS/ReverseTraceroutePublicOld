package revtr

import (
	"fmt"
	"io"
	"sync"

	alc "github.com/NEU-SNS/ReverseTraceroute/atlas/client"
	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
)

type aliasMap struct {
	rw     sync.RWMutex
	ipToId map[uint32]int
	idToIp map[int][]uint32
}

func (am *aliasMap) aliasEquals(l, r uint32) bool {
	// Start with simple case, l and r are equal
	if l == r {
		return true
	}
	am.rw.RLock()
	defer am.rw.RUnlock()
	idl := am.ipToId[l]
	if idl == 0 {
		return false
	}
	idr := am.ipToId[r]
	if idr == 0 {
		return false
	}
	return idr == idl
}

type Alias interface {
	AliasEquals(l, r uint32) bool
}

// Tracerouter is something that can traceroute
type Tracerouter interface {
	Traceroute(src, dst uint32) (*TracerouteStream, error)
}

// TracerouteStream is a stream of traceroutes
type TracerouteStream struct{}

// Recv receives a tracerotue from the stream
func (ts *TracerouteStream) Recv() (*datamodel.Traceroute, error) {
	return nil, nil
}

// Hop is a hop
type Hop interface {
	Addr() uint32
}

// SymmetricHop represents a symmetric hop
type SymmetricHop struct {
	used int
	path []*datamodel.TracerouteHop
}

// Addr is to satisfy the Hop interface
func (sh SymmetricHop) Addr() uint32 {
	return sh.path[sh.used].Addr
}

// ITHop is a hop found from
type ITHop struct {
	index int
	path  *datamodel.Path
	alias bool
}

// Addr satisfies the Hop interface
func (th ITHop) Addr() uint32 {
	hops := th.path.GetHops()
	return hops[th.index].Ip
}

// ReversePath represents a path from a destination to a source
type ReversePath struct {
	src, dst       uint32
	done           bool
	backoffEndhost bool
	hops           []Hop
	alias          Alias
}

// Addr satisfies the hop interface
func (rp *ReversePath) Addr() uint32 {
	return rp.src
}

func (rp *ReversePath) lastHop() Hop {
	l := len(rp.hops)
	if l == 0 {
		return rp
	}
	return rp.hops[l-1]
}

func (rp *ReversePath) checkDone() {
	lh := rp.lastHop()
	rp.done = rp.src == lh.Addr() && len(rp.hops) != 0
}

// Done returns true if the ReversePath reaches the source
func (rp *ReversePath) Done() bool {
	return rp.done
}

// ErrUnresponsiveTrace is the error when a traceroute fails
type ErrUnresponsiveTrace struct {
	src, dst uint32
}

func (e *ErrUnresponsiveTrace) Error() string {
	return fmt.Sprintf("Unresponsive trace to %d from %d", e.src, e.dst)
}

// AddSymmetricHop adds an assumed symmteric hop to the path
// If a traceroute is needed to be run, the src should be set to the src of
// the reverse path before being added.
func (rp *ReversePath) AddSymmetricHop(tr Tracerouter) error {
	if rp.Done() {
		return fmt.Errorf("Trying to add symmetric hop to a completed path")
	}
	last := rp.lastHop()
	if lh, ok := last.(SymmetricHop); ok {
		if lh.used == len(lh.path)-1 {
			return fmt.Errorf("Trying to add symmetric hop when last hop reaches")
		}
		newsym := SymmetricHop{
			used: lh.used + 1,
			path: lh.path,
		}
		rp.hops = append(rp.hops, newsym)
		rp.checkDone()
		return nil
	}
	// Run a traceroute to get a symmetric hop
	st, err := tr.Traceroute(rp.lastHop().Addr(), rp.dst)
	if err != nil {
		return fmt.Errorf("AddSymmetricHop failed: %v", err)
	}
	for {
		tr, err := st.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("AddSymmetricHop failed: %v", err)
		}
		hops := tr.GetHops()
		// I didn't get any hops from the traceroute.
		// Or I didn't complete
		// In this case I'm an error
		if hops == nil || len(hops) == 0 || tr.StopReason != "COMPLETED" {
			return &ErrUnresponsiveTrace{src: rp.lastHop().Addr(), dst: rp.dst}
		}
		// len(hops) - 2 since the last should be the target and I wanna go one back
		rp.hops = append(rp.hops, &SymmetricHop{used: len(hops) - 2, path: hops})
	}
	rp.checkDone()
	return nil
}

// p is the Path that intersects the hop asked for, is is the index into
// the existing reversepath that intersects
func (rp *ReversePath) addIntersectingPath(p *datamodel.Path, is uint32) error {
	hops := p.GetHops()
	ishop := rp.hops[is]
	// I'm going to start at the last hop and work my way towards the source
	pos := len(hops) - 1
	for i := pos; i >= 0; i-- {
		if !rp.alias.AliasEquals(ishop.Addr(), hops[i].Ip) {
			// We're not the matching hop, continue towards the source of the trace
			continue
		}

	}
	return nil
}

// FindIntersection attempts to use an atlas to find an intersecting traceroute
// for any of the hops in the current reverse path
func (rp *ReversePath) FindIntersection(in alc.Atlas) error {
	cl, err := in.GetIntersectingPath()
	if err != nil {
		return err
	}
	defer cl.CloseSend()
	hn := len(rp.hops)
	for i := hn - 1; i >= 0; i-- {
		hop := rp.hops[i].Addr()
		req := &datamodel.IntersectionRequest{
			Address:    hop,
			Dest:       rp.src,
			UseAliases: true,
		}
		err := cl.Send(req)
		if err != nil {
			return err
		}
		resp, err := cl.Recv()
		if err != nil {
			return err
		}
		//TODO eventually the TOKEN aspect needs to be handled
		if resp.Type != datamodel.IResponseType_PATH {
			continue
		}
		// I found an intersection to the dest, add it to the path and we're
		// done at this point
	}
	return nil
}
