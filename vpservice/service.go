package vpservice

import (
	"io"
	"math/rand"
	"sync"
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/controller/pb"
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type vpMap map[uint32]*dm.VantagePoint

func (v vpMap) GetAll() []*dm.VantagePoint {
	out := make([]*dm.VantagePoint, 0, len(v))
	for _, val := range v {
		out = append(out, val)
	}
	return out
}

func (v vpMap) Merge(n vpMap) {
	for key := range v {
		// If the old one is not in the new set, delete it
		// from the old set
		if _, ok := n[key]; !ok {
			delete(v, key)
		}
	}
	for key, vp := range n {
		// If the one in the new set is not in the old one
		// add it to the old set
		if _, ok := v[key]; !ok {
			v[key] = vp
		}
	}
}

func (v vpMap) Update(n vpMap) {
	for key, val := range n {
		if vp, ok := v[key]; ok {
			vp.CanSpoof = val.CanSpoof
			vp.Timestamp = val.Timestamp
			vp.RecordRoute = val.RecordRoute
			vp.ReceiveSpoof = val.ReceiveSpoof
		}
	}
}

func (v vpMap) DeepCopy() vpMap {
	nvpMap := make(vpMap)
	for key, vp := range v {
		nvp := *vp
		nvpMap[key] = &nvp
	}
	return nvpMap
}

// RVPService is a usable VPService
type RVPService struct {
	rw         sync.RWMutex
	vps        vpMap
	lastUpdate time.Time
}

// GetVPs satisfies the VPService interface
func (rvp *RVPService) GetVPs(ctx context.Context, req *dm.VPRequest) (*dm.VPReturn, error) {
	rvp.rw.RLock()
	if len(rvp.vps) != 0 && time.Since(rvp.lastUpdate) < time.Minute*15 {
		defer rvp.rw.RUnlock()
		return &dm.VPReturn{
			Vps: rvp.vps.GetAll(),
		}, nil
	}
	rvp.rw.RUnlock()
	rvp.rw.Lock()
	defer rvp.rw.Unlock()
	// Need to recheck the update time after we get the lock cause someone else
	// may have gotten in before us TODO this better
	if len(rvp.vps) != 0 && time.Since(rvp.lastUpdate) < time.Minute*15 {
		return &dm.VPReturn{
			Vps: rvp.vps.GetAll(),
		}, nil
	}
	cc, err := grpc.Dial("controller.revtr.ccs.neu.edu:4382", grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	defer cc.Close()
	vps := controllerapi.NewControllerClient(cc)
	ret, err := vps.GetVPs(ctx, req)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	gotvps := ret.GetVps()
	log.Debug(gotvps)
	newVps := make(vpMap)
	for _, vp := range gotvps {
		newVps[vp.Ip] = vp
	}
	rvp.vps.Merge(newVps)
	rvp.lastUpdate = time.Now()
	return &dm.VPReturn{
		Vps: rvp.vps.GetAll(),
	}, nil
}

// runs in a go routine
func (rvp *RVPService) checkCapabilities() {
	t := time.NewTicker(time.Minute * 10)
	dirty := make(chan struct{})
	close(dirty)
	for {
		select {
		case <-t.C:
			rvp.rw.Lock()
			// Copy so we don't block everything while this is happening
			vps := rvp.vps.DeepCopy()
			rvp.rw.Unlock()
			// First check spoofing
			// Send a spoof from everyone to everyone else
			// Results determine who can spoof and who can receive spoofs
			testSpoofs(vps)
			// Test RR, just try to RR everyone
			testRR(vps)
			// Test TS, just try to prespec everyone
			testTS(vps)
			rvp.rw.Lock()
			rvp.vps.Update(vps)
			rvp.rw.Unlock()
		case <-dirty:
			rvp.GetVPs(context.Background(), &dm.VPRequest{})
			// Gets us an initial check without waiting 10 min
			rvp.rw.Lock()
			// Copy so we don't block everything while this is happening
			vps := rvp.vps.DeepCopy()
			rvp.rw.Unlock()
			// First check spoofing
			// Send a spoof from everyone to everyone else
			// Results determine who can spoof and who can receive spoofs
			testSpoofs(vps)
			// Test RR, just try to RR everyone
			testRR(vps)
			// Test TS, just try to prespec everyone
			testTS(vps)
			rvp.rw.Lock()
			rvp.vps.Update(vps)
			rvp.rw.Unlock()
			dirty = nil
		}
	}
}

func getRandomN(n int, vpm vpMap) []*dm.VantagePoint {
	randoms := rand.Perm(n)
	var vps []*dm.VantagePoint
	var ret []*dm.VantagePoint
	for _, v := range vpm {
		vps = append(vps, v)
	}
	if n > len(vps) {
		return vps
	}
	for _, r := range randoms {
		ret = append(ret, vps[r])
	}
	return ret
}

func testSpoofs(vpm vpMap) {
	lenvpm := len(vpm)
	var first bool
	var target uint32
	// Sending one spoof from each src to each dst so len in len(vpm)**2 - lenvpm
	var tests = make([]*dm.PingMeasurement, 0, lenvpm*(lenvpm-1))
	vps := getRandomN(50, vpm)
	for _, vp := range vps {
		if !first {
			first = true
			target = vp.Ip
		}
		dests := getRandomN(50, vpm)
		for _, d := range dests {
			ip, _ := util.Int32ToIPString(d.Ip)
			tests = append(tests, &dm.PingMeasurement{
				Count:   "1",
				Src:     vp.Ip,
				Dst:     target,
				Spoof:   true,
				SAddr:   ip,
				Timeout: 20,
			})
		}
	}
	cc, err := grpc.Dial("controller.revtr.ccs.neu.edu:4382", grpc.WithInsecure())
	if err != nil {
		log.Error(err)
		return
	}
	defer cc.Close()
	cl := controllerapi.NewControllerClient(cc)
	res, err := cl.Ping(context.Background(), &dm.PingArg{
		Pings: tests,
	})
	if err != nil {
		log.Error(err)
		return
	}
	for {
		p, err := res.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			return
		}
		if vp, ok := vpm[p.SpoofedFrom]; ok {
			vp.CanSpoof = true
		}
		if vp, ok := vpm[p.Src]; ok {
			vp.ReceiveSpoof = true
		}
	}
}

func testRR(vpm vpMap) {
	lenvpm := len(vpm)
	// Sending one spoof from each src to each dst so len in len(vpm)**2 - lenvpm
	var tests = make([]*dm.PingMeasurement, 0, lenvpm*(lenvpm-1))

	vps := getRandomN(50, vpm)
	for _, vp := range vps {
		dests := getRandomN(50, vpm)
		for _, d := range dests {
			tests = append(tests, &dm.PingMeasurement{
				Count:   "1",
				Src:     vp.Ip,
				Dst:     d.Ip,
				Timeout: 20,
			})
		}
	}
	cc, err := grpc.Dial("controller.revtr.ccs.neu.edu:4382", grpc.WithInsecure())
	if err != nil {
		log.Error(err)
		return
	}
	defer cc.Close()
	cl := controllerapi.NewControllerClient(cc)
	res, err := cl.Ping(context.Background(), &dm.PingArg{
		Pings: tests,
	})
	if err != nil {
		log.Error(err)
		return
	}
	for {
		p, err := res.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			return
		}
		if vp, ok := vpm[p.Src]; ok {
			vp.RecordRoute = true
		}
	}
}

func testTS(vpm vpMap) {
	lenvpm := len(vpm)
	// Sending one spoof from each src to each dst so len in len(vpm)**2 - lenvpm
	var tests = make([]*dm.PingMeasurement, 0, lenvpm*(lenvpm-1))
	vps := getRandomN(50, vpm)
	for _, vp := range vps {
		dests := getRandomN(50, vpm)
		for _, d := range dests {
			tests = append(tests, &dm.PingMeasurement{
				Count:     "1",
				Src:       vp.Ip,
				Dst:       d.Ip,
				Timeout:   20,
				TimeStamp: "tsonly",
			})
		}
	}
	cc, err := grpc.Dial("controller.revtr.ccs.neu.edu:4382", grpc.WithInsecure())
	if err != nil {
		log.Error(err)
		return
	}
	defer cc.Close()
	cl := controllerapi.NewControllerClient(cc)
	res, err := cl.Ping(context.Background(), &dm.PingArg{
		Pings: tests,
	})
	if err != nil {
		log.Error(err)
		return
	}
	for {
		p, err := res.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
		}
		if vp, ok := vpm[p.Src]; ok {
			vp.Timestamp = true
		}
	}
}

// NewRVPService creates a new RVPService
func NewRVPService() *RVPService {
	ret := &RVPService{
		vps: make(vpMap),
	}
	go ret.checkCapabilities()
	return ret
}
