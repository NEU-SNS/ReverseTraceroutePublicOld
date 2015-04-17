package datamodel

import (
	"fmt"
	"net"
)

type Service struct {
	Port   int
	IPAddr string
	Key    string
	ip     net.IP
	Proto  string
	Api    map[MType]string
}

func (s *Service) FormatIp() string {
	return fmt.Sprintf("%s:%s", s.IP(), s.Port)
}

func (s *Service) IP() net.IP {
	if s.ip == nil {
		s.ip = net.ParseIP(s.IPAddr)
	}
	return s.ip
}
