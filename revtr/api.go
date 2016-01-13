package revtr

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	at "github.com/NEU-SNS/ReverseTraceroute/atlas/client"
	"github.com/NEU-SNS/ReverseTraceroute/controller/client"
	"github.com/NEU-SNS/ReverseTraceroute/dataaccess"
	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	vpservice "github.com/NEU-SNS/ReverseTraceroute/vpservice/client"
	"github.com/prometheus/log"
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

const (
	keyHeader = "Revtr-Key"
	errorPage = `An error has occurred: %s.
	Please send a copy of this error to revtr@ccs.neu.edu`
)

// Handle handles all methods for the route of V1Revtr
func (s V1Revtr) Handle(rw http.ResponseWriter, req *http.Request) {
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
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte("Hello there!"))
	default:
		http.Error(rw, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}

func (s V1Revtr) submitRevtr(rw http.ResponseWriter, req *http.Request, user datamodel.RevtrUser) {
	var revtrr datamodel.RevtrRequest
	dec := json.NewDecoder(req.Body)
	if err := dec.Decode(&revtrr); err == io.EOF {
		http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		fmt.Fprintf(rw, err.Error())
	} else if err != nil {
		http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		fmt.Fprintf(rw, err.Error())
		return
	}
	rs := revtrr.GetRevtrs()
	var reqToRun []ReverseTracerouteReq
	for _, r := range rs {
		src, err := util.IPStringToInt32(r.Src)
		dst, err := util.IPStringToInt32(r.Dst)
		if err != nil {
			http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			fmt.Fprintf(rw, err.Error())
			return
		}
		reqToRun = append(reqToRun, ReverseTracerouteReq{Src: src, Dst: dst, Staleness: r.Staleness})
	}
	// At this point the request is valid and i'm loaded up with revtrs to run.
	// I need to check if the api key that came in with the request can run more
	// if not, give them a bad request and a message telling them why.

	// If i'm all good to run, connect to the system services
	servs, err := connectToServices()
	if err != nil {
		// Failed to connect to a service
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(rw, errorPage, err)
		return
	}
	// Now that i'm all connected, I need to create the neccessary state in the db to track these revtrs.
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte("All Good"))
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
