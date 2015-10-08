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
package stats

import (
	"math"
	"sync"
	"time"
)

// Tracker is a min/max value tracker that keeps track of its min/max values
// over a given period of time, and with a given resolution. The initial min
// and max values are math.MaxInt64 and math.MinInt64 respectively.
type Tracker struct {
	mu           sync.RWMutex
	min, max     int64 // All time min/max.
	minTS, maxTS [3]*timeseries
	lastUpdate   time.Time
}

// newTracker returns a new Tracker.
func newTracker() *Tracker {
	now := TimeNow()
	t := &Tracker{}
	t.minTS[hour] = newTimeSeries(now, time.Hour, time.Minute)
	t.minTS[tenminutes] = newTimeSeries(now, 10*time.Minute, 10*time.Second)
	t.minTS[minute] = newTimeSeries(now, time.Minute, time.Second)
	t.maxTS[hour] = newTimeSeries(now, time.Hour, time.Minute)
	t.maxTS[tenminutes] = newTimeSeries(now, 10*time.Minute, 10*time.Second)
	t.maxTS[minute] = newTimeSeries(now, time.Minute, time.Second)
	t.init()
	return t
}

func (t *Tracker) init() {
	t.min = math.MaxInt64
	t.max = math.MinInt64
	for _, ts := range t.minTS {
		ts.set(math.MaxInt64)
	}
	for _, ts := range t.maxTS {
		ts.set(math.MinInt64)
	}
}

func (t *Tracker) advance() time.Time {
	now := TimeNow()
	for _, ts := range t.minTS {
		ts.advanceTimeWithFill(now, math.MaxInt64)
	}
	for _, ts := range t.maxTS {
		ts.advanceTimeWithFill(now, math.MinInt64)
	}
	return now
}

// LastUpdate returns the last update time of the range.
func (t *Tracker) LastUpdate() time.Time {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.lastUpdate
}

// Push adds a new value if it is a new minimum or maximum.
func (t *Tracker) Push(value int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.lastUpdate = t.advance()
	if t.min > value {
		t.min = value
	}
	if t.max < value {
		t.max = value
	}
	for _, ts := range t.minTS {
		if ts.headValue() > value {
			ts.set(value)
		}
	}
	for _, ts := range t.maxTS {
		if ts.headValue() < value {
			ts.set(value)
		}
	}
}

// Min returns the minimum value of the tracker
func (t *Tracker) Min() int64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.min
}

// Max returns the maximum value of the tracker.
func (t *Tracker) Max() int64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.max
}

// Min1h returns the minimum value for the last hour.
func (t *Tracker) Min1h() int64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.advance()
	return t.minTS[hour].min()
}

// Max1h returns the maximum value for the last hour.
func (t *Tracker) Max1h() int64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.advance()
	return t.maxTS[hour].max()
}

// Min10m returns the minimum value for the last 10 minutes.
func (t *Tracker) Min10m() int64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.advance()
	return t.minTS[tenminutes].min()
}

// Max10m returns the maximum value for the last 10 minutes.
func (t *Tracker) Max10m() int64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.advance()
	return t.maxTS[tenminutes].max()
}

// Min1m returns the minimum value for the last 1 minute.
func (t *Tracker) Min1m() int64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.advance()
	return t.minTS[minute].min()
}

// Max1m returns the maximum value for the last 1 minute.
func (t *Tracker) Max1m() int64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.advance()
	return t.maxTS[minute].max()
}

// Reset resets the range to an empty state.
func (t *Tracker) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := TimeNow()
	for _, ts := range t.minTS {
		ts.reset(now)
	}
	for _, ts := range t.maxTS {
		ts.reset(now)
	}
	t.init()
}
