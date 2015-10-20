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
package sql

import (
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
)
import "github.com/go-sql-driver/mysql"

type DB struct {
	wdb []*sql.DB
	rdb []*sql.DB
	rr  *rand.Rand
	wr  *rand.Rand
}

type DbConfig struct {
	WriteConfigs []Config
	ReadConfigs  []Config
}

type Config struct {
	User     string
	Password string
	Host     string
	Port     string
	Db       string
}

var conFmt string = "%s:%s@tcp(%s:%s)/%s?parseTime=true"

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

type VantagePoint struct {
	Ip           uint32
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

func (vp *VantagePoint) ToDataModel() *dm.VantagePoint {
	nvp := &dm.VantagePoint{}
	nvp.Ip = vp.Ip
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

func (db *DB) GetVPs() ([]*dm.VantagePoint, error) {
	rows, err := db.getReader().Query(getVpsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	vps := make([]*dm.VantagePoint, 0)
	for rows.Next() {
		vp := &VantagePoint{}
		err := rows.Scan(
			&vp.Ip,
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

func (db *DB) UpdateController(ip, newc, con uint32) error {
	args := make([]interface{}, 0)
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

func (db *DB) UpdateCheckStatus(ip uint32, result string) error {
	_, err := db.getReader().Exec(updateCheckStatus, result, ip)
	return err
}

func (db *DB) Close() error {
	for _, d := range db.wdb {
		d.Close()
	}
	for _, d := range db.rdb {
		d.Close()
	}
	return nil
}
