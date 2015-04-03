package vantagepoint

type Vantagepoint struct {
	version  int
	ip       string
	port     int
	hostname string
	socket   string
	canspoof bool
}
