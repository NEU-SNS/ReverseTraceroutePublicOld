package atlas

import (
	"io"

	"golang.org/x/net/context"

	"github.com/NEU-SNS/ReverseTraceroute/atlas/pb"
	"github.com/NEU-SNS/ReverseTraceroute/atlas/server"
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/log"
)

// GRPCServ is a grpc service that satisfies the atlas interface
type GRPCServ struct {
	server.AtlasService
}

// GetIntersectingPath gets an intersecting path the the request
func (gs GRPCServ) GetIntersectingPath(stream pb.Atlas_GetIntersectingPathServer) error {
	in := make(chan *dm.IntersectionRequest)
	ec := make(chan error)
	inc := make(chan *dm.IntersectionRequest)
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()
	rets, err := gs.AtlasService.GetIntersectingPath(ctx, in)
	if err != nil {
		log.Error(err)
		return err
	}
	go func() {
		for {
			req, err := stream.Recv()
			if err == io.EOF {
				close(inc)
				return
			}
			if err != nil {
				log.Error(err)
				ec <- err
				return
			}
			inc <- req
		}
	}()
	for {
		select {
		case err = <-ec:
			log.Error(err)
			return err
		case r, ok := <-inc:
			if !ok {
				inc = nil
				close(in)
				continue
			}
			in <- r
		case ir, ok := <-rets:
			if !ok {
				return nil
			}
			if err = stream.Send(ir); err != nil {
				log.Error(err)
				return err
			}
		}
	}
}

// GetPathsWithToken gets paths from traces that were run in response to a request
func (gs GRPCServ) GetPathsWithToken(stream pb.Atlas_GetPathsWithTokenServer) error {
	return nil
}
