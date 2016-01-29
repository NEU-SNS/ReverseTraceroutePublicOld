package revtr

import "github.com/NEU-SNS/ReverseTraceroute/dataaccess"

// Config represents the config options for the revtr service.
type Config struct {
	Db     dataaccess.DbConfig
	RootCA *string `flag:"root-ca"`
}

// NewConfig creates a new config struct
func NewConfig() Config {
	return Config{
		RootCA: new(string),
	}
}
