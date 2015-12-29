package datamodel

// Adjacency represents the adjacency of 2 ips
type Adjacency struct {
	IP1, IP2 uint32
	Cnt      int
}

// AdjacencyToDest is ...
type AdjacencyToDest struct {
	Dest24   uint32
	Address  uint32
	Adjacent uint32
	Cnt      int
}
