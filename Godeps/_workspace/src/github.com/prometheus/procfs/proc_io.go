/*
Copyright (c) 2015, Northeastern University
 All rights reserved.

 Redistribution and use in source and binary forms, with or without
 modification, are permitted provided that the following conditions are met:
     * Redistributions of source code must retain the above copyright
       notice, this list of conditions and the following disclaimer.
     * Redistributions in binary form must reproduce the above copyright
       notice, this list of conditions and the following disclaimer in the
       documentation and/or other materials provided with the distribution.
     * Neither the name of the Northeastern University nor the
       names of its contributors may be used to endorse or promote products
       derived from this software without specific prior written permission.

 THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
 ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 DISCLAIMED. IN NO EVENT SHALL Northeastern University BE LIABLE FOR ANY
 DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/
package procfs

import (
	"fmt"
	"io/ioutil"
)

// ProcIO models the content of /proc/<pid>/io.
type ProcIO struct {
	// Chars read.
	RChar uint64
	// Chars written.
	WChar uint64
	// Read syscalls.
	SyscR uint64
	// Write syscalls.
	SyscW uint64
	// Bytes read.
	ReadBytes uint64
	// Bytes written.
	WriteBytes uint64
	// Bytes written, but taking into account truncation. See
	// Documentation/filesystems/proc.txt in the kernel sources for
	// detailed explanation.
	CancelledWriteBytes int64
}

// NewIO creates a new ProcIO instance from a given Proc instance.
func (p Proc) NewIO() (ProcIO, error) {
	pio := ProcIO{}

	f, err := p.open("io")
	if err != nil {
		return pio, err
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return pio, err
	}

	ioFormat := "rchar: %d\nwchar: %d\nsyscr: %d\nsyscw: %d\n" +
		"read_bytes: %d\nwrite_bytes: %d\n" +
		"cancelled_write_bytes: %d\n"

	_, err = fmt.Sscanf(string(data), ioFormat, &pio.RChar, &pio.WChar, &pio.SyscR,
		&pio.SyscW, &pio.ReadBytes, &pio.WriteBytes, &pio.CancelledWriteBytes)
	if err != nil {
		return pio, err
	}

	return pio, nil
}
