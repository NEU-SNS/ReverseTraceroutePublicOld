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
package prometheus

import (
	"runtime"
	"runtime/debug"
	"time"
)

type goCollector struct {
	goroutines Gauge
	gcDesc     *Desc
}

// NewGoCollector returns a collector which exports metrics about the current
// go process.
func NewGoCollector() *goCollector {
	return &goCollector{
		goroutines: NewGauge(GaugeOpts{
			Name: "go_goroutines",
			Help: "Number of goroutines that currently exist.",
		}),
		gcDesc: NewDesc(
			"go_gc_duration_seconds",
			"A summary of the GC invocation durations.",
			nil, nil),
	}
}

// Describe returns all descriptions of the collector.
func (c *goCollector) Describe(ch chan<- *Desc) {
	ch <- c.goroutines.Desc()
	ch <- c.gcDesc
}

// Collect returns the current state of all metrics of the collector.
func (c *goCollector) Collect(ch chan<- Metric) {
	c.goroutines.Set(float64(runtime.NumGoroutine()))
	ch <- c.goroutines

	var stats debug.GCStats
	stats.PauseQuantiles = make([]time.Duration, 5)
	debug.ReadGCStats(&stats)

	quantiles := make(map[float64]float64)
	for idx, pq := range stats.PauseQuantiles[1:] {
		quantiles[float64(idx+1)/float64(len(stats.PauseQuantiles)-1)] = pq.Seconds()
	}
	quantiles[0.0] = stats.PauseQuantiles[0].Seconds()
	ch <- MustNewConstSummary(c.gcDesc, uint64(stats.NumGC), float64(stats.PauseTotal.Seconds()), quantiles)
}
