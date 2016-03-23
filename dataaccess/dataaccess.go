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

// Package dataaccess provides database access
package dataaccess

import (
	"net"
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/dataaccess/sql"
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
)

// DbConfig is a database configuration
type DbConfig struct {
	WriteConfigs []Config
	ReadConfigs  []Config
}

// Config is a DB server config
type Config struct {
	User     string
	Password string
	Host     string
	Port     string
	Db       string
}

// DataAccess represents data access
type DataAccess struct {
	conf DbConfig
	db   *sql.DB
}

func uToNSec(u int64) int64 {
	//1000 nsec to a usec
	return u * 1000
}

// StoreTraceroute stores a traceroute
func (d *DataAccess) StoreTraceroute(t *dm.Traceroute) error {
	return d.db.StoreTraceroute(t)
}

// GetTRBySrcDst gets a trace by src and dst
func (d *DataAccess) GetTRBySrcDst(src, dst uint32) ([]*dm.Traceroute, error) {
	return d.db.GetTRBySrcDst(src, dst)
}

// GetTRBySrcDstWithStaleness gets a trace with the src and dst no older than the give time
func (d *DataAccess) GetTRBySrcDstWithStaleness(src, dst uint32, s time.Duration) ([]*dm.Traceroute, error) {
	return d.db.GetTRBySrcDstWithStaleness(src, dst, s)
}

// GetTraceMulti gets traceroutes that match the given TracerouteMeasurements
func (d *DataAccess) GetTraceMulti(in []*dm.TracerouteMeasurement) ([]*dm.Traceroute, error) {
	return d.db.GetTraceMulti(in)
}

// GetPingBySrcDst gets a ping
func (d *DataAccess) GetPingBySrcDst(src, dst uint32) ([]*dm.Ping, error) {
	return d.db.GetPingBySrcDst(src, dst)
}

// GetPingBySrcDstWithStaleness gets a ping
func (d *DataAccess) GetPingBySrcDstWithStaleness(src, dst uint32, s time.Duration) ([]*dm.Ping, error) {
	return d.db.GetPingBySrcDstWithStaleness(src, dst, s)
}

// StorePing stores a ping
func (d *DataAccess) StorePing(p *dm.Ping) error {
	return d.db.StorePing(p)
}

// Close closes
func (d *DataAccess) Close() error {
	return d.db.Close()
}

// GetPingsMulti gets pings that match the given PingMeasurements
func (d *DataAccess) GetPingsMulti(in []*dm.PingMeasurement) ([]*dm.Ping, error) {
	return d.db.GetPingsMulti(in)
}

// UpdateCheckStatus updates the health check status for a vp
func (d *DataAccess) UpdateCheckStatus(ip uint32, stat string) error {
	return d.db.UpdateCheckStatus(ip, stat)
}

// GetVPs get the VPs
func (d *DataAccess) GetVPs() ([]*dm.VantagePoint, error) {
	return d.db.GetVPs()
}

// GetActiveVPs get the vps which are currently connected to the controller
func (d *DataAccess) GetActiveVPs() ([]*dm.VantagePoint, error) {
	return d.db.GetActiveVPs()
}

// UpdateController updates the controller for a VP
func (d *DataAccess) UpdateController(ip, old, nc uint32) error {
	return d.db.UpdateController(ip, old, nc)
}

// ErrNoIntFound is returned when no intersection is found
var ErrNoIntFound = sql.ErrNoIntFound

// GetAtlasSources gets the sources used for the current dst and staleness
func (d *DataAccess) GetAtlasSources(dst uint32, stale time.Duration) ([]uint32, error) {
	return d.db.GetAtlasSources(dst, stale)
}

// FindIntersectingTraceroute finds a traceroute that intersects hop towards the dst
func (d *DataAccess) FindIntersectingTraceroute(pairs []dm.SrcDst, alias bool, stale time.Duration) ([]*dm.Path, error) {
	res, err := d.db.FindIntersectingTraceroute(pairs, alias, stale)
	if err == sql.ErrNoIntFound {
		return res, ErrNoIntFound
	}
	return res, err
}

// StoreAtlasTraceroute stores a traceroute in a form that the Atlas requires
func (d *DataAccess) StoreAtlasTraceroute(trace *dm.Traceroute) error {
	return d.db.StoreAtlasTraceroute(trace)
}

// StoreAdjacency stores an ajacienty
func (d *DataAccess) StoreAdjacency(l, r net.IP) error {
	return d.db.StoreAdjacency(l, r)
}

// StoreAdjacencyToDest stores and adjacency to dest
func (d *DataAccess) StoreAdjacencyToDest(dest24, addr, adj net.IP) error {
	return d.db.StoreAdjacencyToDest(dest24, addr, adj)
}

// GetAdjacenciesByIP1 gets ajd by ip1
func (d *DataAccess) GetAdjacenciesByIP1(ip uint32) ([]dm.Adjacency, error) {
	return d.db.GetAdjacenciesByIP1(ip)
}

// GetAdjacenciesByIP2 gets ajd by ip2
func (d *DataAccess) GetAdjacenciesByIP2(ip uint32) ([]dm.Adjacency, error) {
	return d.db.GetAdjacenciesByIP2(ip)
}

// GetAdjacencyToDestByAddrAndDest24 returns adjstodest based on dest24 and addr
func (d *DataAccess) GetAdjacencyToDestByAddrAndDest24(dest24, addr uint32) ([]dm.AdjacencyToDest, error) {
	return d.db.GetAdjacencyToDestByAddrAndDest24(dest24, addr)
}

// StoreAlias stores an alias with id id
func (d *DataAccess) StoreAlias(id int, ips []net.IP) error {
	return d.db.StoreAlias(id, ips)
}

// GetClusterIDByIP gets the cluster id for a given ip
func (d *DataAccess) GetClusterIDByIP(ip uint32) (int, error) {
	return d.db.GetClusterIDByIP(ip)
}

// GetIPsForClusterID gets all the ips associated with the given cluster id
func (d *DataAccess) GetIPsForClusterID(id int) ([]uint32, error) {
	return d.db.GetIPsForClusterID(id)
}

var (
	// ErrNoRevtrUserFound is returned when no user is found with the given key
	ErrNoRevtrUserFound = sql.ErrNoRevtrUserFound
	// ErrCannotAddRevtrBatch is returned when a batch cannot be added
	ErrCannotAddRevtrBatch = sql.ErrCannotAddRevtrBatch
)

// GetUserByKey gets a reverse traceroute user with the given key
func (d *DataAccess) GetUserByKey(key string) (dm.RevtrUser, error) {
	return d.db.GetUserByKey(key)
}

// CreateRevtrBatch creatse a batch of revtrs if the user identified by id
// is allowed to issue more reverse traceroutes
func (d *DataAccess) CreateRevtrBatch(batch []dm.RevtrMeasurement, id string) ([]dm.RevtrMeasurement, uint32, error) {
	return d.db.CreateRevtrBatch(batch, id)
}

// GetRevtrsInBatch gets the reverse traceroutes in batch bid
func (d *DataAccess) GetRevtrsInBatch(uid, bid uint32) ([]*dm.ReverseTraceroute, error) {
	return d.db.GetRevtrsInBatch(uid, bid)
}

// StoreBatchedRevtrs stores the results of a batch of revtrs
// this means updating the initial entries and adding in hops
func (d *DataAccess) StoreBatchedRevtrs(batch []dm.ReverseTraceroute) error {
	return d.db.StoreBatchedRevtrs(batch)
}

// StoreRevtr stores a revtr
func (d *DataAccess) StoreRevtr(r dm.ReverseTraceroute) error {
	return d.db.StoreRevtr(r)
}

// New create a new dataAccess with the given config
func New(c DbConfig) (*DataAccess, error) {
	var conf sql.DbConfig
	for _, cc := range c.ReadConfigs {
		conf.ReadConfigs = append(conf.ReadConfigs, sql.Config{
			User:     cc.User,
			Password: cc.Password,
			Host:     cc.Host,
			Port:     cc.Port,
			Db:       cc.Db,
		})
	}
	for _, cc := range c.WriteConfigs {
		conf.WriteConfigs = append(conf.WriteConfigs, sql.Config{
			User:     cc.User,
			Password: cc.Password,
			Host:     cc.Host,
			Port:     cc.Port,
			Db:       cc.Db,
		})
	}
	db, err := sql.NewDB(conf)
	if err != nil {
		return nil, err
	}
	return &DataAccess{conf: c, db: db}, nil
}
