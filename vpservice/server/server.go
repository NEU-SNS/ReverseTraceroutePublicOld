/*
Copyright (c) 2015, Northeastern University
 All rights reserved.

 Redistribution and use in source and binary forms, with or without
 modification, are permitted provided that the following conditions are met:
     * Redistributions of source code must retain the above copyright
       notice, this list of conditions and the following disclaimer.
     * Redistributions in binary form must reproduce the above copyright
       notice, this list of conditions and the following disclaimer in the
       documentation and/or other materials provided with the distribution.
     * Neither the name of the Northeastern University nor the
       names of its contributors may be used to endorse or promote products
       derived from this software without specific prior written permission.

 THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
 ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 DISCLAIMED. IN NO EVENT SHALL Northeastern University BE LIABLE FOR ANY
 DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

package server

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"golang.org/x/net/context"

	"github.com/NEU-SNS/ReverseTraceroute/controller/client"
	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice/filters"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice/pb"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice/types"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	nameSpace     = "vpservice"
	procCollector = prometheus.NewProcessCollectorPIDFn(func() (int, error) {
		return os.Getpid(), nil
	}, nameSpace)
	spooferGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: nameSpace,
		Subsystem: "vantage_points",
		Name:      "current_spoofers",
		Help:      "The current number of spoofing VPS",
	})
	onlineVPGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: nameSpace,
		Subsystem: "vantage_points",
		Name:      "online_vps",
		Help:      "The current number of online vps",
	})
	activeSiteGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: nameSpace,
		Subsystem: "sites",
		Name:      "active_sites",
		Help:      "The current number of active sites",
	})
	spoofingSiteGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: nameSpace,
		Subsystem: "sites",
		Name:      "spoofing_sites",
		Help:      "The current number of active spoofing sites",
	})
	onlineVPGaugeVec = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: "vantage_points",
		Name:      "vp_status",
		Help:      "The status of individual vantage points, 1 is online 0 is offline.",
	}, []string{"vp"})
	quarantinedVPGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: nameSpace,
		Subsystem: "vantage_points",
		Name:      "quarantined_vps",
		Help:      "The current number of quarantined vps",
	})
)

const (
	defaultLimit = 250
	testSize     = 100
)

func init() {
	prometheus.MustRegister(procCollector)
	prometheus.MustRegister(spooferGauge)
	prometheus.MustRegister(onlineVPGauge)
	prometheus.MustRegister(activeSiteGauge)
	prometheus.MustRegister(spoofingSiteGauge)
	prometheus.MustRegister(onlineVPGaugeVec)
	prometheus.MustRegister(quarantinedVPGauge)
}

// VPServer is the interace for the vantage point server
type VPServer interface {
	GetVPs(*pb.VPRequest) (*pb.VPReturn, error)
	GetRRSpoofers(*pb.RRSpooferRequest) (*pb.RRSpooferResponse, error)
	GetTSSpoofers(*pb.TSSpooferRequest) (*pb.TSSpooferResponse, error)
	QuarantineVPs(vps []types.Quarantine) error
	UnquarantineVPs(vps []types.Quarantine) error
	GetLastQuarantine(ip uint32) (types.Quarantine, error)
	GetQuarantined() ([]types.Quarantine, error)
}

// Option configures the server
type Option func(*serverOptions)

// WithVPProvider configures the server with the given VPProvider
func WithVPProvider(vpp types.VPProvider) Option {
	return func(so *serverOptions) {
		so.vpp = vpp
	}
}

// WithClient configures the server with the given client
func WithClient(c client.Client) Option {
	return func(so *serverOptions) {
		so.cl = c
	}
}

// WithRRFilter configures the server with the given RRFilter
func WithRRFilter(rrf filters.RRFilter) Option {
	return func(so *serverOptions) {
		so.rrf = rrf
	}
}

// WithTSFilter configures the server with the given TSFilter
func WithTSFilter(tsf filters.TSFilter) Option {
	return func(so *serverOptions) {
		so.tsf = tsf
	}

}

type serverOptions struct {
	vpp types.VPProvider
	cl  client.Client
	rrf filters.RRFilter
	tsf filters.TSFilter
}

// NewServer creates a VPServer configured with the given options
func NewServer(opts ...Option) (VPServer, error) {
	var so serverOptions
	for _, opt := range opts {
		opt(&so)
	}
	s := server{opts: so, rrf: makeRRF(so.rrf), tsf: makeTSF(so.tsf)}
	s.initGuages()
	go s.checkCapabilitiesAndUpdate()
	go s.updateGauges()
	go s.tryUnquarantine()
	return s, nil
}

func makeTSF(f filters.TSFilter) tsFilter {
	return func(vps []types.TSVantagePoint) []*pb.VantagePoint {
		var fvps []types.TSVantagePoint
		fvps = vps
		if f != nil {
			fvps = f(vps)
		}
		var final []*pb.VantagePoint
		for _, vp := range fvps {
			currvp := vp.VantagePoint
			final = append(final, &currvp)
		}
		return final
	}
}

func makeRRF(f filters.RRFilter) rrFilter {
	return func(vps []types.RRVantagePoint) []*pb.VantagePoint {
		var fvps []types.RRVantagePoint
		fvps = vps
		if f != nil {
			fvps = f(vps)
		}
		var final []*pb.VantagePoint
		for _, vp := range fvps {
			currvp := vp.VantagePoint
			final = append(final, &currvp)
		}
		return final
	}
}

type tsFilter func([]types.TSVantagePoint) []*pb.VantagePoint
type rrFilter func([]types.RRVantagePoint) []*pb.VantagePoint

type server struct {
	opts serverOptions
	tsf  tsFilter
	rrf  rrFilter
}

func (s server) QuarantineVPs(vps []types.Quarantine) error {
	for _, vp := range vps {
		// Now that a node is quarantened, remove it from the monitoring
		onlineVPGaugeVec.DeleteLabelValues(vp.GetVP().Hostname)
	}
	return s.opts.vpp.QuarantineVPs(vps)
}

func (s server) UnquarantineVPs(vps []types.Quarantine) error {
	return s.opts.vpp.UnquarantineVPs(vps)
}

func (s server) GetVPs(pbr *pb.VPRequest) (*pb.VPReturn, error) {
	vps, err := s.opts.vpp.GetVPs()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return &pb.VPReturn{
		Vps: vps,
	}, nil
}

func (s server) GetRRSpoofers(rrs *pb.RRSpooferRequest) (*pb.RRSpooferResponse, error) {
	log.Debug("Getting rrspoofers ", rrs)
	if rrs.Max == 0 {
		rrs.Max = defaultLimit
	}
	vps, err := s.opts.vpp.GetRRSpoofers(rrs.Addr)
	if err != nil {
		log.Debug(err)
		return nil, err
	}
	log.Debug("Got ", len(vps), " rr spoofers: ", vps)
	var resp pb.RRSpooferResponse
	resp.Addr = rrs.Addr
	resp.Max = rrs.Max
	resp.Spoofers = s.rrf(vps)
	log.Debug("filtered rr spoofers: ", resp.Spoofers)
	if uint32(len(resp.Spoofers)) > rrs.Max {
		resp.Spoofers = resp.Spoofers[:rrs.Max]
	}
	return &resp, nil
}

func (s server) GetTSSpoofers(tsr *pb.TSSpooferRequest) (*pb.TSSpooferResponse, error) {
	log.Debug("Getting tsspoofers ", tsr)
	if tsr.Max == 0 {
		tsr.Max = defaultLimit
	}
	vps, err := s.opts.vpp.GetTSSpoofers()
	if err != nil {
		log.Debug(err)
		return nil, err
	}
	var resp pb.TSSpooferResponse
	resp.Addr = tsr.Addr
	resp.Max = tsr.Max
	resp.Spoofers = s.tsf(vps)
	if uint32(len(resp.Spoofers)) > tsr.Max {
		resp.Spoofers = resp.Spoofers[:tsr.Max]
	}
	return &resp, nil
}

func (s server) GetLastQuarantine(ip uint32) (types.Quarantine, error) {
	return s.opts.vpp.GetLastQuarantine(ip)
}

func (s server) GetQuarantined() ([]types.Quarantine, error) {
	return s.opts.vpp.GetQuarantined()
}

// call in a goroutine
// loop forever checking the capabilities of vantage points
// as well checking for new vps/vps being removed
func (s server) checkCapabilitiesAndUpdate() {
	vpsTimer := time.NewTicker(time.Minute * 5)
	for {
		select {
		case <-vpsTimer.C:
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
			vps, err := s.opts.cl.GetVps(ctx, &datamodel.VPRequest{})
			if err != nil {
				log.Error(err)
				cancel()
				continue
			} else {
				s.addOrUpdateVPs(vps.GetVps())
				cancel()
			}
			s.checkCapabilities()
		}
	}
}

func (s server) addOrUpdateVPs(vps []*datamodel.VantagePoint) {
	var aVps []*pb.VantagePoint
	for _, vp := range vps {
		cvp := new(pb.VantagePoint)
		cvp.Hostname = vp.Hostname
		cvp.Ip = vp.Ip
		cvp.Site = vp.Site
		aVps = append(aVps, cvp)
	}
	add, rem, err := s.opts.vpp.UpdateActiveVPs(aVps)
	if err != nil {
		log.Error(err)
		return
	}
	// we dont want to monitor any of the vps that are currently being quarantined
	// filter them out of the list
	quar, err := s.opts.vpp.GetQuarantined()
	if err != nil {
		log.Error(err)
		return
	}
	quarantinedVPGauge.Set(float64(len(quar)))
	quarMap := make(map[string]struct{})
	for _, vp := range quar {
		quarMap[vp.GetVP().Hostname] = struct{}{}
	}
	log.Debug("Quarantined vps ", quar)
	log.Debug("Adding ", add)
	log.Debug("Removing ", rem)
	// if the nodes are quarantened, we don't want to monitor them
	// so skip them
	for _, vp := range add {
		if _, ok := quarMap[vp.Hostname]; !ok {
			onlineVPGaugeVec.WithLabelValues(vp.Hostname).Set(1)
		}
	}
	for _, vp := range rem {
		if _, ok := quarMap[vp.Hostname]; !ok {
			onlineVPGaugeVec.WithLabelValues(vp.Hostname).Set(-1)
		}
	}
}

func (s server) tryUnquarantine() {
	tick := time.NewTicker(2 * time.Minute)
	for {
		select {
		case t := <-tick.C:
			log.Debug("Trying unquarantine for time: ", t)
			vps, err := s.opts.vpp.GetAllVPs()
			if err != nil {
				log.Error(err)
				continue
			}
			vpmap := make(map[string]*pb.VantagePoint)
			for _, vp := range vps {
				vpmap[vp.Hostname] = vp
			}
			qs, err := s.opts.vpp.GetQuarantined()
			if err != nil {
				log.Error(err)
				continue
			}
			var unquar []types.Quarantine
			var updateQuar []types.Quarantine
			for _, q := range qs {
				log.Debug("Checking Expired on ", q.GetExpire(), " at", time.Now())
				if q.Expired(time.Now()) {
					unquar = append(unquar, q)
					continue
				}
				if q.Elapsed(time.Now()) {
					log.Debug("Checking elapsed on ", q.GetBackoff(), " at", time.Now())
					// If our backoff elapsed, see if the node is up and running
					if vp, ok := vpmap[q.GetVP().Hostname]; ok {
						if !shouldQuarantine(*vp) {
							// Up and running, remove it from the quarantines
							unquar = append(unquar, q)
							continue
						}
					}
					// Not up or not running, set the next backoff and update in db
					q.NextBackoff()
					updateQuar = append(updateQuar, q)
				}
			}
			// if any should be removed from quarantine do that now
			if len(unquar) > 0 {
				if err := s.opts.vpp.UnquarantineVPs(unquar); err != nil {
					log.Error(err)
				}
			}
			// if any should still be quarantined, update them in the db
			if len(updateQuar) > 0 {
				if err := s.opts.vpp.UpdateQuarantines(updateQuar); err != nil {
					log.Error(err)
				}
			}
		}
	}
}

func (s server) checkCapabilities() {
	vps, err := s.opts.vpp.GetVPsForTesting(testSize)
	if err != nil {
		log.Error(err)
		return
	}
	var onePerSite []*pb.VantagePoint
	siteMap := make(map[string]*pb.VantagePoint)
	for _, vp := range vps {
		siteMap[vp.Site] = vp
	}
	for _, vp := range siteMap {
		onePerSite = append(onePerSite, vp)
	}
	log.Debug("Checking Capabilities for: ", onePerSite)
	vpm := make(map[uint32]*pb.VantagePoint)
	var tests []*datamodel.PingMeasurement
	for _, vp := range onePerSite {
		vp.RecSpoof = false
		vp.Spoof = false
		vp.RecordRoute = false
		vp.Timestamp = false
		vp.Ping = false
		vp.Trace = false
		vpm[vp.Ip] = vp
		for _, d := range vps {
			if d.Ip == vp.Ip {
				continue
			}
			tests = append(tests, &datamodel.PingMeasurement{
				Count:   "1",
				Src:     vp.Ip,
				Dst:     d.Ip,
				Timeout: 20,
			})
		}
	}
	var traceTests []*datamodel.TracerouteMeasurement
	for _, vp := range onePerSite {
		for _, d := range vps {
			if d.Ip == vp.Ip {
				continue
			}
			traceTests = append(traceTests, &datamodel.TracerouteMeasurement{
				Src:        vp.Ip,
				Dst:        vp.Ip,
				Timeout:    30,
				Wait:       "2",
				Attempts:   "1",
				LoopAction: "1",
				Loops:      "3",
			})
		}
	}
	s.testRR(tests, vpm)
	s.testTS(tests, vpm)
	s.testSpoof(tests, vpm)
	s.testPing(tests, vpm)
	s.testTrace(traceTests, vpm)
	var quarantines []types.Quarantine
	for _, vp := range vpm {
		err := s.opts.vpp.UpdateVP(*vp)
		if err != nil {
			log.Error(err)
		}
		if shouldQuarantine(*vp) {
			quar, err := s.GetLastQuarantine(vp.Ip)
			if err != nil {
				quarantines = append(quarantines,
					types.NewDefaultQuarantine(*vp, nil, types.CantPerformMeasurement))
			} else {
				quarantines = append(quarantines,
					types.NewDefaultQuarantine(*vp, quar, types.CantPerformMeasurement))
			}
		}
	}
	if err := s.QuarantineVPs(quarantines); err != nil {
		log.Error(err)
	}
}

func shouldQuarantine(vp pb.VantagePoint) bool {
	// if we can perform any type of measurement, quarantine the vp
	if !vp.Ping &&
		!vp.Trace &&
		!vp.RecordRoute &&
		!vp.Timestamp &&
		!vp.Spoof {
		return true
	}
	return false
}

func (s server) testTrace(tms []*datamodel.TracerouteMeasurement, vps map[uint32]*pb.VantagePoint) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*40)
	defer cancel()
	var wg sync.WaitGroup
	wg.Add(len(tms))
	for _, tm := range tms {
		go func(t *datamodel.TracerouteMeasurement, vp *pb.VantagePoint) {
			defer wg.Done()
			ts, err := s.opts.cl.Traceroute(ctx, &datamodel.TracerouteArg{
				Traceroutes: []*datamodel.TracerouteMeasurement{t},
			})
			if err != nil {
				log.Error(err)
				return
			}
			ts.CloseSend()
			for {
				t, err := ts.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					if grpc.Code(err) != codes.DeadlineExceeded {
						log.Error(err)
					}
					break
				}
				if t.Error == "" {
					vp.Trace = true
				}
			}
		}(tm, vps[tm.Src])
	}
	wg.Wait()
}

func (s server) testPing(pms []*datamodel.PingMeasurement, vps map[uint32]*pb.VantagePoint) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	var wg sync.WaitGroup
	wg.Add(len(pms))
	for _, pm := range pms {
		go func(p *datamodel.PingMeasurement, vp *pb.VantagePoint) {
			defer wg.Done()
			ps, err := s.opts.cl.Ping(ctx, &datamodel.PingArg{
				Pings: []*datamodel.PingMeasurement{p},
			})
			if err != nil {
				log.Error(err)
				return
			}
			ps.CloseSend()
			for {
				p, err := ps.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					if grpc.Code(err) != codes.DeadlineExceeded {
						log.Error(err)
					}
					break
				}
				if p.Error == "" && len(p.Responses) > 0 {
					vp.Ping = true
				}
			}
		}(pm, vps[pm.Src])
	}
	wg.Wait()
}

func (s server) testRR(pms []*datamodel.PingMeasurement, vps map[uint32]*pb.VantagePoint) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	var tests []*datamodel.PingMeasurement
	for _, pm := range pms {
		rrpm := new(datamodel.PingMeasurement)
		*rrpm = *pm
		rrpm.RR = true
		tests = append(tests, rrpm)
	}
	ps, err := s.opts.cl.Ping(ctx, &datamodel.PingArg{
		Pings: tests,
	})
	if err != nil {
		log.Error(err)
		return
	}
	ps.CloseSend()
	for {
		p, err := ps.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			if grpc.Code(err) != codes.DeadlineExceeded {
				log.Error(err)
			}
			break
		}

		// Some of the PLE nodes are using dhcp and have addresses
		// in private ranges. For some of these cases, the public
		// ips that are mapped to the nodes are the first address
		// in the RR option. This next section looks for that case
		// as well as just checking the Src of the probe
		if len(p.Responses) == 0 {
			continue
		}
		r1 := p.Responses[0]
		addr1 := new(uint32)
		if len(r1.RR) > 0 {
			*addr1 = r1.RR[0]
		}
		if vp, ok := vps[p.Src]; ok {
			vp.RecordRoute = true
			continue
		}
		if vp, ok := vps[*addr1]; ok {
			vp.RecordRoute = true
			continue
		}
		if p.Statistics.Loss != 1 {
			log.Error("Got rr response with invalid src: ", p)
		}
	}
}

type ipaddress uint32

func (ip ipaddress) String() string {
	ips, _ := util.Int32ToIPString(uint32(ip))
	return ips
}

// like the RR tests, the src of the measurement might not match the src of the returned probe struct
// this is because some PLE nodes have private ips. In order to match things back together,
// im setting tsprespec with the public src address as the first address and the dst as the second
// with that I can match off of the first ts and address in the response if the src doesn't match
// any probe that I sent
func (s server) testTS(pms []*datamodel.PingMeasurement, vps map[uint32]*pb.VantagePoint) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	var tests []*datamodel.PingMeasurement
	for _, pm := range pms {
		tspm := new(datamodel.PingMeasurement)
		*tspm = *pm
		tspm.RR = false
		tspm.TimeStamp = fmt.Sprintf("tsprespec=%v,%v", ipaddress(pm.Src), ipaddress(pm.Dst))
		tests = append(tests, tspm)
	}
	ps, err := s.opts.cl.Ping(ctx, &datamodel.PingArg{
		Pings: tests,
	})
	if err != nil {
		log.Error(err)
		return
	}
	ps.CloseSend()
	for {
		p, err := ps.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			if grpc.Code(err) != codes.DeadlineExceeded {
				log.Error(err)
			}
			break
		}
		if len(p.Responses) == 0 {
			continue
		}
		r1 := p.Responses[0]
		addr1 := new(uint32)
		if len(r1.Tsandaddr) > 0 {
			*addr1 = r1.Tsandaddr[0].Ip
		}
		if vp, ok := vps[p.Src]; ok {
			vp.Timestamp = true
			if len(p.Responses) > 0 {
				vp.RecordRoute = true
			}
			continue
		}
		if vp, ok := vps[*addr1]; ok {
			vp.Timestamp = true
			continue
		}
		if p.Statistics.Loss != 1 {
			log.Error("Got timestamp with wrong source: ", p)
		}
	}
}

func (s server) testSpoof(pms []*datamodel.PingMeasurement, vps map[uint32]*pb.VantagePoint) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	var tests []*datamodel.PingMeasurement
	for i, pm := range pms {
		spm := new(datamodel.PingMeasurement)
		*spm = *pm
		spm.RR = false
		spm.TimeStamp = ""
		// Spoof as the src of the next vp in the list
		spoofAs := pms[(i+1)%len(pms)]
		sip, _ := util.Int32ToIPString(spoofAs.Src)
		spm.Spoof = true
		spm.SAddr = sip
		tests = append(tests, spm)
	}
	ps, err := s.opts.cl.Ping(ctx, &datamodel.PingArg{
		Pings: tests,
	})
	if err != nil {
		log.Error(err)
		return
	}
	ps.CloseSend()
	for {
		p, err := ps.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			if grpc.Code(err) != codes.DeadlineExceeded {
				log.Error(err)
			}
			break
		}
		if vp, ok := vps[p.SpoofedFrom]; ok {
			vp.Spoof = true
			if vp, ok := vps[p.Dst]; ok {
				vp.RecSpoof = true
			}
			continue
		}
		if p.Statistics.Loss != 1 {
			log.Error("Got spoofed probe with invalid spoofed from addr: ", p)
		}
	}
}

func doGauges(vps []*pb.VantagePoint, quar map[string]struct{}) {
	onlineVPGauge.Set(float64(len(vps)))
	var spoofCnt float64
	for _, vp := range vps {
		if vp.Spoof {
			spoofCnt++
		}
	}
	spooferGauge.Set(spoofCnt)
	siteMap := make(map[string]struct{})
	for _, vp := range vps {
		onlineVPGaugeVec.WithLabelValues(vp.Hostname).Set(1)
		siteMap[vp.Site] = struct{}{}
	}
	var siteCnt float64
	for _ = range siteMap {
		siteCnt++
	}
	activeSiteGauge.Set(siteCnt)
	spoofSiteMap := make(map[string]struct{})
	for _, vp := range vps {
		if vp.Spoof {
			spoofSiteMap[vp.Site] = struct{}{}
		}
	}
	var spSiteCnt float64
	for _ = range spoofSiteMap {
		spSiteCnt++
	}
	spoofingSiteGauge.Set(spSiteCnt)
}

func (s server) initGuages() {
	quar, err := s.opts.vpp.GetQuarantined()
	if err != nil {
		log.Error(err)
		return
	}
	quarantinedVPGauge.Set(float64(len(quar)))
	quarMap := make(map[string]struct{})
	for _, vp := range quar {
		quarMap[vp.GetVP().Hostname] = struct{}{}
	}
	vps, err := s.opts.vpp.GetVPs()
	if err != nil {
		log.Error(err)
		return
	}
	for _, vp := range vps {
		if _, ok := quarMap[vp.Hostname]; !ok {
			onlineVPGaugeVec.WithLabelValues(vp.Hostname).Set(1)
		}
	}
	doGauges(vps, quarMap)
}

// call in a goroutine
func (s server) updateGauges() {
	tick := time.NewTicker(time.Minute * 2)
	for {
		select {
		case <-tick.C:
			vps, err := s.opts.vpp.GetVPs()
			if err != nil {
				log.Error(err)
				continue
			}
			quar, err := s.opts.vpp.GetQuarantined()
			if err != nil {
				log.Error(err)
				return
			}
			quarantinedVPGauge.Set(float64(len(quar)))
			quarMap := make(map[string]struct{})
			for _, vp := range quar {
				quarMap[vp.GetVP().Hostname] = struct{}{}
			}
			doGauges(vps, quarMap)
		}
	}
}
