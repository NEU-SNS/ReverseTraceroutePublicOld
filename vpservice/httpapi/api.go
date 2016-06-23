package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice/pb"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice/server"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice/types"
)

const (
	v1Prefix   = "/api/v1/"
	vpLabelKey = "vp"
)

// API is the http api of the vpservice
type API struct {
	s   server.VPServer
	mux *http.ServeMux
}

// NewAPI creates a new API using the given server and mux
func NewAPI(s server.VPServer, mux *http.ServeMux) API {
	api := API{s: s, mux: mux}
	mux.HandleFunc(v1Prefix+"quarantine", api.quarantineVPS)
	mux.HandleFunc(v1Prefix+"unquarantine", api.unquarantineVPS)
	mux.HandleFunc(v1Prefix+"quarantinealert", api.quarantineAlertVPS)
	return api
}

type alert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
}

type quarantine struct {
	Version  string  `json:"version"`
	Status   string  `json:"status"`
	Receiver string  `json:"receiver"`
	Alerts   []alert `json:"alerts"`
}

type manualQuarantine struct {
	Hostname string    `json:"hostname"`
	Expire   time.Time `json:"expire"`
}

type manualQuarantines struct {
	Quarantines []manualQuarantine `json:"quarantines"`
}

func (api API) quarantineVPS(r http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(r, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	var mq manualQuarantines
	if err := json.NewDecoder(req.Body).Decode(&mq); err != nil {
		log.Error(err)
		http.Error(r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	vps, err := api.s.GetVPs(&pb.VPRequest{})
	if err != nil {
		log.Error(err)
		http.Error(r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	vpm := make(map[string]*pb.VantagePoint)
	for _, vp := range vps.GetVps() {
		vpm[vp.Hostname] = vp
	}
	var quars []types.Quarantine
	for _, q := range mq.Quarantines {
		if vp, ok := vpm[q.Hostname]; ok {
			quars = append(quars, types.NewManualQuarantine(*vp, q.Expire))
		}
	}
	if err := api.s.QuarantineVPs(quars); err != nil {
		log.Error(err)
		http.Error(r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (api API) quarantineAlertVPS(r http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(r, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	var quar quarantine
	if err := json.NewDecoder(req.Body).Decode(&quar); err != nil {
		log.Error(err)
		http.Error(r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	// do the simplest thing, get a list of all vps from the alerts, and try to quarentine them
	// ignore resolved
	if quar.Status == "resolved" {
		return
	}
	var vps []string
	for _, al := range quar.Alerts {
		if al.Status == "resolved" {
			continue
		}
		vps = append(vps, al.Labels[vpLabelKey])
	}
	err := api.s.QuarantineVPs(vps)
	if err != nil {
		log.Error(err)
		http.Error(r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

type unquarantine struct {
	VPS []string `json:"vps"`
}

func (api API) unquarantineVPS(r http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(r, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	var unq unquarantine
	if err := json.NewDecoder(req.Body).Decode(&unq); err != nil {
		log.Error(err)
		http.Error(r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	err := api.s.UnquarantineVPs(unq.VPS)
	if err != nil {
		log.Error(err)
		http.Error(r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
