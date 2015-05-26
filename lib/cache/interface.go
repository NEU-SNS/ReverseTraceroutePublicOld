package cache

type Cache interface {
	Get(string) (interface{}, error)
	Set(string, interface{}) error
}

type cache struct{}

func New() Cache {
	return &cache{}
}

func (c *cache) Get(key string) (interface{}, error) {
	return nil, nil
}

func (c *cache) Set(key string, data interface{}) error {
	return nil
}
