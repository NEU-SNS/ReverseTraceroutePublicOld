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
package controller

import (
	con "golang.org/x/net/context"

	"github.com/NEU-SNS/ReverseTraceroute/cache"
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/log"
)

type pingCache struct {
	c cache.Cache
}

func (pc pingCache) pingCacheStep(next pingFunc) pingFunc {

	return func(ctx con.Context, pm <-chan []*dm.PingMeasurement) <-chan *dm.Ping {
		ret := make(chan *dm.Ping)
		n := make(chan []*dm.PingMeasurement, 1)
		go func() {
			for {
				select {
				case <-ctx.Done():
					close(ret)
					return
				case m := <-pm:
					check := make([]cache.Keyer, 0)
					db := make([]*dm.PingMeasurement, 0)
					checking := make(map[string]*dm.PingMeasurement)
					for _, p := range m {
						if p.CheckCache {
							check = append(check, p)
							checking[p.Key()] = p
						} else {
							db = append(db, p)
						}
					}
					res := next(ctx, n)
					n <- db
					cached, err := pc.c.GetMulti(check)
					if err != nil {
						log.Errorf("Failed to check cache: %v", err)
					}
					db = make([]*dm.PingMeasurement, 0)
					for _, item := range check {
						p := &dm.Ping{}
						if i, ok := cached[item.Key()]; ok {
							err := i.Unmarshal(p)
							if err != nil {
								log.Errorf("Failed to unmarshal ping: %v", err)
								db = append(db, checking[item.Key()])
								continue
							}
							ret <- p
						} else {
							db = append(db, checking[item.Key()])
						}
					}
					n <- db
					close(n)
					for p := range res {
						ret <- p
					}
					close(ret)
					return
				}
			}
		}()
		return ret
	}
}
