package atlas

import (
	"io"

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
	ret, err := gs.AtlasService.GetIntersectingPath(stream.Context(), in)
	if err != nil {
		log.Error(err)
		return err
	}
	go func() {
		req, err := stream.Recv()
		if err == io.EOF {
			// set inc to nil to close off that par tof the select loop
			inc = nil
			close(in)
			return
		}
		if err != nil {
			ec <- err
			return
		}
		inc <- req
	}()
	for {
		select {
		case err = <-ec:
			close(in)
			return err
		case in <- <-inc:
		case ir, ok := <-ret:
			if !ok {
				return nil
			}
			if err = stream.Send(ir); err != nil {
				log.Error(err)
				close(in)
				return err
			}
		}
	}
}

// GetPathsWithToken gets paths from traces that were run in response to a request
func (gs GRPCServ) GetPathsWithToken(stream pb.Atlas_GetPathsWithTokenServer) error {
	return nil
}
