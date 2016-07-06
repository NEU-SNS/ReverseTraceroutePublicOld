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
	"net"
	"time"

	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/repository"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	"github.com/go-sql-driver/mysql"
)

/*
This is just duplicate code while for the period of transition
*/

// DB is the data access object
type DB struct {
	*repository.DB
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

// NewDB creates a db object with the given config
func NewDB(con DbConfig) (*DB, error) {
	var conf repository.DbConfig
	for _, wc := range con.WriteConfigs {
		conf.WriteConfigs = append(conf.WriteConfigs, repository.Config(wc))
	}
	for _, rc := range con.ReadConfigs {
		conf.ReadConfigs = append(conf.ReadConfigs, repository.Config(rc))
	}
	db, err := repository.NewDB(conf)
	if err != nil {
		return nil, err
	}
	return &DB{db}, nil
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
	rows, err := db.GetReader().Query(getVpsQuery)
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
	rows, err := db.GetReader().Query(getActiveVpsQuery)
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
WHERE
        controller is not null
`
)

func (db *DB) ClearAllVPs() error {
	_, err := db.GetWriter().Exec(clearAllVps)
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
	_, err := db.GetWriter().Exec(query, args...)
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
	_, err := db.GetWriter().Exec(updateActiveQuery, active, ip)
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
	_, err := db.GetWriter().Exec(updateCanSpoofQuery, canSpoof, ip)
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
	_, err := db.GetReader().Exec(updateCheckStatus, result, ip)
	return err
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
func (db *DB) StoreTraceroute(in *dm.Traceroute) (int64, error) {
	conn := db.GetWriter()
	tx, err := conn.Begin()
	if err != nil {
		return 0, err
	}
	id, err := storeTraceroute(tx, in)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	for i, hop := range in.GetHops() {
		err = storeTraceHop(tx, id, uint32(i), hop)
		if err != nil {
			tx.Rollback()
			return 0, err
		}
	}
	return id, tx.Commit()
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
			tt.src = %d and tt.dst = %d
		ORDER BY
			tt.start DESC
		LIMIT 1
	) t left outer join
	traceroute_hops th on th.traceroute_id = t.id
WHERE t.start >= DATE_SUB(NOW(), interval %d minute)
ORDER BY
	t.start DESC
`

	addTraceBatch      = `insert into trace_batch(user_id) VALUES(?)`
	addTraceBatchTrace = `insert into trace_batch_trace(batch_id, trace_id) VALUES(?, ?)`
	getTraceBatch0     = "SELECT t.id, t.src, t.dst, t.type, t.user_id, t.method, t.sport, " +
		"t.dport, t.stop_reason, t.stop_data, t.start, t.version, t.hop_count, t.attempts, " +
		"t.hop_limit, t.first_hop, t.wait, t.wait_probe, t.tos, t.probe_size  " +
		"FROM users u inner join trace_batch tb on tb.user_id = u.id inner join " +
		"inner join trace_batch_trace tbt on tb.id = tbt.batch_id inner join " +
		"traceroutes t on t.id = tbt.trace_id WHERE u.`key` = ? and tb.id = ?;"
	getTraceBatch = "SELECT t.id, t.src, t.dst, t.type, t.user_id, t.method, " +
		"t.sport, t.dport, t.stop_reason, t.stop_data, t.start, t.version, " +
		"t.hop_count, t.attempts, t.hop_limit, t.first_hop, t.wait, t.wait_probe, " +
		"t.tos, t.probe_size, th.traceroute_id, " +
		"th.hop, th.addr, th.probe_ttl, th.probe_id, th.probe_size, " +
		"th.rtt, th.reply_ttl, th.reply_tos, th.reply_size, " +
		"th.reply_ipid, th.icmp_type, th.icmp_code, th.icmp_q_ttl, th.icmp_q_ipl, th.icmp_q_tos " +
		"FROM " +
		"users u " +
		"inner join trace_batch tb on u.id = tb.user_id " +
		"inner join trace_batch_trace tbt on tb.id = tbt.batch_id " +
		"inner join traceroutes t on tbt.trace_id = t.id " +
		"inner join traceroute_hops th on th.traceroute_id = t.id " +
		"WHERE " +
		"u.`key` = ? and tb.id = ?; "
)

// AddTraceBatch adds a traceroute batch
func (db *DB) AddTraceBatch(u dm.User) (int64, error) {
	res, err := db.GetWriter().Exec(addTraceBatch, u.ID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// AddTraceToBatch adds pings pids to batch bid
func (db *DB) AddTraceToBatch(bid int64, tids []int64) error {
	con := db.GetWriter()
	tx, err := con.Begin()
	if err != nil {
		return err
	}
	for _, tid := range tids {
		_, err := tx.Exec(addTraceBatchTrace, bid, tid)
		if err != nil {
			log.Error(err)
			return tx.Rollback()
		}
	}
	return tx.Commit()
}

// GetTraceBatch gets a batch of traceroute for user u with id bid
func (db *DB) GetTraceBatch(u dm.User, bid int64) ([]*dm.Traceroute, error) {
	rows, err := db.GetReader().Query(getTraceBatch, u.Key, bid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return splitTraces(rows)
}

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
		err := rows.Scan(&id, &curr.Src, &curr.Dst, &curr.Type, &curr.UserId, &curr.Method, &curr.Sport,
			&curr.Dport, &curr.StopReason, &curr.StopData, &start, &curr.Version, &curr.HopCount,
			&curr.Attempts, &curr.Hoplimit, &curr.Firsthop, &curr.Wait, &curr.WaitProbe, &curr.Tos,
			&curr.ProbeSize, &tID, &hopNum, &hop.Addr, &hop.ProbeTtl, &hop.ProbeId, &hop.ProbeSize,
			&rtt, &hop.ReplyTtl, &hop.ReplyTos, &hop.ReplySize, &hop.ReplyIpid, &hop.IcmpType,
			&hop.IcmpCode, &hop.IcmpQTtl, &hop.IcmpQIpl, &hop.IcmpQTos)
		if err != nil {
			return nil, err
		}
		if _, ok := currTraces[id]; !ok {
			curr.Start = &dm.TracerouteTime{}
			nano := start.UnixNano()
			curr.Start.Sec = nano / 1000000000
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
	rows, err := db.GetReader().Query(getTraceBySrcDst, src, dst)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return splitTraces(rows)
}

// GetTRBySrcDstWithStaleness gets a traceroute with the src/dst this is newer than s
func (db *DB) GetTRBySrcDstWithStaleness(src, dst uint32, s time.Duration) ([]*dm.Traceroute, error) {
	rows, err := db.GetReader().Query(fmt.Sprintf(getTraceBySrcDstStale, src, dst, int(s.Minutes())))
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
		ts, err := db.GetTRBySrcDstWithStaleness(tm.Src, tm.Dst, time.Duration(tm.Staleness)*time.Minute)
		if err != nil {
			return nil, err
		}
		ret = append(ret, ts...)
	}
	return ret, nil
}

const (
	getPing = "SELECT p.id, p.src, p.dst, p.start, p.ping_sent, " +
		"p.probe_size, p.user_id, p.ttl, p.wait, p.spoofed_from, " +
		"p.version, p.spoofed, p.record_route, p.payload, p.tsonly, " +
		"p.tsandaddr, p.icmpsum, dl, p.`8` " +
		"FROM pings p " +
		"WHERE p.src = ? and p.dst = ?;"
	getPingStaleness = "SELECT p.id, p.src, p.dst, p.start, p.ping_sent, " +
		"p.probe_size, p.user_id, p.ttl, p.wait, p.spoofed_from, " +
		"p.version, p.spoofed, p.record_route, p.payload, p.tsonly, p.tsandaddr, " +
		"p.icmpsum, dl, p.`8` " +
		"FROM pings p " +
		"WHERE p.src = ? and p.dst = ? and p.start >= DATE_SUB(NOW(), interval ? minute);"
	getPingStalenessRR = "SELECT p.id, p.src, p.dst, p.start, p.ping_sent, " +
		"p.probe_size, p.user_id, p.ttl, p.wait, p.spoofed_from, " +
		"p.version, p.spoofed, p.record_route, p.payload, p.tsonly, p.tsandaddr, " +
		"p.icmpsum, dl, p.`8` " +
		"FROM pings p " +
		"WHERE p.src = ? and p.dst = ? and p.record_route and p.start >= DATE_SUB(NOW(), interval ? minute);"
	getPingResponses = "SELECT pr.id, pr.ping_id, pr.`from`, pr.seq, " +
		"pr.reply_size, pr.reply_ttl, pr.rtt, pr.probe_ipid, pr.reply_ipid, " +
		"pr.icmp_type, pr.icmp_code, pr.tx, pr.rx " +
		"FROM ping_responses pr " +
		"WHERE pr.ping_id = ?;"
	getPingStats = "SELECT ps.loss, ps.min, " +
		"ps.max, ps.avg, ps.std_dev " +
		"FROM ping_stats ps " +
		"WHERE ps.ping_id = ?;"
	getRecordRoutes = "SELECT rr.response_id, rr.hop, rr.ip " +
		"FROM record_routes rr " +
		"WHERE rr.response_id = %d ORDER BY rr.hop;"
	getTimeStamps = "SELECT ts.ts " +
		"FROM timestamps ts " +
		"WHERE ts.response_id = %d ORDER BY ts.`order`;"
	getTimeStampsAndAddr = "SELECT tsa.ip, tsa.ts " +
		"FROM timestamp_addrs tsa " +
		"WHERE tsa.response_id = %d ORDER BY tsa.`order`;"
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
	Ts         uint32
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

// GetPingsMulti gets pings that match the given PingMeasurements
func (db *DB) GetPingsMulti(in []*dm.PingMeasurement) ([]*dm.Ping, error) {
	var ret []*dm.Ping
	for _, pm := range in {
		if pm.TimeStamp != "" {
			continue
		}
		var stale int64
		if pm.Staleness == 0 {
			stale = 60
		}
		var ps []*dm.Ping
		var err error
		if pm.RR {
			ps, err = db.getPingSrcDstStaleRR(pm.Src, pm.Dst, time.Duration(stale)*time.Minute)
			if err != nil {
				return nil, err
			}
		} else {
			ps, err = db.GetPingBySrcDstWithStaleness(pm.Src, pm.Dst, time.Duration(stale)*time.Minute)
			if err != nil {
				return nil, err
			}
		}
		ret = append(ret, ps...)
	}
	return ret, nil
}

func getRR(id int64, pr *dm.PingResponse, tx *sql.Tx) error {
	rows, err := tx.Query(fmt.Sprintf(getRecordRoutes, id))
	if err != nil {
		return err
	}
	defer rows.Close()
	var hops []uint32
	for rows.Next() {
		rrhop := new(rrHop)
		err := rows.Scan(&rrhop.ResponseID, &rrhop.Hop, &rrhop.IP)
		if err != nil {
			return err
		}

		hops = append(hops, rrhop.IP)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	pr.RR = hops
	return nil
}
func getTS(id int64, pr *dm.PingResponse, tx *sql.Tx) error {
	rows, err := tx.Query(fmt.Sprintf(getTimeStamps, id))
	if err != nil {
		return err
	}
	defer rows.Close()
	var tss []uint32
	for rows.Next() {
		timestamp := new(uint32)
		err := rows.Scan(timestamp)
		if err != nil {
			return err
		}
		tss = append(tss, *timestamp)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	pr.Tsonly = tss
	return nil
}

func getStats(p *dm.Ping, stmt *sql.Stmt) error {
	row := stmt.QueryRow(p.Id)
	stats := &dm.PingStats{}
	err := row.Scan(&stats.Loss, &stats.Min, &stats.Max, &stats.Avg, &stats.Stddev)
	if err != nil {
		return err
	}
	p.Statistics = stats
	return nil
}

func getTSAndAddr(id int64, pr *dm.PingResponse, tx *sql.Tx) error {
	rows, err := tx.Query(fmt.Sprintf(getTimeStampsAndAddr, id))
	if err != nil {
		return err
	}
	defer rows.Close()
	var tss []*dm.TsAndAddr
	for rows.Next() {
		tsandaddr := new(dm.TsAndAddr)
		err := rows.Scan(&tsandaddr.Ip, &tsandaddr.Ts)
		if err != nil {
			return err
		}
		tss = append(tss, tsandaddr)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	pr.Tsandaddr = tss
	return nil
}

type errorf func() error

func logError(f errorf) {
	if err := f(); err != nil {
		log.Error(err)
	}
}

func getResponses(ping *dm.Ping, tx *sql.Tx, rr, ts, tsaddr bool) error {
	rspstmt, err := tx.Prepare(getPingResponses)
	if err != nil {
		log.Error(err)
		return err
	}
	defer func() {
		if err := rspstmt.Close(); err != nil {
			log.Error(err)
		}
	}()
	rows, err := rspstmt.Query(ping.Id)
	if err != nil {
		return err
	}

	var responses []*dm.PingResponse
	var respIds []int64
	for rows.Next() {
		resp := new(dm.PingResponse)
		var rID, pID sql.NullInt64
		var tx, rx int64
		err := rows.Scan(&rID, &pID, &resp.From, &resp.Seq, &resp.ReplySize,
			&resp.ReplyTtl, &resp.Rtt, &resp.ProbeIpid, &resp.ReplyIpid,
			&resp.IcmpType, &resp.IcmpCode, &tx, &rx)
		if err != nil {
			rows.Close()
			return err
		}
		resp.Tx = &dm.Time{}
		resp.Tx.Sec = tx / 1000000000
		resp.Tx.Usec = (tx % 1000000000) / 1000
		resp.Rx = &dm.Time{}
		resp.Rx.Sec = rx / 1000000000
		resp.Rx.Usec = (rx % 1000000000) / 1000
		ping.Responses = append(ping.Responses, resp)
		respIds = append(respIds, rID.Int64)
		responses = append(responses, resp)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	for i, resp := range responses {
		switch {
		case rr:
			err = getRR(respIds[i], resp, tx)
		case ts:
			err = getTS(respIds[i], resp, tx)
		case tsaddr:
			err = getTSAndAddr(respIds[i], resp, tx)
		}
		if err != nil {
			return err
		}
	}
	ping.Responses = responses
	return nil
}

// GetPingBySrcDst gets pings with the given src/dst
func (db *DB) GetPingBySrcDst(src, dst uint32) ([]*dm.Ping, error) {
	// We only keep 24 hours worth of data in the db
	return db.GetPingBySrcDstWithStaleness(src, dst, time.Hour*24)
}

func (db *DB) getPingSrcDstStaleRR(src, dst uint32, s time.Duration) ([]*dm.Ping, error) {
	tx, err := db.GetReader().Begin()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	defer func() {
		if err := tx.Commit(); err != nil {
			log.Error(err)
		}
	}()
	rows, err := tx.Query(getPingStalenessRR, src, dst, int(s.Minutes()))
	if err != nil {
		log.Error(err)
		return nil, err
	}
	var pings []*dm.Ping
	for rows.Next() {
		p := new(dm.Ping)
		var spoofed, recordRoute, payload, tsonly, tsandaddr, icmpsum, dl, eight bool
		var start int64
		err := rows.Scan(&p.Id, &p.Src, &p.Dst, &start,
			&p.PingSent, &p.ProbeSize, &p.UserId, &p.Ttl,
			&p.Wait, &p.SpoofedFrom, &p.Version, &spoofed,
			&recordRoute, &payload, &tsonly, &tsandaddr, &icmpsum,
			&dl, &eight)
		if err != nil {
			log.Error(err)
			rows.Close()
			return nil, err
		}
		p.Start = &dm.Time{}
		p.Start.Sec = start / 1000000000
		p.Start.Usec = (start % 1000000000) / 1000
		p.Flags = makeFlags(spoofed, recordRoute, payload, tsonly, tsandaddr, icmpsum, dl, eight)
		pings = append(pings, p)
	}
	if err := rows.Err(); err != nil {
		log.Error(err)
		rows.Close()
		return nil, err
	}
	rows.Close()
	statsstmt, err := tx.Prepare(getPingStats)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	defer statsstmt.Close()
	for _, p := range pings {
		var recordRoute, tsonly, tsandaddr bool
		recordRoute = hasFlag(p.Flags, "v4rr")
		tsonly = hasFlag(p.Flags, "tsonly")
		tsandaddr = hasFlag(p.Flags, "tsandaddr")
		err = getResponses(p, tx, recordRoute, tsonly, tsandaddr)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		err = getStats(p, statsstmt)
		if err != nil {
			if err != sql.ErrNoRows {
				return nil, err
			}
		}
	}
	return pings, nil
}
func hasFlag(flags []string, flag string) bool {
	for _, f := range flags {
		if flag == f {
			return true
		}
	}
	return false
}

// GetPingBySrcDstWithStaleness gets a ping with the src/dst that is newer than s
func (db *DB) GetPingBySrcDstWithStaleness(src, dst uint32, s time.Duration) ([]*dm.Ping, error) {
	tx, err := db.GetReader().Begin()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	defer func() {
		if err := tx.Commit(); err != nil {
			log.Error(err)
		}
	}()
	rows, err := tx.Query(getPingStaleness, src, dst, int(s.Minutes()))
	if err != nil {
		log.Error(err)
		return nil, err
	}
	var pings []*dm.Ping
	for rows.Next() {
		p := new(dm.Ping)
		var spoofed, recordRoute, payload, tsonly, tsandaddr, icmpsum, dl, eight bool
		var start int64
		err := rows.Scan(&p.Id, &p.Src, &p.Dst, &start,
			&p.PingSent, &p.ProbeSize, &p.UserId, &p.Ttl,
			&p.Wait, &p.SpoofedFrom, &p.Version, &spoofed,
			&recordRoute, &payload, &tsonly, &tsandaddr, &icmpsum,
			&dl, &eight)
		if err != nil {
			log.Error(err)
			rows.Close()
			return nil, err
		}
		p.Start = &dm.Time{}
		p.Start.Sec = start / 1000000000
		p.Start.Usec = (start % 1000000000) / 1000
		p.Flags = makeFlags(spoofed, recordRoute, payload, tsonly, tsandaddr, icmpsum, dl, eight)
		pings = append(pings, p)
	}
	if err := rows.Err(); err != nil {
		log.Error(err)
		rows.Close()
		return nil, err
	}
	rows.Close()
	statsstmt, err := tx.Prepare(getPingStats)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	defer statsstmt.Close()
	for _, p := range pings {
		var recordRoute, tsonly, tsandaddr bool
		recordRoute = hasFlag(p.Flags, "v4rr")
		tsonly = hasFlag(p.Flags, "tsonly")
		tsandaddr = hasFlag(p.Flags, "tsandaddr")
		err = getResponses(p, tx, recordRoute, tsonly, tsandaddr)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		err = getStats(p, statsstmt)
		if err != nil {
			if err != sql.ErrNoRows {
				return nil, err
			}
		}
	}
	return pings, nil
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
	insertTS = "INSERT INTO timestamps(response_id, `order`, ts) VALUES(?, ?, ?)"

	insertTSADDR = "INSERT INTO timestamp_addrs(response_id, `order`, ip, ts) VALUES(?, ?, ?, ?)"

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
	res, err := tx.Exec(insertPing, in.Src, in.Dst, start.UnixNano(), in.PingSent, in.ProbeSize,
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

func storeTS(tx *sql.Tx, id int64, order int8, ts uint32) error {
	_, err := tx.Exec(insertTS, id, order, ts)
	return err
}

func storeTSAndAddr(tx *sql.Tx, id int64, order int8, ts *dm.TsAndAddr) error {
	_, err := tx.Exec(insertTSADDR, id, order, ts.Ip, ts.Ts)
	return err
}

func storePingResponse(trx *sql.Tx, id int64, r *dm.PingResponse) error {
	if id == 0 || r == nil {
		return fmt.Errorf("Invalid parameter: storePingResponse")
	}
	tx := r.Tx.Sec*1000000000 + r.Tx.Usec*1000
	rx := r.Rx.Sec*1000000000 + r.Rx.Usec*1000
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
		err := storeTS(trx, nid, int8(i), ts)
		if err != nil {
			return err
		}
	}
	for i, ts := range r.Tsandaddr {
		err := storeTSAndAddr(trx, nid, int8(i), ts)
		if err != nil {
			return err
		}
	}
	return nil
}

// StorePing saves a ping to the DB
func (db *DB) StorePing(in *dm.Ping) (int64, error) {
	conn := db.GetWriter()
	tx, err := conn.Begin()
	if err != nil {
		return 0, err
	}
	id, err := storePing(tx, in)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	for _, pr := range in.GetResponses() {
		err = storePingResponse(tx, id, pr)
		if err != nil {
			tx.Rollback()
			return 0, err
		}
	}
	err = storePingStats(tx, id, in.GetStatistics())
	return id, tx.Commit()
}

const (
	insertAdjQuery = `INSERT INTO adjacencies(ip1, ip2) VALUES (?, ?)
	ON DUPLICATE KEY UPDATE cnt = cnt+1`
)

// StoreAdjacency stores an adjacency
func (db *DB) StoreAdjacency(l, r net.IP) error {
	con := db.GetWriter()
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

const (
	insertAdjDstQuery = `
	INSERT INTO adjacencies_to_dest(dest24, address, adjacent) VALUES(?, ?, ?)
	ON DUPLICATE KEY UPDATE cnt = cnt + 1
	`
)

// StoreAdjacencyToDest stores an adjacencies to dest
func (db *DB) StoreAdjacencyToDest(dest24, addr, adj net.IP) error {
	con := db.GetWriter()
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

const (
	removeAliasCluster = `DELETE FROM ip_aliases WHERE cluster_id = ?`
	storeAlias         = `INSERT INTO ip_aliases(cluster_id, ip_address) VALUES(?, ?)`
)

// StoreAlias stores an IP alias
func (db *DB) StoreAlias(id int, ips []net.IP) error {
	con := db.GetWriter()
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

const (
	getUser          = "select * from users where `key` = ?;"
	addPingBatch     = `insert into ping_batch(user_id) VALUES(?)`
	addPingBatchPing = `insert into ping_batch_ping(batch_id, ping_id) VALUES(?, ?)`
	getPingBatch     = "SELECT p.id, p.src, p.dst, p.start, p.ping_sent, " +
		"p.probe_size, p.user_id, p.ttl, p.wait, p.spoofed_from, " +
		"p.version, p.spoofed, p.record_route, p.payload, p.tsonly, " +
		"p.tsandaddr, p.icmpsum, dl, p.`8` FROM " +
		"users u " +
		"inner join ping_batch pb on pb.user_id = u.id " +
		"inner join ping_batch_ping pbp on pb.id = pbp.batch_id " +
		"inner join pings p on p.id = pbp.ping_id " +
		"Where u.`key` = ? and pb.id = ?;"
)

// AddPingBatch adds a batch of pings
func (db *DB) AddPingBatch(u dm.User) (int64, error) {
	con := db.GetWriter()
	res, err := con.Exec(addPingBatch, u.ID)
	if err != nil {
		return 0, err
	}
	bid, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return bid, err
}

// AddPingsToBatch adds pings pids to batch bid
func (db *DB) AddPingsToBatch(bid int64, pids []int64) error {
	con := db.GetWriter()
	tx, err := con.Begin()
	if err != nil {
		return err
	}
	for _, pid := range pids {
		_, err := tx.Exec(addPingBatchPing, bid, pid)
		if err != nil {
			log.Error(err)
			return tx.Rollback()
		}
	}
	return tx.Commit()
}

// GetUser get a user with the given key
func (db *DB) GetUser(key string) (dm.User, error) {
	con := db.GetReader()
	row := con.QueryRow(getUser, key)
	var user dm.User
	err := row.Scan(&user.ID, &user.Name, &user.EMail, &user.Max, &user.Delay, &user.Key)
	if err != nil {
		return dm.User{}, err
	}
	return user, nil
}

// GetPingBatch gets a batch of pings for user u with id bid
func (db *DB) GetPingBatch(u dm.User, bid int64) ([]*dm.Ping, error) {
	tx, err := db.GetReader().Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := tx.Commit(); err != nil {
			log.Error(err)
		}
	}()
	rows, err := tx.Query(getPingBatch, u.Key, bid)
	if err != nil {
		return nil, err
	}
	var pings []*dm.Ping
	for rows.Next() {
		p := new(dm.Ping)
		var spoofed, recordRoute, payload, tsonly, tsandaddr, icmpsum, dl, eight bool
		var start int64
		err := rows.Scan(&p.Id, &p.Src, &p.Dst, &start,
			&p.PingSent, &p.ProbeSize, &p.UserId, &p.Ttl,
			&p.Wait, &p.SpoofedFrom, &p.Version, &spoofed,
			&recordRoute, &payload, &tsonly, &tsandaddr, &icmpsum,
			&dl, &eight)
		if err != nil {
			rows.Close()
			return nil, err
		}
		p.Start = &dm.Time{}
		p.Start.Sec = start / 1000000000
		p.Start.Usec = (start % 1000000000) / 1000
		p.Flags = makeFlags(spoofed, recordRoute, payload, tsonly, tsandaddr, icmpsum, dl, eight)
		pings = append(pings, p)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	statsstmt, err := tx.Prepare(getPingStats)
	if err != nil {
		return nil, err
	}
	defer statsstmt.Close()
	for _, p := range pings {
		var recordRoute, tsonly, tsandaddr bool
		recordRoute = hasFlag(p.Flags, "v4rr")
		tsonly = hasFlag(p.Flags, "tsonly")
		tsandaddr = hasFlag(p.Flags, "tsandaddr")
		err = getResponses(p, tx, recordRoute, tsonly, tsandaddr)
		if err != nil {
			return nil, err
		}
		err = getStats(p, statsstmt)
		if err != nil {
			if err != sql.ErrNoRows {
				return nil, err
			}
		}
	}
	return pings, nil
}
