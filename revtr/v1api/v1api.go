package v1api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/pb"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/repository"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/server"
	"github.com/gogo/protobuf/jsonpb"
)

const (
	v1Prefix  = "/api/v1/"
	keyHeader = "Revtr-Key"
)

// V1Api is
type V1Api struct {
	s   server.RevtrServer
	mux *http.ServeMux
}

// NewV1Api creates a V1Api using RevtrServer s registering routes on ServeMux mux
func NewV1Api(s server.RevtrServer, mux *http.ServeMux) V1Api {
	api := V1Api{s: s, mux: mux}
	mux.HandleFunc(v1Prefix+"sources", api.sources)
	mux.HandleFunc(v1Prefix+"revtr", api.revtr)
	return api
}

func (v1 V1Api) sources(r http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(r, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	key := req.Header.Get(keyHeader)
	pbr := &pb.GetSourcesReq{
		Auth: key,
	}
	resp, err := v1.s.GetSources(pbr)
	if err != repo.ErrNoRevtrUserFound {
		http.Error(r, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	if err != nil {
		log.Error(err)
		http.Error(r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(r).Encode(resp)
	if err != nil {
		log.Error(err)
		http.Error(r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	r.Header().Set("Content-Type", "application/json")
}

func (v1 V1Api) revtr(r http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		v1.submitRevtr(r, req)
	case http.MethodGet:
		v1.retreiveRevtr(r, req)
	default:
		http.Error(r, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

func (v1 V1Api) retreiveRevtr(r http.ResponseWriter, req *http.Request) {
	key := req.Header.Get(keyHeader)
	ids := req.URL.Query().Get("batchid")
	if len(ids) == 0 {
		http.Error(r, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseUint(ids, 10, 32)
	if err != nil {
		log.Error(err)
		http.Error(r, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	grr := &pb.GetRevtrReq{
		BatchId: uint32(id),
		Auth:    key,
	}
	revtrs, err := v1.s.GetRevtr(grr)
	if err != repo.ErrNoRevtrUserFound {
		http.Error(r, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	if err != nil {
		log.Error(err)
		http.Error(r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	var m jsonpb.Marshaler
	err = m.Marshal(r, revtrs)
	if err != nil {
		log.Debug(err)
		http.Error(r, http.StatusText(http.StatusInternalServerError), http.StatusBadRequest)
		return
	}
	r.Header().Set("Content-Type", "application/json")
}

func (v1 V1Api) submitRevtr(r http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(r, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	key := req.Header.Get(keyHeader)
	var revtr pb.RunRevtrReq
	err := jsonpb.Unmarshal(req.Body, &revtr)
	if err != nil {
		log.Error(err)
		http.Error(r, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	revtr.Auth = key
	resp, err := v1.s.RunRevtr(&revtr)
	if err != nil {
		log.Error(err)
		http.Error(r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(r).Encode(struct {
		ResultURI string `json:"result_uri"`
	}{
		ResultURI: fmt.Sprintf("https://%s%s?batchid=%d", req.Host, v1Prefix+"revtr", resp.BatchId),
	})
	r.Header().Set("Content-Type", "application/json")
}
