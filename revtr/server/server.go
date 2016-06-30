package server

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	at "github.com/NEU-SNS/ReverseTraceroute/atlas/client"
	"github.com/NEU-SNS/ReverseTraceroute/controller/client"
	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/clustermap"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/ip_utils"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/pb"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/repository"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/reverse_traceroute"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/runner"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/types"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	vpservice "github.com/NEU-SNS/ReverseTraceroute/vpservice/client"
	vppb "github.com/NEU-SNS/ReverseTraceroute/vpservice/pb"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	nameSpace   = "revtr"
	goCollector = prometheus.NewProcessCollectorPIDFn(func() (int, error) {
		return os.Getpid(), nil
	}, nameSpace)
	runningRevtrs = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: nameSpace,
		Subsystem: "revtrs",
		Name:      "running_revtrs",
		Help:      "The count of currently running reverse traceroutes.",
	})
	// ErrNoRevtrsToRun is returned when there are no revtrs given in a batch
	ErrNoRevtrsToRun = fmt.Errorf("no runnable revtrs in the batch")
	// ErrConnectFailed is returned when connecting to the services failed
	ErrConnectFailed = fmt.Errorf("could not connect to services")
	// ErrFailedToCreateBatch is returned when creating the batch of revtrs fails
	ErrFailedToCreateBatch = fmt.Errorf("could not create batch")
)

func init() {
	prometheus.MustRegister(goCollector)
	prometheus.MustRegister(runningRevtrs)
}

type errorf func() error

func logError(ef errorf) {
	if err := ef(); err != nil {
		log.Error(err)
	}
}

// BatchIDError is returned when an invalid batch id is sent
type BatchIDError struct {
	batchID uint32
}

func (be BatchIDError) Error() string {
	return fmt.Sprintf("invalid batch id %d", be.batchID)
}

// SrcError is returned when an invalid src address is given
type SrcError struct {
	src string
}

func (se SrcError) Error() string {
	return fmt.Sprintf("invalid src address %s", se.src)
}

// DstError is returned when an invalid src address is given
type DstError struct {
	dst string
}

func (de DstError) Error() string {
	return fmt.Sprintf("invalid dst address %s", de.dst)
}

func validSrc(src string, vps []*vppb.VantagePoint) (string, bool) {
	for _, vp := range vps {
		s, _ := util.Int32ToIPString(vp.Ip)
		if vp.Hostname == src || s == src {
			return s, true
		}
	}
	return "", false
}

func validDest(dst string, vps []*vppb.VantagePoint) (string, bool) {
	var notIP bool
	ip := net.ParseIP(dst)
	if ip == nil {
		notIP = true
	}
	if notIP {
		for _, vp := range vps {
			if vp.Hostname == dst {
				ips, _ := util.Int32ToIPString(vp.Ip)
				return ips, true
			}
		}
		res, err := net.LookupHost(dst)
		if err != nil {
			log.Error(err)
			return "", false
		}
		if len(res) == 0 {
			return "", false
		}
		return res[0], true
	}
	if iputil.IsPrivate(ip) {
		return "", false
	}
	return dst, true
}

func verifyAddrs(src, dst string, vps []*vppb.VantagePoint) (string, string, error) {
	nsrc, valid := validSrc(src, vps)
	if !valid {
		log.Errorf("Invalid source: %s", src)
		return "", "", SrcError{src: src}
	}
	ndst, valid := validDest(dst, vps)
	if !valid {
		log.Errorf("Invalid destination: %s", dst)
		return "", "", DstError{dst: dst}
	}
	// ensure they're valid ips
	if net.ParseIP(nsrc) == nil {
		log.Errorf("Invalid source: %s", nsrc)
		return "", "", SrcError{src: nsrc}
	}
	if net.ParseIP(ndst) == nil {
		log.Errorf("Invalid destination: %s", ndst)
		return "", "", DstError{dst: ndst}
	}
	return nsrc, ndst, nil
}

// RTStore is the interface for storing/loading/allowing revtrs to be run
type RTStore interface {
	GetUserByKey(string) (pb.RevtrUser, error)
	StoreRevtr(pb.ReverseTraceroute) error
	GetRevtrsInBatch(uint32, uint32) ([]*pb.ReverseTraceroute, error)
	CreateRevtrBatch([]*pb.RevtrMeasurement, string) ([]*pb.RevtrMeasurement, uint32, error)
	StoreBatchedRevtrs([]pb.ReverseTraceroute) error
}

// RevtrServer in the interface for the revtr server
type RevtrServer interface {
	RunRevtr(*pb.RunRevtrReq) (*pb.RunRevtrResp, error)
	GetRevtr(*pb.GetRevtrReq) (*pb.GetRevtrResp, error)
	GetSources(*pb.GetSourcesReq) (*pb.GetSourcesResp, error)
	AddRevtr(pb.RevtrMeasurement) (uint32, error)
	StartRevtr(context.Context, uint32) (<-chan Status, error)
}

type serverOptions struct {
	rts                       RTStore
	vps                       vpservice.VPSource
	as                        types.AdjacencySource
	cs                        types.ClusterSource
	ca                        types.Cache
	run                       runner.Runner
	rootCA, certFile, keyFile string
}

// Option configures the server
type Option func(*serverOptions)

// WithCache configures the server to use the cache c
func WithCache(c types.Cache) Option {
	return func(so *serverOptions) {
		so.ca = c
	}
}

// WithRunner returns an  Option that sets the runner to r
func WithRunner(r runner.Runner) Option {
	return func(so *serverOptions) {
		so.run = r
	}
}

// WithRTStore returns an Option that sets the RTStore to rts
func WithRTStore(rts RTStore) Option {
	return func(so *serverOptions) {
		so.rts = rts
	}
}

// WithVPSource returns an Option that sets the VPSource to vps
func WithVPSource(vps vpservice.VPSource) Option {
	return func(so *serverOptions) {
		so.vps = vps
	}
}

// WithAdjacencySource returns an Option that sets the AdjacencySource to as
func WithAdjacencySource(as types.AdjacencySource) Option {
	return func(so *serverOptions) {
		so.as = as
	}
}

// WithClusterSource returns an Option that sets the ClusterSource to cs
func WithClusterSource(cs types.ClusterSource) Option {
	return func(so *serverOptions) {
		so.cs = cs
	}
}

// WithRootCA returns an Option that sets the rootCA to rootCA
func WithRootCA(rootCA string) Option {
	return func(so *serverOptions) {
		so.rootCA = rootCA
	}
}

// WithCertFile returns an Option that sets the certFile to certFile
func WithCertFile(certFile string) Option {
	return func(so *serverOptions) {
		so.certFile = certFile
	}
}

// WithKeyFile returns an Option that sets the keyFile to keyFile
func WithKeyFile(keyFile string) Option {
	return func(so *serverOptions) {
		so.keyFile = keyFile
	}
}

// NewRevtrServer creates an new Server with the given options
func NewRevtrServer(opts ...Option) RevtrServer {
	var serv revtrServer
	for _, opt := range opts {
		opt(&serv.opts)
	}
	serv.nextID = new(uint32)
	s, err := connectToServices(serv.opts.rootCA)
	if err != nil {
		log.Fatalf("Could not connect to services: %v", err)
	}
	serv.s = s
	serv.vps = serv.opts.vps
	serv.cs = serv.opts.cs
	serv.rts = serv.opts.rts
	serv.as = serv.opts.as
	serv.run = serv.opts.run
	serv.cm = clustermap.New(serv.opts.cs, serv.opts.ca)
	serv.ca = serv.opts.ca
	serv.mu = &sync.Mutex{}
	serv.revtrs = make(map[uint32]revtrOutput)
	serv.running = make(map[uint32]<-chan *reversetraceroute.ReverseTraceroute)
	return serv
}

type revtrServer struct {
	rts    RTStore
	vps    vpservice.VPSource
	as     types.AdjacencySource
	cs     types.ClusterSource
	conf   types.Config
	run    runner.Runner
	opts   serverOptions
	nextID *uint32
	s      services
	cm     clustermap.ClusterMap
	ca     types.Cache
	// mu protects the revtr maps
	mu      *sync.Mutex
	revtrs  map[uint32]revtrOutput
	running map[uint32]<-chan *reversetraceroute.ReverseTraceroute
}

func (rs revtrServer) getID() uint32 {
	return atomic.AddUint32(rs.nextID, 1)
}

func (rs revtrServer) RunRevtr(req *pb.RunRevtrReq) (*pb.RunRevtrResp, error) {
	user, err := rs.rts.GetUserByKey(req.Auth)
	if err != nil {
		return nil, err
	}
	vps, err := rs.vps.GetVPs()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	var reqToRun []*pb.RevtrMeasurement
	for _, r := range req.GetRevtrs() {
		src, dst, err := verifyAddrs(r.Src, r.Dst, vps.GetVps())
		if err != nil {
			return nil, err
		}
		r.Src = src
		r.Dst = dst
		reqToRun = append(reqToRun, r)
	}
	if len(reqToRun) == 0 {
		return nil, ErrNoRevtrsToRun
	}
	servs, err := connectToServices(rs.opts.rootCA)
	if err != nil {
		log.Error(err)
		return nil, ErrConnectFailed
	}
	reqToRun, batchID, err := rs.rts.CreateRevtrBatch(reqToRun, user.Key)
	if err == repo.ErrCannotAddRevtrBatch {
		log.Error(err)
		return nil, ErrFailedToCreateBatch
	}
	// run these guys
	go func() {
		defer logError(servs.Close)
		runningRevtrs.Add(float64(len(reqToRun)))
		var rtrs []*reversetraceroute.ReverseTraceroute
		for _, r := range reqToRun {
			rtrs = append(rtrs, reversetraceroute.CreateReverseTraceroute(
				*r,
				rs.cs,
				nil,
				nil,
				nil,
			))
		}
		done := rs.run.Run(rtrs,
			runner.WithContext(context.Background()),
			runner.WithClient(servs.cl),
			runner.WithAtlas(servs.at),
			runner.WithVPSource(servs.vpserv),
			runner.WithAdjacencySource(rs.as),
			runner.WithClusterMap(rs.cm),
		)
		for drtr := range done {
			runningRevtrs.Sub(1)
			err = rs.rts.StoreBatchedRevtrs([]pb.ReverseTraceroute{drtr.ToStorable()})
			if err != nil {
				log.Errorf("Error storing Revtr(%d): %v", drtr.ID, err)

			}
		}
	}()
	return &pb.RunRevtrResp{
		BatchId: batchID,
	}, nil
}

func (rs revtrServer) GetRevtr(req *pb.GetRevtrReq) (*pb.GetRevtrResp, error) {
	usr, err := rs.rts.GetUserByKey(req.Auth)
	if err != nil {
		log.Error(err)
	}
	if err != nil {
		log.Error(err)
		return nil, err
	}
	if req.BatchId == 0 {
		return nil, BatchIDError{batchID: req.BatchId}
	}
	revtrs, err := rs.rts.GetRevtrsInBatch(usr.Id, req.BatchId)
	if err != nil {
		return nil, err
	}
	return &pb.GetRevtrResp{
		Revtrs: revtrs,
	}, nil
}

func (rs revtrServer) GetSources(req *pb.GetSourcesReq) (*pb.GetSourcesResp, error) {
	_, err := rs.rts.GetUserByKey(req.Auth)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	vps, err := rs.vps.GetVPs()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	sr := &pb.GetSourcesResp{}
	for _, vp := range vps.GetVps() {
		s := new(pb.Source)
		s.Hostname = vp.Hostname
		s.Ip, _ = util.Int32ToIPString(vp.Ip)
		s.Site = vp.Site
		sr.Srcs = append(sr.Srcs, s)
	}
	return sr, nil
}

type revtrOutput struct {
	rt *reversetraceroute.ReverseTraceroute
	oc chan Status
}

func (rs revtrServer) AddRevtr(rtm pb.RevtrMeasurement) (uint32, error) {
	vps, err := rs.vps.GetVPs()
	if err != nil {
		log.Error(err)
		return 0, err
	}
	src, dst, err := verifyAddrs(rtm.Src, rtm.Dst, vps.GetVps())
	if err != nil {
		return 0, err
	}
	rtm.Src = src
	rtm.Dst = dst
	rs.mu.Lock()
	defer rs.mu.Unlock()
	id := rs.getID()
	oc := make(chan Status, 5)
	pf := makePrintHTML(oc, rs)
	rt := reversetraceroute.CreateReverseTraceroute(rtm,
		rs.cs,
		reversetraceroute.OnAddFunc(pf),
		reversetraceroute.OnFailFunc(pf),
		reversetraceroute.OnReachFunc(pf))
	// print out the initial state
	pf(rt)
	rs.revtrs[id] = revtrOutput{rt: rt, oc: oc}
	return id, nil
}

// IDError is returned when StartRevtr is called with an invalid id
type IDError struct {
	id uint32
}

func (ide IDError) Error() string {
	return fmt.Sprintf("invalid id %d", ide.id)
}

// Status represents the current running state of a reverse traceroute
// it is use for the web interface. Something better is probably needed
type Status struct {
	Rep    string
	Status bool
	Error  string
}

func (rs revtrServer) StartRevtr(ctx context.Context, id uint32) (<-chan Status, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if rt, ok := rs.revtrs[id]; ok {
		go func() {
			if _, ok := rs.running[id]; ok {
				return
			}
			runningRevtrs.Add(1)
			rs.mu.Lock()
			rtrs := []*reversetraceroute.ReverseTraceroute{rt.rt}
			done := rs.run.Run(rtrs,
				runner.WithContext(context.Background()),
				runner.WithClient(rs.s.cl),
				runner.WithAtlas(rs.s.at),
				runner.WithVPSource(rs.s.vpserv),
				runner.WithAdjacencySource(rs.as),
				runner.WithClusterMap(rs.cm),
			)
			rs.running[id] = done
			rs.mu.Unlock()
			for drtr := range done {
				runningRevtrs.Sub(1)
				rs.mu.Lock()
				delete(rs.revtrs, id)
				delete(rs.running, id)
				// done, close the channel
				close(rt.oc)
				rs.mu.Unlock()
				err := rs.rts.StoreRevtr(drtr.ToStorable())
				if err != nil {
					log.Error(err)
				}
			}
		}()
		return rt.oc, nil
	}
	return nil, IDError{id: id}
}

func (rs revtrServer) resolveHostname(ip string) string {
	// TODO clean up error logging
	item, ok := rs.ca.Get(ip)
	// If it's a cache miss, look through the vps
	// if its found set it.
	if !ok {
		vps, err := rs.vps.GetVPs()
		if err != nil {
			log.Error(err)
			return ""
		}
		for _, vp := range vps.GetVps() {
			ips, _ := util.Int32ToIPString(vp.Ip)
			// found it, set the cache and return the hostname
			if ips == ip {
				rs.ca.Set(ip, vp.Hostname, time.Hour*4)
				return vp.Hostname
			}
		}
		// not found from vps try reverse lookup
		hns, err := net.LookupAddr(ip)
		if err != nil {
			log.Error(err)
			// since the lookup failed, just set it blank for now
			// after it expires we'll try again
			rs.ca.Set(ip, "", time.Hour*4)
			return ""
		}
		rs.ca.Set(ip, hns[0], time.Hour*4)
		return hns[0]
	}
	return item.(string)
}

func (rs revtrServer) getRTT(src, dst string) float32 {
	key := fmt.Sprintf("%s:%s:rtt", src, dst)
	item, ok := rs.ca.Get(key)
	if !ok {
		targ, _ := util.IPStringToInt32(dst)
		src, _ := util.IPStringToInt32(src)
		ping := &datamodel.PingMeasurement{
			Src:     src,
			Dst:     targ,
			Count:   "1",
			Timeout: 5,
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		st, err := rs.s.cl.Ping(ctx,
			&datamodel.PingArg{Pings: []*datamodel.PingMeasurement{ping}})
		if err != nil {
			log.Error(err)
			rs.ca.Set(key, float32(0), time.Minute*30)
			return 0
		}
		for {
			p, err := st.Recv()
			if err == io.EOF {
				break

			}
			if err != nil {
				log.Error(err)
				rs.ca.Set(key, float32(0), time.Minute*30)
				return 0

			}
			if len(p.Responses) == 0 {
				rs.ca.Set(key, float32(0), time.Minute*30)
				return 0
			}
			rs.ca.Set(key, float32(p.Responses[0].Rtt)/1000, time.Minute*30)
			return float32(p.Responses[0].Rtt) / 1000
		}
	}
	return item.(float32)
}

func makePrintHTML(sc chan<- Status, rs revtrServer) func(*reversetraceroute.ReverseTraceroute) {
	return func(rt *reversetraceroute.ReverseTraceroute) {
		//need to find hostnames and rtts
		hopsSeen := make(map[string]bool)
		var out bytes.Buffer
		out.WriteString(`<table class="table">`)
		out.WriteString(`<caption class="text-center">Reverse Traceroute from `)
		out.WriteString(fmt.Sprintf("%s (%s) back to ", rt.Dst, rs.resolveHostname(rt.Dst)))
		out.WriteString(rt.Src)
		out.WriteString(fmt.Sprintf(" (%s)", rs.resolveHostname(rt.Src)))
		out.WriteString("</caption>")
		out.WriteString(`<tbody>`)
		first := new(bool)
		var i int
		if len(*rt.Paths) > 0 {

			for _, segment := range *rt.CurrPath().Path {
				*first = true
				symbol := new(string)
				switch segment.(type) {
				case *reversetraceroute.DstSymRevSegment:
					*symbol = "sym"
				case *reversetraceroute.DstRevSegment:
					*symbol = "dst"
				case *reversetraceroute.TRtoSrcRevSegment:
					*symbol = "tr"
				case *reversetraceroute.SpoofRRRevSegment:
					*symbol = "rr"
				case *reversetraceroute.RRRevSegment:
					*symbol = "rr"
				case *reversetraceroute.SpoofTSAdjRevSegmentTSZeroDoubleStamp:
					*symbol = "ts"
				case *reversetraceroute.SpoofTSAdjRevSegmentTSZero:
					*symbol = "ts"
				case *reversetraceroute.SpoofTSAdjRevSegment:
					*symbol = "ts"
				case *reversetraceroute.TSAdjRevSegment:
					*symbol = "ts"
				}
				for _, hop := range segment.Hops() {
					if hopsSeen[hop] {
						continue

					}
					hopsSeen[hop] = true
					tech := new(string)
					if *first {
						*tech = *symbol
						*first = false

					} else {
						*tech = "-" + *symbol

					}
					if hop == "0.0.0.0" || hop == "*" {
						out.WriteString(fmt.Sprintf("<tr><td>%-2d</td><td>%-80s</td><td></td><td>%s</td></tr>", i, "* * *", *tech))

					} else {
						out.WriteString(fmt.Sprintf("<tr><td>%-2d</td><td>%-80s (%s)</td><td>%.3fms</td><td>%s</td></tr>", i, hop, rs.resolveHostname(hop), rs.getRTT(rt.Src, hop), *tech))

					}
					i++

				}

			}

		}
		out.WriteString("</tbody></table>")
		out.WriteString(fmt.Sprintf("\n%s", rt.StopReason))
		var showError = rt.StopReason == reversetraceroute.Failed
		var errorText string
		if showError {
			errorText = strings.Replace(rt.ErrorDetails.String(), "\n", "<br>", -1)

		}
		var stat Status
		stat.Rep = strings.Replace(out.String(), "\n", "<br>", -1)
		stat.Status = rt.StopReason != ""
		stat.Error = errorText
		select {
		case sc <- stat:
		default:
		}
	}
}

type services struct {
	cl     client.Client
	clc    *grpc.ClientConn
	at     at.Atlas
	atc    *grpc.ClientConn
	vpserv vpservice.VPSource
	vpsc   *grpc.ClientConn
}

func (s services) Close() error {
	var err error
	if s.clc != nil {
		err = s.clc.Close()
	}
	if s.atc != nil {
		err = s.atc.Close()
	}
	if s.vpsc != nil {
		err = s.vpsc.Close()
	}
	return err
}

func connectToServices(rootCA string) (services, error) {
	var ret services
	_, srvs, err := net.LookupSRV("controller", "tcp", "revtr.ccs.neu.edu")
	if err != nil {
		return ret, err
	}
	ccreds, err := credentials.NewClientTLSFromFile(rootCA, srvs[0].Target)
	if err != nil {
		return ret, err
	}
	connstr := fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port)
	cc, err := grpc.Dial(connstr, grpc.WithTransportCredentials(ccreds))
	if err != nil {
		return ret, err
	}
	cli := client.New(context.Background(), cc)
	_, srvs, err = net.LookupSRV("atlas", "tcp", "revtr.ccs.neu.edu")
	if err != nil {
		logError(cc.Close)
		return ret, err
	}
	atcreds, err := credentials.NewClientTLSFromFile(rootCA, srvs[0].Target)
	if err != nil {
		logError(cc.Close)
		return ret, err
	}
	connstrat := fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port)
	c2, err := grpc.Dial(connstrat, grpc.WithTransportCredentials(atcreds))
	if err != nil {
		logError(cc.Close)
		return ret, err
	}
	atl := at.New(context.Background(), c2)
	_, srvs, err = net.LookupSRV("vpservice", "tcp", "revtr.ccs.neu.edu")
	if err != nil {
		return ret, err
	}
	vpcreds, err := credentials.NewClientTLSFromFile(rootCA, srvs[0].Target)
	if err != nil {
		logError(cc.Close)
		logError(c2.Close)
		return ret, err
	}
	connvp := fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port)
	c3, err := grpc.Dial(connvp, grpc.WithTransportCredentials(vpcreds))
	if err != nil {
		logError(cc.Close)
		logError(c2.Close)
		return ret, err
	}
	vps := vpservice.New(context.Background(), c3)

	ret.cl = cli
	ret.clc = cc
	ret.at = atl
	ret.atc = c2
	ret.vpserv = vps
	ret.vpsc = c3
	return ret, nil
}
