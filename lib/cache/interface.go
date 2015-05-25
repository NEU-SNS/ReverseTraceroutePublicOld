package cache

type Cache interface {
	Get(interface{}) (interface{}, error)
	Set(string, interface{}) error
}
