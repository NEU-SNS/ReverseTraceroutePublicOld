package datamodel

// User is a user of the controller api
type User struct {
	ID    uint32
	Name  string
	EMail string
	Max   uint32
	Delay uint32
	Key   string
}
