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

// Package sql provides a sql data provider for reverse traceroute
package sql

import (
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	da "github.com/NEU-SNS/ReverseTraceroute/dataaccess"
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
)
import "github.com/go-sql-driver/mysql"

// DB represents a database collection
type DB struct {
	wdb []*sql.DB
	rdb []*sql.DB
	rr  *rand.Rand
	wr  *rand.Rand
}

// DbConfig is the database config
type DbConfig struct {
	WriteConfigs []Config
	ReadConfigs  []Config
}

// Config is the configuration for an indivual database
type Config struct {
	User     string
	Password string
	Host     string
	Port     string
	Db       string
}

var conFmt = "%s:%s@tcp(%s:%s)/%s?parseTime=true"

func makeDb(conf Config) (*sql.DB, error) {
	conString := fmt.Sprintf(conFmt, conf.User, conf.Password, conf.Host, conf.Port, conf.Db)
	db, err := sql.Open("mysql", conString)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(24)
	return db, nil
}

// NewDB creates a new DB with the given config
func NewDB(con DbConfig) (*DB, error) {
	ret := &DB{}
	ret.rr = rand.New(rand.NewSource(time.Now().UnixNano()))
	ret.wr = rand.New(rand.NewSource(time.Now().UnixNano()))
	for _, conf := range con.WriteConfigs {
		db, err := makeDb(conf)
		if err != nil {
			return nil, err
		}
		ret.wdb = append(ret.wdb, db)
	}
	for _, conf := range con.ReadConfigs {
		db, err := makeDb(conf)
		if err != nil {
			return nil, err
		}
		ret.rdb = append(ret.rdb, db)
	}
	return ret, nil
}

func (db *DB) getReader() *sql.DB {
	l := len(db.rdb)
	if l == 1 {
		return db.rdb[0]
	}
	return db.rdb[db.rr.Intn(len(db.rdb))]
}

func (db *DB) getWriter() *sql.DB {
	l := len(db.wdb)
	if l == 1 {
		return db.wdb[0]
	}
	return db.wdb[db.wr.Intn(len(db.wdb))]
}

// VantagePoint represents a vantage point
type VantagePoint struct {
	IP           uint32
	Controller   sql.NullInt64
	HostName     string
	Site         string
	TimeStamp    bool
	RecordRoute  bool
	CanSpoof     bool
	Active       bool
	ReceiveSpoof bool
	LastUpdated  time.Time
	SpoofChecked mysql.NullTime
	Port         uint32
}

// ToDataModel converts a sql.VantagePoint to a dm.VantagePoint
func (vp *VantagePoint) ToDataModel() *dm.VantagePoint {
	nvp := &dm.VantagePoint{}
	nvp.Ip = vp.IP
	if vp.Controller.Valid {
		nvp.Controller = uint32(vp.Controller.Int64)
	}
	nvp.Hostname = vp.HostName
	nvp.Timestamp = vp.TimeStamp
	nvp.RecordRoute = vp.TimeStamp
	nvp.CanSpoof = vp.CanSpoof
	nvp.ReceiveSpoof = vp.ReceiveSpoof
	nvp.LastUpdated = vp.LastUpdated.Unix()
	nvp.Site = vp.Site
	if vp.SpoofChecked.Valid {
		nvp.SpoofChecked = vp.LastUpdated.Unix()
	}
	nvp.Port = vp.Port
	return nvp
}

const (
	getVpsQuery string = `
SELECT
	ip, controller, hostname, timestamp,
	record_route, can_spoof,
    receive_spoof, last_updated, port
FROM
	vantage_point;
`
)

// GetVPs gets all the VPs in the database
func (db *DB) GetVPs() ([]*dm.VantagePoint, error) {
	rows, err := db.getReader().Query(getVpsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var vps []*dm.VantagePoint
	for rows.Next() {
		vp := &VantagePoint{}
		err := rows.Scan(
			&vp.IP,
			&vp.Controller,
			&vp.HostName,
			&vp.TimeStamp,
			&vp.RecordRoute,
			&vp.CanSpoof,
			&vp.ReceiveSpoof,
			&vp.LastUpdated,
			&vp.Port,
		)
		if err != nil {
			return vps, err
		}
		vps = append(vps, vp.ToDataModel())
	}
	err = rows.Err()
	return vps, err
}

const (
	updateControllerQuery string = `
UPDATE
	vantage_point
SET
	controller = ?
WHERE
	ip = ?
`
	updateControllerQueryNull string = `
UPDATE
	vantage_point
SET
	controller = IF(controller = ?, NULL, controller)
WHERE
	ip = ?
`
)

// UpdateController updates a vantage point's controller
func (db *DB) UpdateController(ip, newc, con uint32) error {
	var args []interface{}
	query := updateControllerQuery
	if newc == 0 {
		query = updateControllerQueryNull
		args = append(args, con)
	} else {
		args = append(args, newc)
	}
	args = append(args, ip)
	_, err := db.getWriter().Exec(query, args...)
	return err
}

const (
	updateActiveQuery string = `
UPDATE
	vantage_point
SET
	active = ?
WHERE
	ip = ?
`
)

// UpdateActive updates the acive flag of a vantage point
func (db *DB) UpdateActive(ip uint32, active bool) error {
	_, err := db.getWriter().Exec(updateActiveQuery, active, ip)
	return err
}

const (
	updateCanSpoofQuery string = `
UPDATE
	vantage_point
SET
	can_spoof = ?,
	spoof_checked = NOW()
WHERE
	ip = ?
`
)

// UpdateCanSpoof updates the can spoof flag for a vantage point
func (db *DB) UpdateCanSpoof(ip uint32, canSpoof bool) error {
	_, err := db.getWriter().Exec(updateCanSpoofQuery, canSpoof, ip)
	return err
}

const (
	updateCheckStatus string = `
UPDATE
	vantage_point
SET
	last_health_check = ?
WHERE
	ip = ?
`
)

// UpdateCheckStatus updates the result of the health check for a vantage point
func (db *DB) UpdateCheckStatus(ip uint32, result string) error {
	_, err := db.getReader().Exec(updateCheckStatus, result, ip)
	return err
}

// Close closes the DB connections
func (db *DB) Close() error {
	for _, d := range db.wdb {
		d.Close()
	}
	for _, d := range db.rdb {
		d.Close()
	}
	return nil
}

const (
	insertTrace = `
INSERT INTO
traceroutes(src, dst, type, user_id, method, sport, 
			dport, stop_reason, stop_data, start, 
			version, hop_count, attempts, hop_limit, 
			first_hop, wait, wait_probe, tos, probe_size)
VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`
	insertTraceHop = `
INSERT INTO
traceroute_hops(traceroute_id, hop, addr, probe_ttl, probe_id, 
				probe_size, rtt, reply_ttl, reply_tos, reply_size, 
				reply_ipid, icmp_type, icmp_code, icmp_q_ttl, icmp_q_ipl, icmp_q_tos)
VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`
)

func storeTraceroute(tx *sql.Tx, in *dm.Traceroute) (int64, error) {
	start := time.Unix(in.Start.Sec, in.Start.Usec*1000)
	res, err := tx.Exec(insertTrace, in.Src, in.Dst, in.Type,
		in.UserId, in.Method, in.Sport, in.Dport,
		in.StopReason, in.StopData, start,
		in.Version, in.HopCount, in.Attempts,
		in.Hoplimit, in.Firsthop, in.Wait, in.WaitProbe,
		in.Tos, in.ProbeSize)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func storeTraceHop(tx *sql.Tx, id int64, hop uint32, in *dm.TracerouteHop) error {
	rtt := in.GetRtt()
	var inRtt uint32
	if rtt != nil {
		/*
			I would generally say this cast is not safe, but we know that it will fit in
			a uint32 because that's what we got it from
		*/
		inRtt = uint32(rtt.Sec)*1000000 + uint32(rtt.Usec)
	}
	_, err := tx.Exec(insertTraceHop, id, hop, in.Addr, in.ProbeTtl,
		in.ProbeId, in.ProbeSize, inRtt, in.ReplyTtl,
		in.ReplyTos, in.ReplySize, in.ReplyIpid,
		in.IcmpType, in.IcmpCode, in.IcmpQTtl,
		in.IcmpQIpl, in.IcmpQTos)
	return err
}

// StoreTraceroute saves a traceroute to the DB
func (db *DB) StoreTraceroute(in *dm.Traceroute) error {
	conn := db.getWriter()
	tx, err := conn.Begin()
	if err != nil {
		return err
	}
	id, err := storeTraceroute(tx, in)
	if err != nil {
		tx.Rollback()
		return err
	}
	for i, hop := range in.GetHops() {
		err = storeTraceHop(tx, id, uint32(i), hop)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// GetTRBySrcDst gets traceroutes with the given src, dst
func (db *DB) GetTRBySrcDst(src, dst uint32) (<-chan *dm.Traceroute, error) {
	return nil, nil
}

// GetTRBySrcDstWithStaleness gets a traceroute with the src/dst this is newer than s
func (db *DB) GetTRBySrcDstWithStaleness(src, dst uint32, s da.Staleness) (<-chan *dm.Traceroute, error) {
	return nil, nil
}

// GetTraceMulti gets traceroutes that match the given TracerouteMeasurements
func (db *DB) GetTraceMulti(in []*dm.TracerouteMeasurement) (<-chan *dm.Traceroute, error) {
	return nil, nil
}

// GetPingsMulti gets pings that match the given PingMeasurements
func (db *DB) GetPingsMulti(in []*dm.PingMeasurement) (<-chan *dm.Ping, error) {
	return nil, nil
}

// GetPingBySrcDst gets pings with the given src/dst
func (db *DB) GetPingBySrcDst(src, dst uint32) (<-chan *dm.Ping, error) {
	return nil, nil
}

// GetPingBySrcDstWithStaleness gets a ping with the src/dst that is newer than s
func (db *DB) GetPingBySrcDstWithStaleness(src, dst uint32, s da.Staleness) (<-chan *dm.Ping, error) {
	return nil, nil
}

const (
	insertPing = `
INSERT INTO
pings(src, dst, start, ping_sent, probe_size, 
	  user_id, ttl, wait, spoofed_from, version, 
	  spoofed, record_route, payload, tsonly, tsandaddr, icmpsum, dl, 8)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

	insertPingResponse = `
INSERT INTO
ping_responses(ping_id, from, seq, reply_size, reply_ttl, 
			   reply_proto, rtt, probe_ipid, reply_ipid, 
			   icmp_type, icmp_code, tx, rx)
VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`
	insertRR = `
INSERT INTO
record_routes(response_id, hop, ip)
VALUES(?, ?, ?)
`
	insertTS = `
INSERT INTO
timestamps(response_id, order, ts)
VALUES(?, ?, ?)
`
	insertTSADDR = `
INSERT INTO 
timestamp_addrs(response_id, order, ip, ts)
VALUES(?, ?, ?, ?)
`
)

func storePing(tx *sql.Tx, in *dm.Ping) (int64, error) {
	/*
		These are the posible keys for the map

		"v4rr"
		"spoof"
		"payload"
		"tsonly"
		"tsandaddr"
		"icmpsum"
		"dl"
		"8"
	*/

	start := time.Unix(in.Start.Sec, in.Start.Usec*1000)
	flags := make(map[string]bool)
	for _, flag := range in.Flags {
		flags[flag] = true
	}
	res, err := tx.Exec(insertPing, in.Src, in.Dst, start, in.PingSent, in.ProbeSize,
		in.Userid, in.Ttl, in.Wait, in.SpoofedFrom, in.Version,
		flags["spoof"], flags["v4rr"], flags["payload"], flags["tsonly"],
		flags["tsandaddr"], flags["icmpsum"], flags["dl"], flags["8"])

	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func storePingRR(tx *sql.Tx, id int64, rr uint32, hop int8) error {
	_, err := tx.Exec(insertRR, id, hop, rr)
	return err
}

type tstamp uint32

func (ts tstamp) ToTimeUTC() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, int(ts)*1000, time.UTC)
}

func storeTS(tx *sql.Tx, id int64, order int8, ts tstamp) error {
	_, err := tx.Exec(insertTS, id, order, ts.ToTimeUTC())
	return err
}

func storeTSAndAddr(tx *sql.Tx, id int64, order int8, ts *dm.TsAndAddr) error {
	_, err := tx.Exec(insertTSADDR, id, order, tstamp(ts.Ts).ToTimeUTC(), ts.Ip)
	return err
}

func storePingResponse(trx *sql.Tx, id int64, r *dm.PingResponse) error {
	if id == 0 || r == nil {
		return fmt.Errorf("Invalid parameter: storePingResponse")
	}
	tx := time.Unix(r.Tx.Sec, r.Tx.Usec*1000)
	rx := time.Unix(r.Rx.Sec, r.Rx.Usec*1000)
	res, err := trx.Exec(insertPingResponse, id, r.From, r.Seq,
		r.ReplySize, r.ReplyTtl, r.ReplyProto,
		r.Rtt, r.ProbeIpid, r.ReplyIpid,
		r.IcmpType, r.IcmpCode, tx, rx)
	if err != nil {
		return err
	}
	nid, err := res.LastInsertId()
	if err != nil {
		return err
	}
	for i, rr := range r.RR {
		err := storePingRR(trx, nid, rr, int8(i))
		if err != nil {
			return err
		}
	}

	for i, ts := range r.Tsonly {
		storeTS(trx, nid, int8(i), tstamp(ts))
		if err != nil {
			return err
		}
	}
	return nil
}

// StorePing saves a ping to the DB
func (db *DB) StorePing(in *dm.Ping) error {
	conn := db.getWriter()
	tx, err := conn.Begin()
	if err != nil {
		return err
	}
	id, err := storePing(tx, in)
	if err != nil {
		tx.Rollback()
		return err
	}
	for _, pr := range in.GetResponses() {
		err = storePingResponse(tx, id, pr)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}
