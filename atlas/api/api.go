package api

import (
	"crypto/tls"
	"io"

	"github.com/NEU-SNS/ReverseTraceroute/atlas/pb"
	"github.com/NEU-SNS/ReverseTraceroute/atlas/server"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// CreateServer creates a grpc server fro the Atlas api
func CreateServer(s server.AtlasServer, conf *tls.Config) *grpc.Server {
	opts := []grpc.ServerOption{
		grpc.Creds(credentials.NewTLS(conf)),
	}
	serv := grpc.NewServer(opts...)
	pb.RegisterAtlasServer(serv, CreateAPI(s))
	return serv
}

type api struct {
	s server.AtlasServer
}

// CreateAPI returns a pb.AtlasServer that uses the given server
func CreateAPI(s server.AtlasServer) pb.AtlasServer {
	return api{s: s}
}

func (a api) GetIntersectingPath(stream pb.Atlas_GetIntersectingPathServer) error {
	ctx := stream.Context()
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			log.Error(err)
			return err
		}
		resp, err := a.s.GetIntersectingPath(req)
		if err != nil {
			log.Error(err)
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := stream.Send(resp); err != nil {
				return err
			}
		}
	}
}

func (a api) GetPathsWithToken(stream pb.Atlas_GetPathsWithTokenServer) error {
	ctx := stream.Context()
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			log.Error(err)
			return err
		}
		resp, err := a.s.GetPathsWithToken(req)
		if err != nil {
			log.Error(err)
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := stream.Send(resp); err != nil {
				return err
			}
		}
	}
}
