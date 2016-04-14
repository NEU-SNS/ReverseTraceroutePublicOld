package v2api

import (
	"crypto/tls"

	"golang.org/x/net/context"

	"github.com/NEU-SNS/ReverseTraceroute/dataaccess"
	"github.com/NEU-SNS/ReverseTraceroute/revtr"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

// CreateServer creates a grpc Server that serves the v2api
func CreateServer(s revtr.Server, conf *tls.Config) *grpc.Server {
	opts := []grpc.ServerOption{
		grpc.Creds(credentials.NewTLS(conf)),
	}
	serv := grpc.NewServer(opts...)
	pb.RegisterRevtrServer(serv, CreateAPI(s))
	return serv
}

// CreateAPI returns pb.RevtrServer that uses the revtr.Server s
func CreateAPI(s revtr.Server) pb.RevtrServer {
	return api{s: s}
}

type api struct {
	s revtr.Server
}

const (
	authHeader = "revtr-key"
)

var (
	ErrUnauthorizedRequest = grpc.Errorf(codes.Unauthenticated, "unauthorized request")
	ErrInvalidBatchId      = grpc.Errorf(codes.FailedPrecondition, "invalid batch id")
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
			ret, err := a.s.RunRevtr(req, key)
			if err != nil {
				return nil, rpcError(err)
			}
			return ret, nil
		}
	}
	return nil, ErrUnauthorizedRequest
}

func (a api) GetRevtr(ctx context.Context, req *pb.GetRevtrReq) (*pb.GetRevtrResp, error) {
	if md, hasMD := metadata.FromContext(ctx); hasMD {
		if key, auth := checkAuth(md); auth {
			ret, err := a.s.GetRevtr(req, key)
			if err != nil {
				return nil, rpcError(err)
			}
			return ret, nil
		}
	}
	return nil, ErrUnauthorizedRequest
}

func (a api) GetSources(ctx context.Context, req *pb.GetSourcesReq) (*pb.GetSourcesResp, error) {
	if md, hasMD := metadata.FromContext(ctx); hasMD {
		if key, auth := checkAuth(md); auth {
			ret, err := a.s.GetSources(req, key)
			if err != nil {
				return nil, rpcError(err)
			}
			return ret, nil
		}
	}
	return nil, ErrUnauthorizedRequest
}

func rpcError(err error) error {
	switch err {
	case dataaccess.ErrNoRevtrUserFound:
		return ErrUnauthorizedRequest
	case revtr.ErrInvalidBatchId:
		return ErrInvalidBatchId
	default:
		return err
	}
}
