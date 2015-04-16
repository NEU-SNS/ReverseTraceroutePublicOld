package controller

import (
	"net"
)

type Service struct {
	Port   int
	IPAddr string
	Key    string
	ip     net.IP
}

func (s *Service) IP() net.IP {
	if s.ip == nil {
		s.ip = net.ParseIP(s.IPAddr)
	}
	return s.ip
}

type router struct {
	services map[string]*Service
}

func createRouter() Router {
	s := make(map[string]*Service)
	return &router{services: s}
}

func NewRouter() Router {
	s := make(map[string]*Service)
	return &router{services: s}
}

type Router interface {
	RegisterServices(services ...*Service)
	RouteRequest(r Request) (RoutedRequest, error)
}

func (r *router) RegisterServices(services ...*Service) {
	for _, service := range services {
		r.services[service.Key] = service
	}
}

func (r *router) RouteRequest(req Request) (RoutedRequest, error) {

	return nil, nil
}
