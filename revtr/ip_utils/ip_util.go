package iputil

import "net"

// IsPrivate returns true if ip is a private address
func IsPrivate(ip net.IP) bool {
	return !ip.IsGlobalUnicast()
}
