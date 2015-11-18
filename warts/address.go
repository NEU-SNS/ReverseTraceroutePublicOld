package warts

import (
	"io"
	"sync"

	"github.com/NEU-SNS/ReverseTraceroute/util"
)

// Address represents an address
type Address struct {
	Type    uint8
	Address uint64
}

func (a Address) String() string {
	switch a.Type {
	case 0x01:
		ip, err := util.Int32ToIPString(uint32(a.Address))
		if err != nil {
			return ""
		}
		return ip
	case 0x02, 0x03, 0x04:
		return ""
	}
	return ""
}

// AddressRefs tracks the addresses in a warts file
type AddressRefs struct {
	mu    sync.Mutex
	id    uint32
	addrs map[uint32]Address
}

// Add adds an address to the AddressRefs
func (ar *AddressRefs) Add(addr Address) {
	ar.mu.Lock()
	id := ar.id
	ar.id++
	ar.addrs[id] = addr
	ar.mu.Unlock()
}

// Get gets an address by id from addressrefs
func (ar *AddressRefs) Get(id uint32) Address {
	return ar.addrs[id]
}

// NewAddressRefs creates a new AddressRefs structure
func NewAddressRefs() *AddressRefs {
	return &AddressRefs{
		addrs: make(map[uint32]Address),
	}
}

func readReferencedAddress(f io.Reader, addrs *AddressRefs) (Address, error) {
	addr, err := readUint32(f)
	if err != nil {
		return Address{}, err
	}
	return addrs.Get(addr), nil
}

func readAddress(f io.Reader, addrs *AddressRefs) (Address, error) {
	a := Address{}
	length, err := readUint8(f)
	if err != nil {
		return a, err
	}
	if length == 0 {
		id, err := readUint32(f)
		if err != nil {
			return a, err
		}
		return addrs.Get(id), nil
	}
	t, err := readUint8(f)
	if err != nil {
		return a, err
	}
	res, err := readBytes(f, int(length))
	if err != nil {
		return a, err
	}
	addr := sliceToUint64(res)
	a.Type = t
	a.Address = addr
	addrs.Add(a)
	return a, nil

}
