package revtr

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	at "github.com/NEU-SNS/ReverseTraceroute/atlas/client"
	"github.com/NEU-SNS/ReverseTraceroute/controller/client"
	"github.com/NEU-SNS/ReverseTraceroute/dataaccess"
	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	vpservice "github.com/NEU-SNS/ReverseTraceroute/vpservice/client"
	"github.com/golang/protobuf/jsonpb"
	"github.com/gorilla/websocket"
)

// V1Revtr is the api endpoint for interacting with revtrs
type V1Revtr struct {
	da    *dataaccess.DataAccess
	Route string
}

// NewV1Revtr creates a new V1Revtr
func NewV1Revtr(da *dataaccess.DataAccess) V1Revtr {
	return V1Revtr{
		da:    da,
		Route: "/api/v1/revtr",
	}
}

type runningModel struct {
	Key string
	URL string
}

// RunRevtr handles /runrevtr
type RunRevtr struct {
	da        *dataaccess.DataAccess
	s         services
	Route     string
	ipToRevtr map[string]*ReverseTraceroute
	mu        *sync.Mutex // protects ipToRevtr
}

// NewRunRevtr creates a new RunRevtr
func NewRunRevtr(da *dataaccess.DataAccess) RunRevtr {
	s, err := connectToServices()
	if err != nil {
		log.Error(err)
	}
	return RunRevtr{
		da:        da,
		s:         s,
		Route:     "/runrevtr",
		ipToRevtr: make(map[string]*ReverseTraceroute),
		mu:        &sync.Mutex{},
	}
}

func validSrc(src string, vps []*datamodel.VantagePoint) (string, bool) {
	for _, vp := range vps {
		if vp.Hostname == src {
			s, _ := util.Int32ToIPString(vp.Ip)
			return s, true
		}
	}
	return "", false
}

func validDest(dst string, vps []*datamodel.VantagePoint) (string, bool) {
	var notIP bool
	ip := net.ParseIP(dst)
	if ip == nil {
		notIP = true
	}
	if notIP {
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
	return dst, true
}

// WS is the endpoint for websockets
func (rr RunRevtr) WS(rw http.ResponseWriter, req *http.Request) {
	var upgrader websocket.Upgrader
	ws, err := upgrader.Upgrade(rw, req, nil)
	if err != nil {
		log.Error(err)
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	key := req.URL.Query().Get("key")
	log.Debug("WS request for key: ", key)
	if key == "" {
		defer ws.Close()
		err = ws.WriteMessage(websocket.TextMessage, []byte("Missing key."))
		if err != nil {
			http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}
	c := wsConnection{c: ws}
	rr.mu.Lock()
	defer rr.mu.Unlock()
	rtr, ok := rr.ipToRevtr[key]
	if !ok {
		defer ws.Close()
		log.Error("Invalid Key")
		http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	rtr.ws = append(rtr.ws, c)
	go func() {
		defer rtr.ws.Close()
		rtr.print = true
		if !rtr.isRunning() {
			err := rtr.run()
			if err != nil {
				log.Error(err)
				rtr.output()
			}
		} else {
			rtr.output()
			return
		}
		rr.mu.Lock()
		defer rr.mu.Unlock()
		delete(rr.ipToRevtr, key)
		rr.da.StoreRevtr(rtr.ToStorable())
	}()

}

// RunRevtr handles /runrevtr
func (rr RunRevtr) RunRevtr(rw http.ResponseWriter, req *http.Request) {
	log.Debug("RunRevtr")

	if req.Method != http.MethodGet {
		http.Error(rw, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	src := req.FormValue("src")
	dst := req.FormValue("dst")
	if src == "" || dst == "" {
		http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		log.Errorf("bad request, src: %s, dst: %s", src, dst)
		return
	}
	vps, err := rr.s.cl.GetVps(&datamodel.VPRequest{})
	if err != nil {
		log.Error(err)
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	dst, valid := validDest(dst, vps.GetVps())
	if !valid {
		http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	src, valid = validSrc(src, vps.GetVps())
	if !valid {
		http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	reqIP := req.RemoteAddr
	head := req.Header.Get("X-Forwarded-For")
	if head == "" {
		head = req.Header.Get("x-forwarded-for")
	}
	if head == "" {
		head = req.Header.Get("X-FORWARDED-FOR")
	}
	if head == "" {
		head = reqIP
	}
	headSplit := strings.Split(head, ",")
	if len(headSplit) > 0 {
		head = headSplit[0]
	}
	if strings.Contains(head, ":") {
		head, _, err = net.SplitHostPort(head)
		if err != nil {
			log.Error(err)
			http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}
	var rt runningModel
	rt.Key = head
	rt.URL = req.Host
	// Split should be the actual ip of the client now
	rr.mu.Lock()
	defer rr.mu.Unlock()
	if _, ok := rr.ipToRevtr[head]; !ok {
		serv, err := connectToServices()
		if err != nil {
			log.Error(err)
			http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		// At this point we need to run the revtr and redirect approprately
		log.Debug("Running RTR with src: ", src, " ", "dst: ", dst)
		rt := CreateReverseTraceroute(datamodel.RevtrMeasurement{
			Src: src,
			Dst: dst,
		}, true, true, serv.cl, serv.at, serv.vpserv, rr.da, rr.da)
		rr.ipToRevtr[head] = rt
	}
	runningTemplate.Execute(rw, &rt)
	return
}

var (
	homeTemplate    = template.Must(template.ParseFiles("webroot/templates/home.html"))
	runningTemplate = template.Must(template.ParseFiles("webroot/templates/running.html"))
)

type homeModel struct {
	Nodes []vpModel
}

type vpModel struct {
	Host string
	IP   string
}

type vpModelSort []vpModel

func (a vpModelSort) Len() int           { return len(a) }
func (a vpModelSort) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a vpModelSort) Less(i, j int) bool { return a[i].Host < a[j].Host }

// Home handles the home route for revtr
type Home struct {
	da    *dataaccess.DataAccess
	cl    client.Client
	vps   vpservice.VPSource
	Route string
}

// NewHome creates a new home
func NewHome(da *dataaccess.DataAccess, cl client.Client, vps vpservice.VPSource) Home {
	return Home{
		da:    da,
		cl:    cl,
		vps:   vps,
		Route: "/",
	}
}

// Home handles the "/" route
func (h Home) Home(rw http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		http.Error(rw, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	vps, err := h.vps.GetVPs()
	if err != nil {
		log.Error(err)
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusBadRequest)
		return
	}
	var model homeModel
	nodes := vps.GetVps()
	sites := make(map[string]bool)
	var vpl []vpModel
	for _, node := range nodes {
		if !node.RecordRoute || !node.Timestamp {
			continue
		}
		if sites[node.Site] {
			continue
		}
		sites[node.Site] = true
		var vp vpModel
		vp.Host = node.Hostname
		vp.IP, err = util.Int32ToIPString(node.Ip)
		if err != nil {
			log.Error(err)
			http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusBadRequest)
			return
		}
		vpl = append(vpl, vp)
	}
	sort.Sort(vpModelSort(vpl))
	model.Nodes = vpl
	homeTemplate.Execute(rw, &model)
}

const (
	keyHeader = "Revtr-Key"
	errorPage = `An error has occurred: %s.
	Please send a copy of this error to revtr@ccs.neu.edu`
)

// Handle handles all methods for the route of V1Revtr
func (s V1Revtr) Handle(rw http.ResponseWriter, req *http.Request) {
	log.Debug(req.Host)
	key := req.Header.Get(keyHeader)
	if key == "" {
		http.Error(rw, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	user, err := s.da.GetUserByKey(key)
	if err == dataaccess.ErrNoRevtrUserFound {
		http.Error(rw, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	if err != nil {
		log.Error(err)
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	// This would check the db for credentials found in the request
	// If I get here I should be authorized
	switch req.Method {
	case http.MethodPost:
		s.submitRevtr(rw, req, user)
	case http.MethodGet:
		s.retreiveRevtr(rw, req, user)
	default:
		http.Error(rw, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}

func (s V1Revtr) retreiveRevtr(rw http.ResponseWriter, req *http.Request, user datamodel.RevtrUser) {
	ids := req.URL.Query().Get("revtrid")
	if len(ids) == 0 {
		http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseUint(ids, 10, 32)
	if err != nil {
		log.Error(err)
		http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	revtrs, err := s.da.GetRevtrsInBatch(user.ID, uint32(id))
	if err != nil {
		log.Error(err)
		http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	ret := &datamodel.RevtrResponse{}
	ret.Revtrs = revtrs
	var m jsonpb.Marshaler
	err = m.Marshal(rw, ret)
	if err != nil {
		log.Debug(err)
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusBadRequest)
		return
	}
	rw.Header().Set("Content-Type", "application/json")
	return
}

func (s V1Revtr) submitRevtr(rw http.ResponseWriter, req *http.Request, user datamodel.RevtrUser) {
	var revtrr datamodel.RevtrRequest
	err := jsonpb.Unmarshal(req.Body, &revtrr)
	if err != nil {
		http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		fmt.Fprintf(rw, err.Error())
		return
	}
	rs := revtrr.GetRevtrs()
	var reqToRun []datamodel.RevtrMeasurement
	for _, r := range rs {
		// Doing this conversion just ensures they're valid ips
		_, err := util.IPStringToInt32(r.Src)
		_, err = util.IPStringToInt32(r.Dst)
		if err != nil {
			http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			fmt.Fprintf(rw, err.Error())
			return
		}
		reqToRun = append(reqToRun, *r)
	}
	// Connect to all the necessary services before trying to run anything
	servs, err := connectToServices()
	if err != nil {
		// Failed to connect to a service
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(rw, errorPage, err)
		return
	}
	// At this point the request is valid, i've made all my connections
	// and i'm loaded up with revtrs to run.
	// I need to check if the api key that came in with the request can run more
	// if not, give them a bad request and a message telling them why.
	reqToRun, batchID, err := s.da.CreateRevtrBatch(reqToRun, user.Key)
	if err == dataaccess.ErrCannotAddRevtrBatch {
		fmt.Fprintf(rw, "There was an error when trying to run your reverse traceroutes.\n")
		fmt.Fprintf(rw, "Please try again later.\n")
		return
	}
	// I've gotten my id, i've added all the state to the DB that I need
	// its time now to start the revtrs running and return the proper results
	go func() {
		defer servs.Close()
		var rtrdone []*ReverseTraceroute
		var rtrsave []datamodel.ReverseTraceroute
		var mu sync.Mutex
		var wg sync.WaitGroup
		for _, rtr := range reqToRun {
			wg.Add(1)
			go func(r datamodel.RevtrMeasurement) {
				defer wg.Done()
				res, err := RunReverseTraceroute(r, true, servs.cl, servs.at, servs.vpserv, s.da, s.da)
				if err != nil {
					log.Errorf("Error running Revtr(%d): %v", res.ID, err)
				}
				mu.Lock()
				log.Debug(res)
				rtrdone = append(rtrdone, res)
				rtrsave = append(rtrsave, res.ToStorable())
				mu.Unlock()
			}(rtr)
		}
		wg.Wait()
		log.Debug("Storing: ", rtrsave)
		err = s.da.StoreBatchedRevtrs(rtrsave)
		if err != nil {
			log.Error(err)
		}
	}()
	rw.WriteHeader(http.StatusOK)
	rw.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(rw).Encode(struct {
		ResultURI string `json:"result_uri"`
	}{
		ResultURI: fmt.Sprintf("%s%s?revtrid=%d", req.Host, s.Route, batchID),
	})
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
	// The connections should never be nil because I only
	// set them after all are successfully connected
	// but i'm checking anyway
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

func connectToServices() (services, error) {
	var ret services
	_, srvs, err := net.LookupSRV("controller", "tcp", "revtr.ccs.neu.edu")
	if err != nil {
		return ret, err
	}
	connstr := fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port)
	cc, err := grpc.Dial(connstr, grpc.WithInsecure())
	if err != nil {
		return ret, err
	}
	cli := client.New(context.Background(), cc)
	_, srvs, err = net.LookupSRV("atlas", "tcp", "revtr.ccs.neu.edu")
	if err != nil {
		cc.Close()
		return ret, err
	}
	connstrat := fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port)
	c2, err := grpc.Dial(connstrat, grpc.WithInsecure())
	if err != nil {
		cc.Close()
		return ret, err
	}
	atl := at.New(context.Background(), c2)
	_, srvs, err = net.LookupSRV("vpservice", "tcp", "revtr.ccs.neu.edu")
	if err != nil {
		return ret, err
	}
	connvp := fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port)
	c3, err := grpc.Dial(connvp, grpc.WithInsecure())
	if err != nil {
		cc.Close()
		c2.Close()
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
