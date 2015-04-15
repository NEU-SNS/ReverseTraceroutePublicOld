package router

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
	services []*Service
}

func New() Router {
	s := make([]*Service, 0, 10)
	return &router{services: s}
}

type Router interface {
	RegisterServices(services ...*Service)
	RouteRequest(interface{})
}

func (r *router) RegisterServices(services ...*Service) {
	for _, service := range services {
		r.services = append(r.services, service)
	}
}

func (r *router) RouteRequest(x interface{}) {

}
