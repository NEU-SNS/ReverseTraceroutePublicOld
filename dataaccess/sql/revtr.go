package sql

import (
	"database/sql"
	"fmt"

	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/log"
)

const (
	revtrStoreRevtr = `INSERT INTO reverse_traceroutes(src, dst, runtime, rr_issued, ts_issued, stop_reason) VALUES
	(?, ?, ?, ?, ?, ?)`
	revtrStoreRevtrHop = `INSERT INTO reverse_traceroute_hops(reverse_traceroute_id, hop, hop_type, order) VALUES
	(?, ?, ?, ?)`
	revtrGetUserByKey = "SELECT " +
		"`id`, `name`, `email`, `max`, `delay`, `key` " +
		"FROM " +
		"users " +
		"WHERE " +
		"`key` = ?"
)

// StoreRevtr stores a Revtr
func (db *DB) StoreRevtr(r dm.Revtr) error {
	con := db.getWriter()
	tx, err := con.Begin()
	if err != nil {
		return err
	}
	res, err := tx.Exec(revtrStoreRevtr, r.Src, r.Dst, r.Runtime.Nanoseconds(), r.RRIssued, r.TSIssued, r.StopReason)
	if err != nil {
		tx.Rollback()
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return err
	}
	for i, h := range r.Path {
		_, err := tx.Exec(revtrStoreRevtrHop, id, h.Hop, h.Type, i)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

var (
	// ErrNoRevtrUserFound is returned when no user is found with the given key
	ErrNoRevtrUserFound = fmt.Errorf("No user found")
)

// GetUserByKey gets a reverse traceroute user with the given key
func (db *DB) GetUserByKey(key string) (dm.RevtrUser, error) {
	con := db.getReader()
	res := con.QueryRow(revtrGetUserByKey, key)
	var ret dm.RevtrUser
	err := res.Scan(&ret.ID, &ret.Name, &ret.Email, &ret.Max, &ret.Delay, &ret.Key)
	switch {
	case err == sql.ErrNoRows:
		return ret, ErrNoRevtrUserFound
	case err != nil:
		log.Error(err)
		return ret, err
	default:
		return ret, nil
	}
}
