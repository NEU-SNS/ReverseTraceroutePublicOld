package atlas

import (
	"fmt"
	"io"
	"net"
	"time"

	"google.golang.org/grpc"

	cclient "github.com/NEU-SNS/ReverseTraceroute/controller/client"
	"github.com/NEU-SNS/ReverseTraceroute/dataaccess"
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice/client"
	"golang.org/x/net/context"
)

// Atlas is the atlas
type Atlas struct {
	da    *dataaccess.DataAccess
	donec chan struct{}
}

// GetIntersectingPath satisfies the server interface
func (a *Atlas) GetIntersectingPath(ctx context.Context, ir *dm.IntersectionRequest) ([]*dm.IntersectionResponse, error) {
	in := make(chan *dm.IntersectionResponse)
	go func() {
		log.Debug("Looing for intersect for ", ir)
		req := []dm.SrcDst{
			dm.SrcDst{
				Src: ir.Address,
				Dst: ir.Dest,
			},
		}
		res, err := a.da.FindIntersectingTraceroute(req, ir.UseAliases, time.Duration(ir.Staleness))
		log.Debug("FindIntersectingTraceroute resp ", res)
		if err != nil {
			log.Error(err)
			in <- &dm.IntersectionResponse{
				Type:  dm.IResponseType_ERROR,
				Error: err.Error(),
			}
			close(in)
			return
		}
		if len(res) == 0 {
			intr := &dm.IntersectionResponse{
				Type: dm.IResponseType_NONE_FOUND,
			}
			in <- intr
			close(in)
			return
		}
		for _, resp := range res {
			intr := &dm.IntersectionResponse{
				Type: dm.IResponseType_PATH,
				Path: resp,
			}
			in <- intr
		}
		close(in)
		return
	}()
	var ret []*dm.IntersectionResponse
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("Find IntersectionRequest timed out")
		case ir, ok := <-in:
			if !ok {
				return ret, nil
			}
			ret = append(ret, ir)
		}
	}
}

// GetPathsWithToken satisfies the server interface
func (a *Atlas) GetPathsWithToken(ctx context.Context, in <-chan *dm.TokenRequest) (<-chan *dm.TokenResponse, error) {
	return nil, nil
}

func (a *Atlas) runTraces(vp, con *net.SRV) error {
	connstr := fmt.Sprintf("%s:%d", vp.Target, vp.Port)
	cc, err := grpc.Dial(connstr, grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer cc.Close()
	vpc := client.New(context.Background(), cc)
	vpr, err := vpc.GetVPs()
	if err != nil {
		return err
	}
	vps := vpr.GetVps()
	set := make(map[string]*dm.VantagePoint)
	for _, vp := range vps {
		set[vp.Site] = vp
	}
	var meas []*dm.TracerouteMeasurement
	for _, vp := range set {
		for _, dst := range vps {
			curr := &dm.TracerouteMeasurement{
				Src:     vp.Ip,
				Dst:     dst.Ip,
				Timeout: 60 * 3,
			}
			meas = append(meas, curr)
		}
	}
	conc, err := grpc.Dial(fmt.Sprintf("%s:%d", con.Target, con.Port), grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conc.Close()
	ccl := cclient.New(context.Background(), conc)

	total := len(meas)
	times := total/100 + 2
	var curr int
	for i := 1; i < times; i++ {
		end := i * 2
		if i*2 > total {
			end = total
		} else {
			end = i * 2
		}
		st, err := ccl.Traceroute(&dm.TracerouteArg{
			Traceroutes: meas[curr:end],
		})
		curr = end
		if err != nil {
			return err
		}
		for {
			tr, err := st.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			go func() {
				err := a.da.StoreAtlasTraceroute(tr)
				if err != nil {
					log.Error(err)
				}
			}()
		}
	}
	return nil
}

func (a *Atlas) updateTraceroutes() {
	_, srvs, err := net.LookupSRV("vpservice", "tcp", "revtr.ccs.neu.edu")
	if err != nil {
		log.Error(err)
		return
	}
	srv := srvs[0]
	_, srvs, err = net.LookupSRV("controller", "tcp", "revtr.ccs.neu.edu")
	if err != nil {
		log.Error(err)
		return
	}
	tick := time.NewTicker(time.Minute * 10)
	for {
		select {
		case <-a.donec:
			return
		case <-tick.C:
			a.runTraces(srv, srvs[0])
		}
	}

}

// NewAtlasService creates a new Atlas
func NewAtlasService(da *dataaccess.DataAccess) *Atlas {
	ret := &Atlas{
		da: da,
	}
	go ret.updateTraceroutes()
	return ret
}
