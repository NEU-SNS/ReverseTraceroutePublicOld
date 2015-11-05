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

/*
Package controller is the package that is a controller
*/
package controller

import (
	da "github.com/NEU-SNS/ReverseTraceroute/dataaccess"
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/prometheus/log"
	"golang.org/x/net/context"
)

type pingDB struct {
	db *da.DataAccess
}

func (pdb pingDB) pingDBStep(next pingFunc) pingFunc {
	return func(ctx context.Context, pm <-chan []*dm.PingMeasurement) <-chan *dm.Ping {
		ret := make(chan *dm.Ping)
		exit := make(chan struct{})
		n := make(chan []*dm.PingMeasurement, 2)
		go func() {
			for {
				select {
				case <-ctx.Done():
					close(exit)
					close(ret)
					return
				case m := <-pm:
					log.Info("Ping DB step")
					var check []*dm.PingMeasurement
					var meas []*dm.PingMeasurement
					checking := make(map[string]*dm.PingMeasurement)
					for _, p := range m {
						if p.CheckDb {
							check = append(check, p)
							checking[p.Key()] = p
						} else {
							meas = append(meas, p)
						}
					}
					res := next(ctx, n)
					log.Info("sending to remote")
					n <- meas
					log.Info("done sending from remote")
					stored, err := pdb.db.GetPingsMulti(check)
					if err != nil {
						log.Errorf("Failed to check db: %v", err)
					}
					meas = make([]*dm.PingMeasurement, 0)
					for _, item := range stored {
						delete(checking, item.Key())
						ret <- item
					}
					for _, left := range checking {
						meas = append(meas, left)
					}
					n <- meas
					close(n)
					for p := range res {
						go func(ping *dm.Ping) {
							if ping.Error != "" {
								return
							}
							err := pdb.db.StorePing(ping)
							if err != nil {
								log.Error(err)
							}
						}(p)
						select {
						case <-exit:
							continue
						case ret <- p:
						}
					}
					log.Info("Closing return")
					close(ret)
					return
				}
			}
		}()
		return ret
	}
}

type traceDB struct {
	db *da.DataAccess
}

func (tdb traceDB) traceDBStep(next traceFunc) traceFunc {
	return func(ctx context.Context, pm <-chan []*dm.TracerouteMeasurement) <-chan *dm.Traceroute {
		exit := make(chan struct{})
		ret := make(chan *dm.Traceroute)
		n := make(chan []*dm.TracerouteMeasurement, 2)
		go func() {
			for {
				select {
				case <-ctx.Done():
					close(exit)
					close(ret)
					return
				case m := <-pm:
					if len(m) == 0 {
						return
					}
					var check []*dm.TracerouteMeasurement
					var meas []*dm.TracerouteMeasurement
					checking := make(map[string]*dm.TracerouteMeasurement)
					for _, p := range m {
						if p.CheckDb {
							check = append(check, p)
							checking[p.Key()] = p
						} else {
							meas = append(meas, p)
						}
					}
					res := next(ctx, n)
					n <- meas
					stored, err := tdb.db.GetTraceMulti(check)
					if err != nil {
						log.Errorf("Failed to check db: %v", err)
					}
					meas = make([]*dm.TracerouteMeasurement, 0)
					for _, item := range stored {
						delete(checking, item.Key())
						ret <- item
					}
					for _, left := range checking {
						meas = append(meas, left)
					}
					n <- meas
					close(n)
					for t := range res {
						go func(trace *dm.Traceroute) {
							err := tdb.db.StoreTraceroute(trace)
							if err != nil {
								log.Error(err)
							}
						}(t)
						select {
						case <-exit:
							continue
						case ret <- t:
						}
					}
					close(ret)
					return
				}
			}
		}()
		return ret
	}
}
