package types

// AdjacencySource is the interface for something that provides adjacnecies
type AdjacencySource interface {
	GetAdjacenciesByIP1(uint32) ([]Adjacency, error)
	GetAdjacenciesByIP2(uint32) ([]Adjacency, error)
	GetAdjacencyToDestByAddrAndDest24(uint32, uint32) ([]AdjacencyToDest, error)
}

// ClusterSource is the interface for something that provides cluster data
type ClusterSource interface {
	GetClusterIDByIP(uint32) (int, error)
	GetIPsForClusterID(int) ([]uint32, error)
}

// Config represents the config options for the revtr service.
type Config struct {
	RootCA   *string `flag:"root-ca"`
	CertFile *string `flag:"cert-file"`
	KeyFile  *string `flag:"key-file"`
}

// NewConfig creates a new config struct
func NewConfig() Config {
	return Config{
		RootCA:   new(string),
		CertFile: new(string),
		KeyFile:  new(string),
	}
}

// Adjacency represents the adjacency of 2 ips
type Adjacency struct {
	IP1, IP2 uint32
	Cnt      uint32
}

// AdjacencyToDest is ...
type AdjacencyToDest struct {
	Dest24   uint32
	Address  uint32
	Adjacent uint32
	Cnt      uint32
}
