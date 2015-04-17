package dataaccess

import (
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
)

type DataAccess interface {
	GetServices()
}

type dataAccess struct {
}

func (d *dataAccess) GetServices(ip string) []*dm.Service {
	return nil
}
