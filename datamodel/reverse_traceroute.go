package datamodel

// RevtrUser is an authorized user of the revtr system
type RevtrUser struct {
	ID    uint32
	Name  string
	Email string
	Max   uint32
	Delay uint32
	Key   string
}
