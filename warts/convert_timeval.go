// +build !386

package warts

import "syscall"

func convertTimeval(t *syscall.Timeval, val uint32) {
	t.Sec = int64(val / 1000000)
	t.Usec = int64(val % 1000000)
}

func setSecond(t *syscall.Timeval, val uint32) {
	t.Sec = int64(val)
}

func setUSecond(t *syscall.Timeval, val uint32) {
	t.Usec = int64(val)
}
