package server

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"strings"
	"time"

	"golang.org/x/net/context"

	"github.com/NEU-SNS/ReverseTraceroute/controller/client"
	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice/pb"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice/types"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	procCollector = prometheus.NewProcessCollectorPIDFn(func() (int, error) {
		return os.Getpid(), nil
	}, getName())
	spooferGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: getName(),
		Subsystem: "spoofers",
		Name:      "current_spoofers",
		Help:      "The current number of spoofing VPS",
	})
)

const (
	defaultLimit = 150
	testSize     = 50
)

var id = rand.Uint32()

func init() {
	prometheus.MustRegister(procCollector)
	prometheus.MustRegister(spooferGauge)
}

func getName() string {
	name, err := os.Hostname()
	if err != nil {
		return fmt.Sprintf("vpservice_%d", id)
	}
	return fmt.Sprintf("vpservice_%s", strings.Replace(name, ".", "_", -1))
}

// VPServer is the interace for the vantage point server
type VPServer interface {
	GetVPs(*pb.VPRequest) (*pb.VPReturn, error)
	GetRRSpoofers(*pb.RRSpooferRequest) (*pb.RRSpooferResponse, error)
	GetTSSpoofers(*pb.TSSpooferRequest) (*pb.TSSpooferResponse, error)
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

type serverOptions struct {
	vpp types.VPProvider
	cl  client.Client
}

// NewServer creates a VPServer configured with the given options
func NewServer(opts ...Option) (VPServer, error) {
	var so serverOptions
	for _, opt := range opts {
		opt(&so)
	}
	s := server{opts: so}
	go s.checkCapabilitiesAndUpdate()
	return s, nil
}

type server struct {
	opts serverOptions
}

func (s server) GetVPs(pbr *pb.VPRequest) (*pb.VPReturn, error) {
	vps, err := s.opts.vpp.GetAllVPs()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return &pb.VPReturn{
		Vps: vps,
	}, nil
}

func (s server) GetRRSpoofers(rrs *pb.RRSpooferRequest) (*pb.RRSpooferResponse, error) {
	if rrs.Max == 0 {
		rrs.Max = defaultLimit
	}
	vps, err := s.opts.vpp.GetRRSpoofers(rrs.Addr, rrs.Max)
	if err != nil {
		return nil, err
	}
	var resp pb.RRSpooferResponse
	resp.Addr = rrs.Addr
	resp.Max = rrs.Max
	resp.Spoofers = vps
	return &resp, nil
}

func (s server) GetTSSpoofers(tsr *pb.TSSpooferRequest) (*pb.TSSpooferResponse, error) {
	if tsr.Max == 0 {
		tsr.Max = defaultLimit
	}
	vps, err := s.opts.vpp.GetTSSpoofers(tsr.Addr, tsr.Max)
	if err != nil {
		return nil, err
	}
	var resp pb.TSSpooferResponse
	resp.Addr = tsr.Addr
	resp.Max = tsr.Max
	resp.Spoofers = vps
	return &resp, nil
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
			}
			s.addOrUpdateVPs(vps.GetVps())
			cancel()
		}
	}
}

func (s server) addOrUpdateVPs(vps []*datamodel.VantagePoint) {
	var aVps []pb.VantagePoint
	for _, vp := range vps {
		var cvp pb.VantagePoint
		cvp.Hostname = vp.Hostname
		cvp.Ip = vp.Ip
		cvp.Site = vp.Site
		aVps = append(aVps, cvp)
	}
	err := s.opts.vpp.UpdateActiveVPs(aVps)
	if err != nil {
		log.Error(err)
	}
}

func (s server) checkCapabilities() {
	vps, err := s.opts.vpp.GetVPsForTesting(testSize)
	if err != nil {
		log.Error(err)
	}
	vpm := make(map[uint32]*pb.VantagePoint)
	var tests []*datamodel.PingMeasurement
	for _, vp := range vps {
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
	s.testRR(tests, vpm)
	s.testTS(tests, vpm)
	s.testSpoof(tests, vpm)
	for _, vp := range vpm {
		err := s.opts.vpp.UpdateVP(*vp)
		if err != nil {
			log.Error(err)
		}
	}
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
	for {
		p, err := ps.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			break
		}
		vp := vps[p.Src]
		vp.RecordRoute = false
		if len(p.Responses) > 0 {
			vp.RecordRoute = true
		}
	}
}

func (s server) testTS(pms []*datamodel.PingMeasurement, vps map[uint32]*pb.VantagePoint) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	var tests []*datamodel.PingMeasurement
	for _, pm := range pms {
		tspm := new(datamodel.PingMeasurement)
		*tspm = *pm
		tspm.TimeStamp = "tsonly"
		tests = append(tests, tspm)
	}
	ps, err := s.opts.cl.Ping(ctx, &datamodel.PingArg{
		Pings: tests,
	})
	if err != nil {
		log.Error(err)
		return
	}
	for {
		p, err := ps.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			break
		}
		vp := vps[p.Src]
		vp.Timestamp = false
		if len(p.Responses) > 0 {
			vp.Timestamp = true
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
	for {
		p, err := ps.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			break
		}
		vp := vps[p.SpoofedFrom]
		vp.Spoof = false
		if len(p.Responses) > 0 {
			vp.Spoof = true
		}
	}
}
