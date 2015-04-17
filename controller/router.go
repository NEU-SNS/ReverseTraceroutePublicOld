package controller

import (
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"net/rpc/jsonrpc"
	"time"
)

type router struct {
	services map[string]*dm.Service
}

func createRouter() Router {
	s := make(map[string]*dm.Service)
	return &router{services: s}
}

func NewRouter() Router {
	s := make(map[string]*dm.Service)
	return &router{services: s}
}

type Router interface {
	RegisterServices(services ...*dm.Service)
	RouteRequest(r Request) (RoutedRequest, error)
}

func (r *router) RegisterServices(services ...*dm.Service) {
	for _, service := range services {
		r.services[service.Key] = service
	}
}

func (r *router) RouteRequest(req Request) (RoutedRequest, error) {
	s := r.services[req.Key]
	if s == nil {
		return nil, ErrorServiceNotFound
	}
	return wrapRequest(req, s), nil
}

func wrapRequest(req Request, s *dm.Service) RoutedRequest {
	return func() (*MReturn, error) {

		req.Stime = time.Now()
		c, err := jsonrpc.Dial(s.Proto, s.FormatIp())
		if err != nil {
			return nil, err
		}
		defer c.Close()
		err = c.Call(s.Api[req.Type], nil, nil)
		req.Dur = time.Since(req.Stime)
		if err != nil {
			return nil, err
		}
		return nil, nil
	}

}
