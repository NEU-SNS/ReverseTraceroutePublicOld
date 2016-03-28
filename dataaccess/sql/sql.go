/*
Copyright (c) 2015, Northeastern University

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
	"net"
	"time"

	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/util"
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

var conFmt = "%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Local"

func makeDb(conf Config) (*sql.DB, error) {
	conString := fmt.Sprintf(conFmt, conf.User, conf.Password, conf.Host, conf.Port, conf.Db)
	db, err := sql.Open("mysql", conString)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	if err = db.Ping(); err != nil {
		log.Error(err)
		return nil, err
	}
	db.SetMaxOpenConns(24)
	db.SetMaxIdleConns(4)
	db.SetConnMaxLifetime(time.Hour)
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

/*
	getReader and getWriter are fine because
	we just get pointers to thread safe sql.DBs
*/

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
    receive_spoof, last_updated, port, site
FROM
	vantage_point;
`
	getActiveVpsQuery string = `
SELECT
	ip, controller, hostname, timestamp,
	record_route, can_spoof,
    receive_spoof, last_updated, port, site
FROM
	vantage_point
WHERE
	controller is not null;
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
			&vp.Site,
		)
		if err != nil {
			return vps, err
		}
		vps = append(vps, vp.ToDataModel())
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return vps, nil
}

// GetActiveVPs gets all the VPs in the database that are connected
func (db *DB) GetActiveVPs() ([]*dm.VantagePoint, error) {
	rows, err := db.getReader().Query(getActiveVpsQuery)
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
			&vp.Site,
		)
		if err != nil {
			return vps, err
		}
		vps = append(vps, vp.ToDataModel())
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return vps, nil
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
	clearAllVps string = `
UPDATE
        vantage_point
SET
        controller = NULL
`
)

func (db *DB) ClearAllVPs() error {
	_, err := db.getWriter().Exec(clearAllVps)
	if err != nil {
		return err
	}
	return nil
}

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
VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`
	insertTraceHop = `
INSERT INTO
traceroute_hops(traceroute_id, hop, addr, probe_ttl, probe_id, 
				probe_size, rtt, reply_ttl, reply_tos, reply_size, 
				reply_ipid, icmp_type, icmp_code, icmp_q_ttl, icmp_q_ipl, icmp_q_tos)
VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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

const (
	getTraceBySrcDst = `
	SELECT 
	t.id, src, dst, type, user_id, method, sport, dport, stop_reason, stop_data, start, version,
	hop_count, attempts, hop_limit, first_hop, wait, wait_probe, tos, t.probe_size, traceroute_id, 
	hop, addr, probe_ttl, probe_id, th.probe_size, rtt, reply_ttl, reply_tos, reply_size, 
	reply_ipid, icmp_type, icmp_code, icmp_q_ttl, icmp_q_ipl, icmp_q_tos
	FROM
	(
		SELECT 
			*
		FROM
			traceroutes tt
		WHERE
			tt.src = ? and tt.dst = ?
		ORDER BY
			tt.start DESC
		LIMIT 1
	) t left outer join
	traceroute_hops th on th.traceroute_id = t.id
	ORDER BY
	t.start DESC
`
	getTraceBySrcDstStale = `
SELECT 
	t.id, src, dst, type, user_id, method, sport, dport, stop_reason, stop_data, start, version,
    hop_count, attempts, hop_limit, first_hop, wait, wait_probe, tos, t.probe_size, traceroute_id, 
    hop, addr, probe_ttl, probe_id, th.probe_size, rtt, reply_ttl, reply_tos, reply_size, 
	reply_ipid, icmp_type, icmp_code, icmp_q_ttl, icmp_q_ipl, icmp_q_tos
FROM
	(
		SELECT 
			*
		FROM
			traceroutes tt
		WHERE
			tt.src = ? and tt.dst = ?
		ORDER BY
			tt.start DESC
		LIMIT 1
	) t left outer join
	traceroute_hops th on th.traceroute_id = t.id
WHERE t.start > ?
ORDER BY
	t.start DESC
`
)

func splitTraces(rows *sql.Rows) ([]*dm.Traceroute, error) {
	currTraces := make(map[int64]*dm.Traceroute)
	currHops := make(map[int64][]*dm.TracerouteHop)
	for rows.Next() {
		curr := &dm.Traceroute{}
		hop := &dm.TracerouteHop{}
		var start time.Time
		var id int64
		var tID sql.NullInt64
		var hopNum, rtt sql.NullInt64
		rows.Scan(&id, &curr.Src, &curr.Dst, &curr.Type, &curr.UserId, &curr.Method, &curr.Sport,
			&curr.Dport, &curr.StopReason, &curr.StopData, &start, &curr.Version, &curr.HopCount,
			&curr.Attempts, &curr.Hoplimit, &curr.Firsthop, &curr.Wait, &curr.WaitProbe, &curr.Tos,
			&curr.ProbeSize, &tID, &hopNum, &hop.Addr, &hop.ProbeTtl, &hop.ProbeId, &hop.ProbeSize,
			&rtt, &hop.IcmpCode, &hop.IcmpQTtl, &hop.IcmpQIpl, &hop.IcmpQTos)
		if _, ok := currTraces[id]; !ok {
			curr.Start = &dm.TracerouteTime{}
			nano := start.UnixNano()
			curr.Start.Sec = nano * 1000000000
			curr.Start.Usec = (nano % 1000000000) / 1000
			currTraces[id] = curr
		}
		if tID.Valid {
			currHops[tID.Int64] = append(currHops[tID.Int64], hop)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var ret []*dm.Traceroute
	for key, val := range currTraces {
		if hops, ok := currHops[key]; ok {
			val.Hops = hops
		}
		ret = append(ret, val)
	}
	return ret, nil
}

// GetTRBySrcDst gets traceroutes with the given src, dst
func (db *DB) GetTRBySrcDst(src, dst uint32) ([]*dm.Traceroute, error) {
	rows, err := db.getReader().Query(getTraceBySrcDst, src, dst)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return splitTraces(rows)
}

// GetTRBySrcDstWithStaleness gets a traceroute with the src/dst this is newer than s
func (db *DB) GetTRBySrcDstWithStaleness(src, dst uint32, s time.Duration) ([]*dm.Traceroute, error) {
	minTime := time.Now().Add(-s)
	rows, err := db.getReader().Query(getTraceBySrcDstStale, src, dst, minTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return splitTraces(rows)
}

// GetTraceMulti gets traceroutes that match the given TracerouteMeasurements
func (db *DB) GetTraceMulti(in []*dm.TracerouteMeasurement) ([]*dm.Traceroute, error) {
	var ret []*dm.Traceroute
	for _, tm := range in {
		ts, err := db.GetTRBySrcDstWithStaleness(tm.Src, tm.Dst, time.Duration(tm.Staleness)*time.Second)
		if err != nil {
			return nil, err
		}
		ret = append(ret, ts...)
	}
	return ret, nil
}

const (
	getPing = "SELECT " +
		"p.id, p.src, p.dst, p.start, p.ping_sent, p.probe_size, " +
		"p.user_id, p.ttl, p.wait, p.spoofed_from, p.version, p.spoofed," +
		"p.record_route, p.payload, p.tsonly, p.tsandaddr, p.icmpsum, " +
		"p.dl, p.`8`, pr.ping_id, pr.id, pr.`from`, pr.seq, pr.reply_size, " +
		"pr.seq, pr.reply_size, pr.reply_ttl, pr.reply_proto, pr.rtt, " +
		"pr.probe_ipid, pr.icmp_code, pr.icmp_type, pr.tx, pr.rx, ps.ping_id, ps.loss, " +
		"ps.min, ps.max,ps.avg, ps.std_dev, rr.response_id, rr.hop, rr.ip, " +
		"ts.response_id, ts.`order`, ts.ts, taa.response_id, taa.`order`, taa.ip, " +
		"taa.ts " +
		"FROM " +
		"(SELECT * FROM pings WHERE src = ? and dst = ? ORDER BY start DESC LIMIT 1) p left outer join " +
		"ping_responses pr on pr.ping_id = p.id " +
		"left outer join ping_stats ps on ps.ping_id = p.id " +
		"left outer join record_routes rr on rr.response_id = pr.id " +
		"left outer join timestamps ts on ts.response_id = pr.id " +
		"left outer join timestamp_addrs taa on taa.response_id = pr.id;"
	getPingStaleness = "SELECT " +
		"p.id, p.src, p.dst, p.start, p.ping_sent, p.probe_size," +
		"p.user_id, p.ttl, p.wait, p.spoofed_from, p.version, p.spoofed, " +
		"p.record_route, p.payload, p.tsonly, p.tsandaddr, p.icmpsum, " +
		"p.dl, p.`8`, pr.ping_id, pr.id, pr.`from`, pr.seq, pr.reply_size, " +
		"pr.seq, pr.reply_size, pr.reply_ttl, pr.reply_proto, pr.rtt, " +
		"pr.probe_ipid, pr.icmp_code, pr.icmp_type, pr.tx, pr.rx, ps.ping_id, ps.loss, " +
		"ps.min, ps.max,ps.avg, ps.std_dev, rr.response_id, rr.hop, rr.ip, " +
		"ts.response_id, ts.`order`, ts.ts, taa.response_id, taa.`order`, taa.ip, " +
		"taa.ts" +
		"FROM " +
		"(SELECT * FROM pings WHERE src = ? and dst = ? and start > ? ORDER BY start DESC LIMIT 1) p left outer join " +
		"ping_responses pr on pr.ping_id = p.id " +
		"left outer join ping_stats ps on ps.ping_id = p.id " +
		"left outer join record_routes rr on rr.response_id = pr.id " +
		"left outer join timestamps ts on ts.response_id = pr.id " +
		"left outer join timestamp_addrs taa on taa.response_id = pr.id;"
)

type rrHop struct {
	ResponseID sql.NullInt64
	Hop        uint8
	IP         uint32
}

type ts struct {
	ResponseID sql.NullInt64
	Order      uint8
	Ts         time.Time
}

type tsAndAddr struct {
	ResponseID sql.NullInt64
	Order      uint8
	Ts         time.Time
	IP         uint32
}

func makeFlags(spoofed, recordRoute, payload, tsonly, tsandaddr, icmpsum, dl, eight bool) []string {
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
	var ret []string
	if spoofed {
		ret = append(ret, "spoof")
	}
	if recordRoute {
		ret = append(ret, "v4rr")
	}
	if payload {
		ret = append(ret, "payload")
	}
	if tsonly {
		ret = append(ret, "tsonly")
	}
	if tsandaddr {
		ret = append(ret, "tsandaddr")
	}
	if icmpsum {
		ret = append(ret, "icmpsum")
	}
	if dl {
		ret = append(ret, "dl")
	}
	if eight {
		ret = append(ret, "8")
	}
	return ret
}

func splitPings(rows *sql.Rows) ([]*dm.Ping, error) {
	currPings := make(map[int64]*dm.Ping)
	currResponses := make(map[int64]*dm.PingResponse)
	currRR := make(map[int64][]uint32)
	currTS := make(map[int64][]uint32)
	currTSA := make(map[int64][]*dm.TsAndAddr)
	currStats := make(map[int64]*dm.PingStats)
	for rows.Next() {
		p := &dm.Ping{}
		ps := &dm.PingStats{}
		pr := &dm.PingResponse{}
		prr := &rrHop{}
		pts := &ts{}
		ptsa := &tsAndAddr{}
		var pID, prpID, prID, statsID sql.NullInt64
		var rtt uint32
		var start, tx, rx time.Time
		var spoofed, recordRoute, payload, tsonly, tsandaddr, icmpsum, dl, eight bool
		rows.Scan(&pID, &p.Src, &p.Dst, &start, &p.PingSent, &p.ProbeSize,
			&p.UserId, &p.Ttl, &p.Wait, &p.SpoofedFrom, &p.Version,
			&spoofed, &recordRoute, &payload, &tsonly, &tsandaddr, &icmpsum, &dl,
			&eight, &prpID, &prID, &pr.From, &pr.Seq, &pr.ReplySize, &pr.ReplyTtl,
			&pr.ReplyProto, &rtt, &pr.ProbeIpid, &pr.IcmpCode, &pr.IcmpType,
			&tx, &rx, &statsID, &ps.Loss, &ps.Min, &ps.Max, &ps.Avg, &ps.Stddev,
			&prr.ResponseID, &prr.Hop, &prr.IP, &pts.ResponseID, &pts.Order, &pts.Ts,
			&ptsa.ResponseID, &ptsa.Order, &ptsa.IP, &ptsa.Ts)
		if _, ok := currPings[pID.Int64]; !ok && pID.Valid {
			p.Start = &dm.Time{}
			nano := start.UnixNano()
			p.Start.Sec = nano * 1000000000
			p.Start.Usec = (nano % 1000000000) / 1000
			p.Flags = makeFlags(spoofed, recordRoute, payload, tsonly, tsandaddr, icmpsum, dl, eight)
			currPings[prID.Int64] = p
		}
		if prpID.Valid {
			var trx, ttx dm.Time
			tnano := tx.UnixNano()
			rnano := rx.UnixNano()
			trx.Sec = rnano * 1000000000
			trx.Usec = (rnano % 1000000000) / 1000
			ttx.Sec = tnano * 1000000000
			ttx.Usec = (tnano % 1000000000) / 1000
			pr.Tx = &ttx
			pr.Rx = &trx
			currResponses[prID.Int64] = pr
			currPings[prpID.Int64].Responses = append(currPings[prpID.Int64].Responses, pr)
		}
		if statsID.Valid {
			if _, ok := currStats[statsID.Int64]; !ok {
				currStats[statsID.Int64] = ps
			}
		}
		if prr.ResponseID.Valid {
			id := prr.ResponseID.Int64
			currRR[id] = append(currRR[id], prr.IP)
		}
		if pts.ResponseID.Valid {
			id := pts.ResponseID.Int64
			midNightUtc := time.Date(pts.Ts.Year(), pts.Ts.Month(), pts.Ts.Day(), 0, 0, 0, 0, time.UTC)
			timeSinceUtc := uint32(pts.Ts.Sub(midNightUtc).Seconds())
			currTS[id] = append(currTS[id], timeSinceUtc)
		}
		if ptsa.ResponseID.Valid {
			var use dm.TsAndAddr
			id := pts.ResponseID.Int64
			use.Ip = ptsa.IP
			midNightUtc := time.Date(ptsa.Ts.Year(), ptsa.Ts.Month(), ptsa.Ts.Day(), 0, 0, 0, 0, time.UTC)
			timeSinceUtc := uint32(pts.Ts.Sub(midNightUtc).Seconds())
			use.Ts = timeSinceUtc
			currTSA[id] = append(currTSA[id], &use)
		}

	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var ret []*dm.Ping
	for id, stats := range currStats {
		currPings[id].Statistics = stats
	}
	for id, rr := range currRR {
		currResponses[id].RR = rr
	}
	for id, ts := range currTS {
		currResponses[id].Tsonly = ts
	}
	for id, tsanda := range currTSA {
		currResponses[id].Tsandaddr = tsanda
	}
	for _, p := range currPings {
		ret = append(ret, p)
	}
	return ret, nil
}

// GetPingsMulti gets pings that match the given PingMeasurements
func (db *DB) GetPingsMulti(in []*dm.PingMeasurement) ([]*dm.Ping, error) {
	var ret []*dm.Ping
	for _, pm := range in {
		var stale int64
		if pm.Staleness == 0 {
			stale = 60
		}
		ps, err := db.GetPingBySrcDstWithStaleness(pm.Src, pm.Dst, time.Duration(stale)*time.Second)
		if err != nil {
			return nil, err
		}
		ret = append(ret, ps...)
	}
	return ret, nil
}

// GetPingBySrcDst gets pings with the given src/dst
func (db *DB) GetPingBySrcDst(src, dst uint32) ([]*dm.Ping, error) {
	rows, err := db.getReader().Query(getPing, src, dst)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return splitPings(rows)
}

// GetPingBySrcDstWithStaleness gets a ping with the src/dst that is newer than s
func (db *DB) GetPingBySrcDstWithStaleness(src, dst uint32, s time.Duration) ([]*dm.Ping, error) {
	minTime := time.Now().Add(-s)
	rows, err := db.getReader().Query(getPing, src, dst, minTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return splitPings(rows)
}

const (
	insertPing = "INSERT INTO" +
		" pings(src, dst, start, ping_sent, probe_size," +
		" user_id, ttl, wait, spoofed_from, version," +
		" spoofed, record_route, payload, tsonly, tsandaddr, icmpsum, dl, `8`)" +
		" VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"

	insertPingResponse = "INSERT INTO " +
		"ping_responses(ping_id, `from`, seq, reply_size, reply_ttl, " +
		"			   reply_proto, rtt, probe_ipid, reply_ipid, " +
		"icmp_type, icmp_code, tx, rx) " +
		"VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) "

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
	insertPingStats = `
INSERT INTO
ping_stats(ping_id, loss, min, max, avg, std_dev)
VALUES(?, ?, ?, ?, ?, ?)
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
	var start time.Time
	if in.Start == nil {
		start = time.Now()
	} else {
		start = time.Unix(in.Start.Sec, in.Start.Usec*1000)
	}
	flags := make(map[string]byte)
	for _, flag := range in.Flags {
		flags[flag] = 1
	}
	res, err := tx.Exec(insertPing, in.Src, in.Dst, start, in.PingSent, in.ProbeSize,
		in.UserId, in.Ttl, in.Wait, in.SpoofedFrom, in.Version,
		flags["spoof"], flags["v4rr"], flags["payload"], flags["tsonly"],
		flags["tsandaddr"], flags["icmpsum"], flags["dl"], flags["8"])

	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func storePingStats(tx *sql.Tx, id int64, stat *dm.PingStats) error {
	if stat == nil {
		return nil
	}
	_, err := tx.Exec(insertPingStats, id, stat.Loss, stat.Min, stat.Max, stat.Avg, stat.Stddev)
	return err
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
	err = storePingStats(tx, id, in.GetStatistics())
	return tx.Commit()
}

const (
	findIntersecting = `
SELECT 
	? as src, A.dest, hops.hop, hops.ttl
FROM 
(
SELECT
	*
FROM
(
(SELECT atr.Id, atr.date, atr.dest FROM
atlas_traceroutes atr 
WHERE atr.dest = ? AND atr.date >= DATE_SUB(NOW(), interval ?  minute) 
ORDER BY atr.date desc)
) X 
INNER JOIN atlas_traceroute_hops ath on ath.trace_id = X.Id
INNER JOIN
(
SELECT ? IP
UNION
SELECT
		b.ip_address
	FROM
		ip_aliases a INNER JOIN ip_aliases b on a.cluster_id = b.cluster_id
	WHERE
		a.ip_address = ?	
) Z ON ath.hop = Z.IP
ORDER BY date desc
limit 1
) A
INNER JOIN atlas_traceroute_hops hops on hops.trace_id = A.Id
ORDER BY hops.ttl
`
	findIntersectingIgnoreSource = `
SELECT 
	? as src, A.dest, hops.hop, hops.ttl
FROM 
(
SELECT
	*
FROM
(
(SELECT atr.Id, atr.date, atr.dest FROM
atlas_traceroutes atr 
WHERE  atr.src != ? AND atr.dest = ? AND atr.date >= DATE_SUB(NOW(), interval ?  minute) 
ORDER BY atr.date desc)
) X 
INNER JOIN atlas_traceroute_hops ath on ath.trace_id = X.Id
INNER JOIN
(
SELECT ? IP
UNION
SELECT
		b.ip_address
	FROM
		ip_aliases a INNER JOIN ip_aliases b on a.cluster_id = b.cluster_id
	WHERE
		a.ip_address = ?	
) Z ON ath.hop = Z.IP
ORDER BY date desc
limit 1
) A
INNER JOIN atlas_traceroute_hops hops on hops.trace_id = A.Id
ORDER BY hops.ttl
`
	getSources = `
SELECT
    src
FROM
    atlas_traceroutes
WHERE
    dest = ? AND date >= DATE_SUB(NOW(), interval ? minute);
`
)

type hopRow struct {
	src  uint32
	dest uint32
	hop  uint32
	ttl  uint32
}

var (
	// ErrNoIntFound is returned when no intersection is found
	ErrNoIntFound = fmt.Errorf("No Intersection Found")
)

// GetAtlasSources gets all sources that were used for existing atlas traceroutes
// the set of vps - this would be the sources to use to run traces
func (db *DB) GetAtlasSources(dst uint32, stale time.Duration) ([]uint32, error) {
	rows, err := db.getReader().Query(getSources, dst, int64(stale.Minutes()))
	var srcs []uint32
	if err != nil {
		log.Error(err)
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var curr uint32
		rows.Scan(&curr)
		srcs = append(srcs, curr)
	}
	if err = rows.Err(); err != nil {
		log.Error(err)
		return nil, err
	}
	return srcs, nil
}

// FindIntersectingTraceroute finds a traceroute that intersects hop towards the dst
func (db *DB) FindIntersectingTraceroute(pairs []dm.SrcDst) ([]*dm.Path, error) {
	log.Debug("Finding intersecting traceroute ", pairs)
	pair := pairs[0]
	var rows *sql.Rows
	var err error
	if pair.IgnoreSource {
		rows, err = db.getReader().Query(findIntersectingIgnoreSource, pair.Addr, pair.Src, pair.Dst, int64(pair.Stale.Minutes()), pair.Addr, pair.Addr)
		if err != nil {
			log.Error(err)
			return nil, err
		}
	} else {
		rows, err = db.getReader().Query(findIntersecting, pair.Addr, pair.Dst, int64(pair.Stale.Minutes()), pair.Addr, pair.Addr)
		if err != nil {
			log.Error(err)
			return nil, err
		}
	}
	defer rows.Close()
	ret := dm.Path{}
	for rows.Next() {
		row := hopRow{}
		rows.Scan(&row.src, &row.dest, &row.hop, &row.ttl)
		ret.Hops = append(ret.Hops, &dm.Hop{
			Ip:  row.hop,
			Ttl: row.ttl,
		})
		ret.Address = row.src
	}
	if err := rows.Err(); err != nil {
		log.Error(err)
		return nil, err
	}
	if len(ret.Hops) == 0 {
		return nil, ErrNoIntFound
	}
	return []*dm.Path{&ret}, nil
}

const (
	insertAtlasTrace = `INSERT INTO atlas_traceroutes(dest, src) VALUES(?, ?)`
	insertAtlasHop   = `
	INSERT INTO atlas_traceroute_hops(trace_id, hop, ttl) 
	VALUES (?, ?, ?)`
)

// StoreAtlasTraceroute stores a traceroute in a form that the Atlas requires
func (db *DB) StoreAtlasTraceroute(trace *dm.Traceroute) error {
	conn := db.getWriter()
	tx, err := conn.Begin()
	if err != nil {
		return err
	}
	res, err := tx.Exec(insertAtlasTrace, trace.Dst, trace.Src)
	if err != nil {
		tx.Rollback()
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return err
	}
	stmt, err := tx.Prepare(insertAtlasHop)
	if err != nil {
		tx.Rollback()
		return err
	}
	_, err = stmt.Exec(int32(id), trace.Src, 0)
	if err != nil {
		tx.Rollback()
		return err
	}
	for _, hop := range trace.GetHops() {
		_, err := stmt.Exec(int32(id), hop.Addr, hop.ProbeTtl)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	err = stmt.Close()
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

const (
	insertAdjQuery = `INSERT INTO adjacencies(ip1, ip2) VALUES (?, ?)
	ON DUPLICATE KEY UPDATE cnt = cnt+1`
	selectByIP1AdjQuery = `SELECT ip1, ip2, cnt from adjacencies WHERE ip1 = ?
							ORDER BY cnt DESC LIMIT 500`
	selectByIP2AdjQuery = `SELECT ip1, ip2, cnt from adjacencies WHERE ip2 = ?
							ORDER BY cnt DESC LIMIT 500`
)

// StoreAdjacency stores an adjacency
func (db *DB) StoreAdjacency(l, r net.IP) error {
	con := db.getWriter()
	ip1, err := util.IPtoInt32(l)
	if err != nil {
		return err
	}
	ip2, err := util.IPtoInt32(r)
	if err != nil {
		return err
	}
	_, err = con.Exec(insertAdjQuery, ip1, ip2)
	if err != nil {
		return err
	}
	return nil
}

// GetAdjacenciesByIP1 gets ajds by ip1
func (db *DB) GetAdjacenciesByIP1(ip uint32) ([]dm.Adjacency, error) {
	con := db.getReader()
	res, err := con.Query(selectByIP1AdjQuery, ip)
	if err != nil {
		return nil, err
	}
	defer res.Close()
	var adjs []dm.Adjacency
	for res.Next() {
		var adj dm.Adjacency
		err := res.Scan(&adj.IP1, &adj.IP2, &adj.Cnt)
		if err != nil {
			return nil, err
		}
		adjs = append(adjs, adj)
	}
	if err = res.Err(); err != nil {
		return nil, err
	}
	return adjs, nil
}

// GetAdjacenciesByIP2 gets ajds by ip2
func (db *DB) GetAdjacenciesByIP2(ip uint32) ([]dm.Adjacency, error) {
	con := db.getReader()
	res, err := con.Query(selectByIP2AdjQuery, ip)
	if err != nil {
		return nil, err
	}
	defer res.Close()
	var adjs []dm.Adjacency
	for res.Next() {
		var adj dm.Adjacency
		err := res.Scan(&adj.IP1, &adj.IP2, &adj.Cnt)
		if err != nil {
			return nil, err
		}
		adjs = append(adjs, adj)
	}
	if err = res.Err(); err != nil {
		return nil, err
	}
	return adjs, nil
}

const (
	insertAdjDstQuery = `
	INSERT INTO adjacencies_to_dest(dest24, address, adjacent) VALUES(?, ?, ?)
	ON DUPLICATE KEY UPDATE cnt = cnt + 1
	`
	selectByAddressAndDest24AdjDstQuery = `
	SELECT dest24, address, adjacent, cnt
	FROM adjacencies_to_dest 
	WHERE address = ? AND dest24 = ?
	ORDER BY cnt DESC LIMIT 500`
)

// StoreAdjacencyToDest stores an adjacencies to dest
func (db *DB) StoreAdjacencyToDest(dest24, addr, adj net.IP) error {
	con := db.getWriter()
	destip, _ := util.IPtoInt32(dest24)
	destip = destip >> 8
	addrip, _ := util.IPtoInt32(addr)
	adjip, _ := util.IPtoInt32(adj)
	_, err := con.Exec(insertAdjDstQuery, destip, addrip, adjip)
	if err != nil {
		return err
	}
	return nil
}

// GetAdjacencyToDestByAddrAndDest24 does what it says
func (db *DB) GetAdjacencyToDestByAddrAndDest24(dest24, addr uint32) ([]dm.AdjacencyToDest, error) {
	con := db.getReader()
	res, err := con.Query(selectByAddressAndDest24AdjDstQuery, addr, dest24)
	if err != nil {
		return nil, err
	}
	defer res.Close()
	var adjs []dm.AdjacencyToDest
	for res.Next() {
		var adj dm.AdjacencyToDest
		err = res.Scan(&adj.Dest24, &adj.Address, &adj.Adjacent, &adj.Cnt)
		if err != nil {
			return nil, err
		}
		adjs = append(adjs, adj)
	}
	return adjs, nil
}

const (
	removeAliasCluster    = `DELETE FROM ip_aliases WHERE cluster_id = ?`
	storeAlias            = `INSERT INTO ip_aliases(cluster_id, ip_address) VALUES(?, ?)`
	aliasGetByIP          = `SELECT cluster_id FROM ip_aliases WHERE ip_address = ? LIMIT 1`
	aliasGetIPsForCluster = `SELECT ip_address FROM ip_aliases WHERE cluster_id = ? LIMIT 2000`
)

// StoreAlias stores an IP alias
func (db *DB) StoreAlias(id int, ips []net.IP) error {
	con := db.getWriter()
	tx, err := con.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(removeAliasCluster, id)
	if err != nil {
		tx.Rollback()
		return err
	}
	for _, ip := range ips {
		ipint, _ := util.IPtoInt32(ip)
		_, err = tx.Exec(storeAlias, id, ipint)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

var (
	// ErrNoAlias is returned when no alias is found for an ip
	ErrNoAlias = fmt.Errorf("No alias found")
)

// GetClusterIDByIP gets a the cluster ID for a give ip
func (db *DB) GetClusterIDByIP(ip uint32) (int, error) {
	con := db.getReader()
	var ret int
	err := con.QueryRow(aliasGetByIP, ip).Scan(&ret)
	switch {
	case err == sql.ErrNoRows:
		return ret, ErrNoAlias
	case err != nil:
		return ret, err
	default:
		return ret, nil
	}
}

// GetIPsForClusterID gets all IPs associated with the given cluster id
func (db *DB) GetIPsForClusterID(id int) ([]uint32, error) {
	con := db.getReader()
	var scan uint32
	var ret []uint32
	res, err := con.Query(aliasGetByIP, id)
	defer res.Close()
	if err != nil {
		return nil, err
	}
	for res.Next() {
		err := res.Scan(&scan)
		if err != nil {
			return nil, err
		}
		ret = append(ret, scan)
	}
	if err = res.Err(); err != nil {
		return nil, err
	}
	return ret, nil
}
