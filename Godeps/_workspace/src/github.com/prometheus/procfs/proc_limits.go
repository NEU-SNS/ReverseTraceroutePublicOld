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
	"bufio"
	"fmt"
	"regexp"
	"strconv"
)

// ProcLimits represents the soft limits for each of the process's resource
// limits.
type ProcLimits struct {
	CPUTime          int
	FileSize         int
	DataSize         int
	StackSize        int
	CoreFileSize     int
	ResidentSet      int
	Processes        int
	OpenFiles        int
	LockedMemory     int
	AddressSpace     int
	FileLocks        int
	PendingSignals   int
	MsqqueueSize     int
	NicePriority     int
	RealtimePriority int
	RealtimeTimeout  int
}

const (
	limitsFields    = 3
	limitsUnlimited = "unlimited"
)

var (
	limitsDelimiter = regexp.MustCompile("  +")
)

// NewLimits returns the current soft limits of the process.
func (p Proc) NewLimits() (ProcLimits, error) {
	f, err := p.open("limits")
	if err != nil {
		return ProcLimits{}, err
	}
	defer f.Close()

	var (
		l = ProcLimits{}
		s = bufio.NewScanner(f)
	)
	for s.Scan() {
		fields := limitsDelimiter.Split(s.Text(), limitsFields)
		if len(fields) != limitsFields {
			return ProcLimits{}, fmt.Errorf(
				"couldn't parse %s line %s", f.Name(), s.Text())
		}

		switch fields[0] {
		case "Max cpu time":
			l.CPUTime, err = parseInt(fields[1])
		case "Max file size":
			l.FileLocks, err = parseInt(fields[1])
		case "Max data size":
			l.DataSize, err = parseInt(fields[1])
		case "Max stack size":
			l.StackSize, err = parseInt(fields[1])
		case "Max core file size":
			l.CoreFileSize, err = parseInt(fields[1])
		case "Max resident set":
			l.ResidentSet, err = parseInt(fields[1])
		case "Max processes":
			l.Processes, err = parseInt(fields[1])
		case "Max open files":
			l.OpenFiles, err = parseInt(fields[1])
		case "Max locked memory":
			l.LockedMemory, err = parseInt(fields[1])
		case "Max address space":
			l.AddressSpace, err = parseInt(fields[1])
		case "Max file locks":
			l.FileLocks, err = parseInt(fields[1])
		case "Max pending signals":
			l.PendingSignals, err = parseInt(fields[1])
		case "Max msgqueue size":
			l.MsqqueueSize, err = parseInt(fields[1])
		case "Max nice priority":
			l.NicePriority, err = parseInt(fields[1])
		case "Max realtime priority":
			l.RealtimePriority, err = parseInt(fields[1])
		case "Max realtime timeout":
			l.RealtimeTimeout, err = parseInt(fields[1])
		}

		if err != nil {
			return ProcLimits{}, err
		}
	}

	return l, s.Err()
}

func parseInt(s string) (int, error) {
	if s == limitsUnlimited {
		return -1, nil
	}
	i, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("couldn't parse value %s: %s", s, err)
	}
	return int(i), nil
}
