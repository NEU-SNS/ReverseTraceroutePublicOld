package dataaccess

type DataAccess interface {
	GetServices()
}

type dataAccess struct {
}

func (d *dataAccess) GetServices() {
}
