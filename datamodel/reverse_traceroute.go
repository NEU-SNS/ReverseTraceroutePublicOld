package datamodel

import "time"

// RevtrHop is a hop of a reverse traceroute
type RevtrHop struct {
	Hop  uint32
	Type int
}

// Revtr is a reverse traceroute
type Revtr struct {
	Src, Dst   uint32
	Runtime    time.Duration
	RRIssued   int
	TSIssued   int
	StopReason string
	Path       []RevtrHop
}
