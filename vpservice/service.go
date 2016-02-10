package vpservice

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/controller/pb"
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	"github.com/golang/protobuf/proto"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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
		return fmt.Sprintf("vpservice_%d", id)
	}
	return fmt.Sprintf("vpservice_%s", strings.Replace(name, ".", "_", -1))
}

func init() {
	prometheus.MustRegister(procCollector)
}

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
	rootCA     string
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

	_, srvs, err := net.LookupSRV("controller", "tcp", "revtr.ccs.neu.edu")
	if err != nil {
		log.Error(err)
		return nil, err
	}
	creds, err := credentials.NewClientTLSFromFile(rvp.rootCA, srvs[0].Target)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	cc, err := grpc.Dial(fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port), grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Error(err)
		return nil, err
	}
	defer func() {
		err := cc.Close()
		if err != nil {
			log.Error(err)
		}
	}()
	vps := controllerapi.NewControllerClient(cc)
	ret, err := vps.GetVPs(ctx, req)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	gotvps := ret.GetVps()
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

// StoreInFile stores the current state in a file
func (rvp *RVPService) StoreInFile(file string) {
	f, err := os.Create(file)
	if err != nil {
		return
	}
	defer func() {
		err := f.Close()
		if err != nil {
			log.Error(err)
		}
	}()
	for _, v := range rvp.vps {
		_, err := f.WriteString(v.String() + "\n")
		if err != nil {
			log.Error(err)
		}
	}
}

// LoadFromFile loads the state in from a file
func (rvp *RVPService) LoadFromFile(file string) {
	f, err := os.Open(file)
	if err != nil {
		return
	}
	defer func() {
		err := f.Close()
		if err != nil {
			log.Error(err)
		}
	}()
	scan := bufio.NewScanner(f)
	for scan.Scan() {
		vp := &dm.VantagePoint{}
		err := proto.UnmarshalText(scan.Text(), vp)
		if err != nil {
			continue
		}
		rvp.vps[vp.Ip] = vp
	}

}

// runs in a go routine
func (rvp *RVPService) checkCapabilities() {
	t := time.NewTicker(time.Minute * 10)
	dirty := make(chan struct{})
	close(dirty)
	for {
		select {
		case <-t.C:
			log.Debug("Checking Capabilities")
			rvp.rw.Lock()
			// Copy so we don't block everything while this is happening
			vps := rvp.vps.DeepCopy()
			rvp.rw.Unlock()
			// First check spoofing
			// Send a spoof from everyone to everyone else
			// Results determine who can spoof and who can receive spoofs
			testSpoofs(vps, rvp.rootCA)
			// Test RR, just try to RR everyone
			testRR(vps, rvp.rootCA)
			// Test TS, just try to prespec everyone
			testTS(vps, rvp.rootCA)
			rvp.rw.Lock()
			rvp.vps.Update(vps)
			rvp.rw.Unlock()
		case <-dirty:
			_, err := rvp.GetVPs(context.Background(), &dm.VPRequest{})
			if err != nil {
				log.Error(err)
			}
			// Gets us an initial check without waiting 10 min
			rvp.rw.Lock()
			// Copy so we don't block everything while this is happening
			vps := rvp.vps.DeepCopy()
			rvp.rw.Unlock()
			// First check spoofing
			// Send a spoof from everyone to everyone else
			// Results determine who can spoof and who can receive spoofs
			testSpoofs(vps, rvp.rootCA)
			// Test RR, just try to RR everyone
			testRR(vps, rvp.rootCA)
			// Test TS, just try to prespec everyone
			testTS(vps, rvp.rootCA)
			rvp.rw.Lock()
			rvp.vps.Update(vps)
			rvp.rw.Unlock()
			dirty = nil
		}
	}
}

func getRandomN(n int, vpm vpMap) []*dm.VantagePoint {
	var vps []*dm.VantagePoint
	var ret []*dm.VantagePoint
	for _, v := range vpm {
		vps = append(vps, v)
	}
	if n > len(vps) {
		return vps
	}
	randoms := rand.Perm(len(vps))
	for _, r := range randoms[:n] {
		ret = append(ret, vps[r])
	}
	return ret
}

func testSpoofs(vpm vpMap, rootCA string) {
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

	_, srvs, err := net.LookupSRV("controller", "tcp", "revtr.ccs.neu.edu")
	if err != nil {
		log.Error(err)
		return
	}
	connstr := fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port)
	creds, err := credentials.NewClientTLSFromFile(rootCA, srvs[0].Target)
	if err != nil {
		log.Error(err)
		return
	}
	cc, err := grpc.Dial(connstr, grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Error(err)
		return
	}
	defer func() {
		err := cc.Close()
		if err != nil {
			log.Error(err)
		}
	}()
	cl := controllerapi.NewControllerClient(cc)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	res, err := cl.Ping(ctx, &dm.PingArg{
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
		log.Debug("Got spoof response: ", p)
		if vp, ok := vpm[p.SpoofedFrom]; ok {
			vp.CanSpoof = true
		}
		if vp, ok := vpm[p.Src]; ok {
			vp.ReceiveSpoof = true
		}
	}
}

func testRR(vpm vpMap, rootCA string) {
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

	_, srvs, err := net.LookupSRV("controller", "tcp", "revtr.ccs.neu.edu")
	if err != nil {
		log.Error(err)
		return
	}
	connstr := fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port)
	creds, err := credentials.NewClientTLSFromFile(rootCA, srvs[0].Target)
	if err != nil {
		log.Error(err)
		return
	}
	cc, err := grpc.Dial(connstr, grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Error(err)
		return
	}
	defer func() {
		err := cc.Close()
		if err != nil {
			log.Error(err)
		}
	}()
	cl := controllerapi.NewControllerClient(cc)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	res, err := cl.Ping(ctx, &dm.PingArg{
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

func testTS(vpm vpMap, rootCA string) {
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
	_, srvs, err := net.LookupSRV("controller", "tcp", "revtr.ccs.neu.edu")
	if err != nil {
		log.Error(err)
		return
	}
	connstr := fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port)
	creds, err := credentials.NewClientTLSFromFile(rootCA, srvs[0].Target)
	if err != nil {
		log.Error(err)
		return
	}
	cc, err := grpc.Dial(connstr, grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Error(err)
		return
	}
	defer func() {
		err := cc.Close()
		if err != nil {
			log.Error(err)
		}
	}()
	cl := controllerapi.NewControllerClient(cc)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	res, err := cl.Ping(ctx, &dm.PingArg{
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
		if p == nil {
			continue
		}
		if vp, ok := vpm[p.Src]; ok {
			vp.Timestamp = true
		}
	}
}

// NewRVPService creates a new RVPService
func NewRVPService(rootCA string) *RVPService {
	ret := &RVPService{
		vps:    make(vpMap),
		rootCA: rootCA,
	}
	go ret.checkCapabilities()
	ret.startHTTP()
	return ret
}

func startHTTP(addr string) {
	for {
		log.Error(http.ListenAndServe(addr, nil))
	}
}

func (rvp *RVPService) startHTTP() {
	http.Handle("/metrics", prometheus.Handler())
	go startHTTP(":8080")
}
