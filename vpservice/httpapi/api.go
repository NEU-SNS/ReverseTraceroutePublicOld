package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice/server"
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

func (api API) quarantineVPS(r http.ResponseWriter, req *http.Request) {
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
