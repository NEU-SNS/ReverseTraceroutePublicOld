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
// Copyright 2015 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package prometheus

import "github.com/prometheus/procfs"

type processCollector struct {
	pid             int
	collectFn       func(chan<- Metric)
	pidFn           func() (int, error)
	cpuTotal        Counter
	openFDs, maxFDs Gauge
	vsize, rss      Gauge
	startTime       Gauge
}

// NewProcessCollector returns a collector which exports the current state of
// process metrics including cpu, memory and file descriptor usage as well as
// the process start time for the given process id under the given namespace.
func NewProcessCollector(pid int, namespace string) *processCollector {
	return NewProcessCollectorPIDFn(
		func() (int, error) { return pid, nil },
		namespace,
	)
}

// NewProcessCollectorPIDFn returns a collector which exports the current state
// of process metrics including cpu, memory and file descriptor usage as well
// as the process start time under the given namespace. The given pidFn is
// called on each collect and is used to determine the process to export
// metrics for.
func NewProcessCollectorPIDFn(
	pidFn func() (int, error),
	namespace string,
) *processCollector {
	c := processCollector{
		pidFn:     pidFn,
		collectFn: func(chan<- Metric) {},

		cpuTotal: NewCounter(CounterOpts{
			Namespace: namespace,
			Name:      "process_cpu_seconds_total",
			Help:      "Total user and system CPU time spent in seconds.",
		}),
		openFDs: NewGauge(GaugeOpts{
			Namespace: namespace,
			Name:      "process_open_fds",
			Help:      "Number of open file descriptors.",
		}),
		maxFDs: NewGauge(GaugeOpts{
			Namespace: namespace,
			Name:      "process_max_fds",
			Help:      "Maximum number of open file descriptors.",
		}),
		vsize: NewGauge(GaugeOpts{
			Namespace: namespace,
			Name:      "process_virtual_memory_bytes",
			Help:      "Virtual memory size in bytes.",
		}),
		rss: NewGauge(GaugeOpts{
			Namespace: namespace,
			Name:      "process_resident_memory_bytes",
			Help:      "Resident memory size in bytes.",
		}),
		startTime: NewGauge(GaugeOpts{
			Namespace: namespace,
			Name:      "process_start_time_seconds",
			Help:      "Start time of the process since unix epoch in seconds.",
		}),
	}

	// Set up process metric collection if supported by the runtime.
	if _, err := procfs.NewStat(); err == nil {
		c.collectFn = c.processCollect
	}

	return &c
}

// Describe returns all descriptions of the collector.
func (c *processCollector) Describe(ch chan<- *Desc) {
	ch <- c.cpuTotal.Desc()
	ch <- c.openFDs.Desc()
	ch <- c.maxFDs.Desc()
	ch <- c.vsize.Desc()
	ch <- c.rss.Desc()
	ch <- c.startTime.Desc()
}

// Collect returns the current state of all metrics of the collector.
func (c *processCollector) Collect(ch chan<- Metric) {
	c.collectFn(ch)
}

// TODO(ts): Bring back error reporting by reverting 7faf9e7 as soon as the
// client allows users to configure the error behavior.
func (c *processCollector) processCollect(ch chan<- Metric) {
	pid, err := c.pidFn()
	if err != nil {
		return
	}

	p, err := procfs.NewProc(pid)
	if err != nil {
		return
	}

	if stat, err := p.NewStat(); err == nil {
		c.cpuTotal.Set(stat.CPUTime())
		ch <- c.cpuTotal
		c.vsize.Set(float64(stat.VirtualMemory()))
		ch <- c.vsize
		c.rss.Set(float64(stat.ResidentMemory()))
		ch <- c.rss

		if startTime, err := stat.StartTime(); err == nil {
			c.startTime.Set(startTime)
			ch <- c.startTime
		}
	}

	if fds, err := p.FileDescriptorsLen(); err == nil {
		c.openFDs.Set(float64(fds))
		ch <- c.openFDs
	}

	if limits, err := p.NewLimits(); err == nil {
		c.maxFDs.Set(float64(limits.OpenFiles))
		ch <- c.maxFDs
	}
}
