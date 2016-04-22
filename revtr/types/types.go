package types

import "github.com/NEU-SNS/ReverseTraceroute/datamodel"

// AdjacencySource is the interface for something that provides adjacnecies
type AdjacencySource interface {
	GetAdjacenciesByIP1(uint32) ([]datamodel.Adjacency, error)
	GetAdjacenciesByIP2(uint32) ([]datamodel.Adjacency, error)
	GetAdjacencyToDestByAddrAndDest24(uint32, uint32) ([]datamodel.AdjacencyToDest, error)
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
