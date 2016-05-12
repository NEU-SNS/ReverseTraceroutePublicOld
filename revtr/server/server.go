package server

import (
	"fmt"
	"net"
	"os"
	"sync"
	"sync/atomic"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	at "github.com/NEU-SNS/ReverseTraceroute/atlas/client"
	"github.com/NEU-SNS/ReverseTraceroute/controller/client"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/ip_utils"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/pb"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/repository"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/reverse_traceroute"
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
	StartRevtr(uint32) (<-chan reversetraceroute.Status, error)
}

type serverOptions struct {
	rts                       RTStore
	vps                       vpservice.VPSource
	as                        types.AdjacencySource
	cs                        types.ClusterSource
	rootCA, certFile, keyFile string
}

// Option configures the server
type Option func(*serverOptions)

// WithRTStore returns a Option that sets the RTStore to rts
func WithRTStore(rts RTStore) Option {
	return func(so *serverOptions) {
		so.rts = rts
	}
}

// WithVPSource returns a Option that sets the VPSource to vps
func WithVPSource(vps vpservice.VPSource) Option {
	return func(so *serverOptions) {
		so.vps = vps
	}
}

// WithAdjacencySource returns a Option that sets the AdjacencySource to as
func WithAdjacencySource(as types.AdjacencySource) Option {
	return func(so *serverOptions) {
		so.as = as
	}
}

// WithClusterSource returns a Option that sets the ClusterSource to cs
func WithClusterSource(cs types.ClusterSource) Option {
	return func(so *serverOptions) {
		so.cs = cs
	}
}

// WithRootCA returns a Option that sets the rootCA to rootCA
func WithRootCA(rootCA string) Option {
	return func(so *serverOptions) {
		so.rootCA = rootCA
	}
}

// WithCertFile returns a Option that sets the certFile to certFile
func WithCertFile(certFile string) Option {
	return func(so *serverOptions) {
		so.certFile = certFile
	}
}

// WithKeyFile returns a Option that sets the keyFile to keyFile
func WithKeyFile(keyFile string) Option {
	return func(so *serverOptions) {
		so.keyFile = keyFile
	}
}

// NewRevtrServer creates a new Server with the given options
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
	serv.mu = &sync.Mutex{}
	serv.revtrs = make(map[uint32]*reversetraceroute.ReverseTraceroute)
	return serv
}

type revtrServer struct {
	rts    RTStore
	vps    vpservice.VPSource
	as     types.AdjacencySource
	cs     types.ClusterSource
	conf   types.Config
	opts   serverOptions
	nextID *uint32
	s      services
	// mu protects the revtrs map
	mu     *sync.Mutex
	revtrs map[uint32]*reversetraceroute.ReverseTraceroute
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
		var wg sync.WaitGroup
		defer logError(servs.Close)
		wg.Add(len(reqToRun))
		for _, rtr := range reqToRun {
			runningRevtrs.Add(1)
			go func(r *pb.RevtrMeasurement) {
				defer runningRevtrs.Sub(1)
				defer wg.Done()
				if r.Staleness == 0 {
					r.Staleness = 60
				}
				res, err := reversetraceroute.RunReverseTraceroute(*r, servs.cl, servs.at, servs.vpserv, rs.as, rs.cs)
				if err != nil {
					log.Errorf("Error running Revtr(%d): %v", res.ID, err)
				}
				err = rs.rts.StoreBatchedRevtrs([]pb.ReverseTraceroute{res.ToStorable()})
				if err != nil {
					log.Errorf("Error storing Revtr(%d): %v", res.ID, err)
				}
			}(rtr)
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
		sr.Srcs = append(sr.Srcs, s)
	}
	return sr, nil
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
	rt := reversetraceroute.CreateReverseTraceroute(rtm,
		false, true, rs.s.cl, rs.s.at, rs.s.vpserv, rs.as, rs.cs)
	rs.revtrs[id] = rt
	return id, nil
}

// IDError is returned when StartRevtr is called with an invalid id
type IDError struct {
	id uint32
}

func (ide IDError) Error() string {
	return fmt.Sprintf("invalid id %d", ide.id)
}

func (rs revtrServer) StartRevtr(id uint32) (<-chan reversetraceroute.Status, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if rt, ok := rs.revtrs[id]; ok {
		go func() {
			if !rt.IsRunning() {
				runningRevtrs.Add(1)
				err := rt.Run()
				if err != nil {
					log.Error(err)
				}
			} else {
				return
			}
			runningRevtrs.Sub(1)
			rs.mu.Lock()
			delete(rs.revtrs, id)
			rs.mu.Unlock()
			err := rs.rts.StoreRevtr(rt.ToStorable())
			if err != nil {
				log.Error(err)
			}
		}()
		return rt.GetOutputChan(), nil
	}
	return nil, IDError{id: id}
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
