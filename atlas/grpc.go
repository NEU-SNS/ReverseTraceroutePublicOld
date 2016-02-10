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
	ec := make(chan error, 1)
	rec := make(chan *dm.IntersectionResponse, 1)
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()
	go func() {
		for {
			req, err := stream.Recv()
			if err == io.EOF {
				close(rec)
				return
			}
			log.Debug("Recv request: ", req)
			if err != nil {
				log.Error(err)
				ec <- err
				return
			}
			rets, err := gs.AtlasService.GetIntersectingPath(ctx, req)
			if err != nil {
				log.Error(err)
				ec <- err
				return
			}

			for _, ret := range rets {
				rec <- ret
			}
		}
	}()
	for {
		select {
		case err := <-ec:
			log.Error(err)
			return err
		case ir, ok := <-rec:
			log.Debug("Got from rest: ", ir, " ", ok)
			if !ok {
				return nil
			}
			if err := stream.Send(ir); err != nil {
				log.Error(err)
				return err
			}

		}
	}
}

// GetPathsWithToken gets paths from traces that were run in response to a request
func (gs GRPCServ) GetPathsWithToken(stream pb.Atlas_GetPathsWithTokenServer) error {
	log.Debug("GetPathsWithToken")
	ec := make(chan error, 1)
	rets := make(chan *dm.TokenResponse, 1)
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()
	go func() {
		for {
			req, err := stream.Recv()
			log.Debug("GetPathsWithToken: ", req)
			if err == io.EOF {
				close(rets)
				return
			}
			if err != nil {
				ec <- err
				log.Error(err)
				return
			}
			rest, err := gs.AtlasService.GetPathsWithToken(ctx, req)
			if err != nil {
				log.Error(err)
				ec <- err
				return
			}
			for _, ret := range rest {
				rets <- ret
			}
		}
	}()
	for {
		select {
		case err := <-ec:
			log.Error(err)
			return err
		case tr, ok := <-rets:
			log.Debug("Got from token intersection: ", tr)
			if !ok {
				return nil
			}
			if err := stream.Send(tr); err != nil {
				log.Error(err)
				return err
			}
		}
	}

}
