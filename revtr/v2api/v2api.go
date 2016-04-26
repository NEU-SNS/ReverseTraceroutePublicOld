package v2api

import (
	"crypto/tls"

	"golang.org/x/net/context"

	"github.com/NEU-SNS/ReverseTraceroute/revtr/pb"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/repository"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

// CreateServer creates a grpc Server that serves the v2api
func CreateServer(s server.RevtrServer, conf *tls.Config) *grpc.Server {
	opts := []grpc.ServerOption{
		grpc.Creds(credentials.NewTLS(conf)),
	}
	serv := grpc.NewServer(opts...)
	pb.RegisterRevtrServer(serv, CreateAPI(s))
	return serv
}

// CreateAPI returns pb.RevtrServer that uses the revtr.Server s
func CreateAPI(s server.RevtrServer) pb.RevtrServer {
	return api{s: s}
}

type api struct {
	s server.RevtrServer
}

const (
	authHeader = "revtr-key"
)

var (
	ErrUnauthorizedRequest = grpc.Errorf(codes.Unauthenticated, "unauthorized request")
	ErrInvalidBatchId      = grpc.Errorf(codes.FailedPrecondition, "invalid batch id")
	ErrNoRevtrsToRun       = grpc.Errorf(codes.FailedPrecondition, "no revtrs to run")
	ErrFailedToCreateBatch = grpc.Errorf(codes.Internal, "failed to create batch")
)

func checkAuth(m metadata.MD) (string, bool) {
	if val, ok := m[authHeader]; ok {
		if len(val) != 1 {
			return "", false
		}
		return val[0], true
	}
	return "", false
}

func (a api) RunRevtr(ctx context.Context, req *pb.RunRevtrReq) (*pb.RunRevtrResp, error) {
	if md, hasMD := metadata.FromContext(ctx); hasMD {
		if key, auth := checkAuth(md); auth {
			req.Auth = key
		}
	}
	if req.Auth == "" {
		return nil, ErrUnauthorizedRequest
	}
	ret, err := a.s.RunRevtr(req)
	if err != nil {
		return nil, rpcError(err)
	}
	return ret, nil
}

func (a api) GetRevtr(ctx context.Context, req *pb.GetRevtrReq) (*pb.GetRevtrResp, error) {
	if md, hasMD := metadata.FromContext(ctx); hasMD {
		if key, auth := checkAuth(md); auth {
			req.Auth = key
		}
	}
	if req.Auth == "" {
		return nil, ErrUnauthorizedRequest
	}
	ret, err := a.s.GetRevtr(req)
	if err != nil {
		return nil, rpcError(err)
	}
	return ret, nil
}

func (a api) GetSources(ctx context.Context, req *pb.GetSourcesReq) (*pb.GetSourcesResp, error) {
	if md, hasMD := metadata.FromContext(ctx); hasMD {
		if key, auth := checkAuth(md); auth {
			req.Auth = key
		}
	}
	if req.Auth == "" {
		return nil, ErrUnauthorizedRequest
	}
	ret, err := a.s.GetSources(req)
	if err != nil {
		return nil, rpcError(err)
	}
	return ret, nil
}

func rpcError(err error) error {
	switch err {
	case repo.ErrNoRevtrUserFound:
		return ErrUnauthorizedRequest
	default:
		return err
	}
}
