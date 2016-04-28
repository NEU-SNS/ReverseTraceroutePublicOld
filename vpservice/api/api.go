package api

import (
	"crypto/tls"

	"golang.org/x/net/context"

	"github.com/NEU-SNS/ReverseTraceroute/vpservice/pb"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// CreateServer creates a grpc server for the vpservice api
func CreateServer(s server.VPServer, conf *tls.Config) *grpc.Server {
	opts := []grpc.ServerOption{
		grpc.Creds(credentials.NewTLS(conf)),
	}
	serv := grpc.NewServer(opts...)
	pb.RegisterVPServiceServer(serv, CreateAPI(s))
	return serv
}

type api struct {
	s server.VPServer
}

// CreateAPI returns a pb.VPService that uses the server.VPServer s
func CreateAPI(s server.VPServer) pb.VPServiceServer {
	return api{s: s}
}

func (a api) GetVPs(ctx context.Context, req *pb.VPRequest) (*pb.VPReturn, error) {
	return a.s.GetVPs(req)
}

func (a api) GetRRSpoofers(ctx context.Context, req *pb.RRSpooferRequest) (*pb.RRSpooferResponse, error) {
	return nil, nil
}

func (a api) GetTSSpoofers(ctx context.Context, req *pb.TSSpooferRequest) (*pb.TSSpooferResponse, error) {
	return nil, nil
}
