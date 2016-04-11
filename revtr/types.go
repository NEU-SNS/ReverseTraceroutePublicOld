package revtr

import "github.com/NEU-SNS/ReverseTraceroute/dataaccess"

// Config represents the config options for the revtr service.
type Config struct {
	Db dataaccess.DbConfig

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
