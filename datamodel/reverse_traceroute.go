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

// RevtrUser is an authorized user of the revtr system
type RevtrUser struct {
	ID    uint32
	Name  string
	Email string
	Max   uint32
	Delay uint32
	Key   string
}
