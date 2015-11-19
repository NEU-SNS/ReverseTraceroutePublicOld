package vpservice

import (
	"io"
	"sync"
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/controllerapi"
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

func testSpoofs(vpm vpMap) {
	lenvpm := len(vpm)
	var first bool
	var target uint32
	var count, count2 int
	// Sending one spoof from each src to each dst so len in len(vpm)**2 - lenvpm
	var tests = make([]*dm.PingMeasurement, 0, lenvpm*(lenvpm-1))
	for _, vps := range vpm {
		if count == 50 {
			break
		}
		if !first {
			first = true
			target = vps.Ip
		}
		for _, vpd := range vpm {
			if count2 == 50 {
				break
			}
			if vps.Ip == vpd.Ip {
				continue
			}
			ip, _ := util.Int32ToIPString(vpd.Ip)
			tests = append(tests, &dm.PingMeasurement{
				Count: "1",
				Src:   vps.Ip,
				Dst:   target,
				Spoof: true,
				SAddr: ip,
			})
			count2++
		}
		count++
		count2 = 0
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
	var count, count2 int
	for _, vps := range vpm {
		if count == 50 {
			break
		}
		for _, vpd := range vpm {
			if vps.Ip == vpd.Ip {
				continue
			}
			if count2 == 50 {
				break
			}
			tests = append(tests, &dm.PingMeasurement{
				Count: "1",
				Src:   vps.Ip,
				Dst:   vpd.Ip,
				RR:    true,
			})
			count2++
		}
		count2 = 0
		count++
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
			vp.RecordRoute = true
		}
	}
}

func testTS(vpm vpMap) {
	lenvpm := len(vpm)
	// Sending one spoof from each src to each dst so len in len(vpm)**2 - lenvpm
	var tests = make([]*dm.PingMeasurement, 0, lenvpm*(lenvpm-1))
	var count, count2 int
	for _, vps := range vpm {
		if count == 50 {
			break
		}
		for _, vpd := range vpm {
			if vps.Ip == vpd.Ip {
				continue
			}
			if count2 == 50 {
				break
			}
			tests = append(tests, &dm.PingMeasurement{
				Count:     "1",
				Src:       vps.Ip,
				Dst:       vpd.Ip,
				TimeStamp: "tsonly",
			})
			count2++
		}
		count++
		count2 = 0
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
