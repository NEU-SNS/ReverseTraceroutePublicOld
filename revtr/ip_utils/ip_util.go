package iputil

import "net"

var (
	pn1, pn2, pn3 *net.IPNet
)

func init() {
	var err error

	_, pn1, err = net.ParseCIDR("192.168.0.0/16")
	if err != nil {
		panic(err)
	}
	_, pn2, err = net.ParseCIDR("10.0.0.0/8")
	if err != nil {
		panic(err)
	}
	_, pn3, err = net.ParseCIDR("172.16.0.0/12")
	if err != nil {
		panic(err)
	}
}

// IsPrivate returns true if ip is a private address
func IsPrivate(ip net.IP) bool {
	if pn1.Contains(ip) {
		return true
	}
	if pn2.Contains(ip) {
		return true
	}
	return pn3.Contains(ip)
}
