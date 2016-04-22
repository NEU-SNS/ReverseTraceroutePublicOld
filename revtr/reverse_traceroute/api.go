package reversetraceroute

import (
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/context"

	"github.com/NEU-SNS/ReverseTraceroute/controller/client"
	"github.com/NEU-SNS/ReverseTraceroute/dataaccess"
	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/pb"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	vpservice "github.com/NEU-SNS/ReverseTraceroute/vpservice/client"
	"github.com/gorilla/websocket"
)

var (
	homeTemplate, _    = template.ParseFiles("webroot/templates/home.html")
	runningTemplate, _ = template.ParseFiles("webroot/templates/running.html")
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

type runningModel struct {
	Key string
	URL string
}

type revtrAndService struct {
	rt   *ReverseTraceroute
	serv *services
}

// RunRevtr handles /runrevtr
type RunRevtr struct {
	da         *dataaccess.DataAccess
	vps        vpservice.VPSource
	s          services
	Route      string
	keyToRevtr map[uint32]revtrAndService
	mu         *sync.Mutex // protects ipToRevtr
	next       *uint32
	rootCa     string
}

// NewRunRevtr creates a new RunRevtr
func NewRunRevtr(da *dataaccess.DataAccess, rootCa string) RunRevtr {
	s, err := connectToServices(rootCa)
	if err != nil {
		log.Error(err)
	}
	return RunRevtr{
		da:         da,
		s:          s,
		Route:      "/runrevtr",
		keyToRevtr: make(map[uint32]revtrAndService),
		mu:         &sync.Mutex{},
		rootCa:     rootCa,
		next:       new(uint32),
	}
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
	keyint, err := strconv.ParseUint(key, 10, 32)
	if err != nil {
		log.Error(err)
		log.Error("Invalid Key")
		http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	rtrs, ok := rr.keyToRevtr[uint32(keyint)]
	rtr := rtrs.rt
	if !ok {
		defer ws.Close()
		log.Error("Invalid Key")
		http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	rtr.ws = append(rtr.ws, c)
	go func() {
		rtr.print = true
		if !rtr.isRunning() {
			runningRevtrs.Add(1)
			rtr.output()
			err := rtr.run()
			if err != nil {
				log.Error(err)
			}
			rtr.output()
		} else {
			rtr.output()
			return
		}
		runningRevtrs.Sub(1)
		defer rtr.ws.Close()
		defer rtrs.serv.Close()
		rr.mu.Lock()
		defer rr.mu.Unlock()
		delete(rr.keyToRevtr, uint32(keyint))
		log.Debug("Revtrs running: ", rr.keyToRevtr)
		err = rtr.ws.Close()
		if err != nil {
			log.Error(err)
		}
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	vps, err := rr.s.cl.GetVps(ctx, &datamodel.VPRequest{})
	if err != nil {
		log.Error(err)
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	dst, valid := validDest(dst, vps.GetVps())
	if !valid {
		log.Debug("Invalid destination ", dst)
		http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	src, valid = validSrc(src, vps.GetVps())
	if !valid {
		log.Debug("Invalid src: ", src)
		http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	key := atomic.AddUint32(rr.next, 1)
	log.Debug("New key: ", key)
	var rt runningModel
	rt.Key = fmt.Sprintf("%d", key)
	rt.URL = req.Host
	// Split should be the actual ip of the client now
	rr.mu.Lock()
	defer rr.mu.Unlock()
	if _, ok := rr.keyToRevtr[key]; !ok {
		serv, err := connectToServices(rr.rootCa)
		if err != nil {
			log.Error(err)
			http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		// At this point we need to run the revtr and redirect approprately
		log.Debug("Running RTR with src: ", src, " ", "dst: ", dst)
		rt := CreateReverseTraceroute(pb.RevtrMeasurement{
			Src:       src,
			Dst:       dst,
			Staleness: 60,
		}, false, true, serv.cl, serv.at, serv.vpserv, rr.da, rr.da)
		rr.keyToRevtr[key] = revtrAndService{rt: rt, serv: &serv}
	}
	runningTemplate.Execute(rw, &rt)
	return
}
