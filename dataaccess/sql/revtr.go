package sql

import (
	"database/sql"
	"fmt"
	"time"

	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/util"
)

const (
	revtrStoreRevtr = `INSERT INTO reverse_traceroutes(src, dst, runtime, rr_issued, ts_issued, stop_reason) VALUES
	(?, ?, ?, ?, ?, ?)`
	revtrInitRevtr         = `INSERT INTO reverse_traceroutes(src, dst) VALUES (?, ?)`
	revtrUpdateRevtrStatus = `UPDATE reverse_traceroutes SET status = ? WHERE id = ?`
	revtrStoreRevtrHop     = "INSERT INTO reverse_traceroute_hops(reverse_traceroute_id, hop, hop_type, `order`) VALUES (?, ?, ?, ?)"
	revtrGetUserByKey      = "SELECT " +
		"`id`, `name`, `email`, `max`, `delay`, `key` " +
		"FROM " +
		"users " +
		"WHERE " +
		"`key` = ?"
	revtrCanAddTraces = "SELECT " +
		"	CASE WHEN COUNT(*) + ? < u.max THEN TRUE ELSE FALSE END AS Valid " +
		"	FROM " +
		" 	users u INNER JOIN batch b ON u.id = b.user_id " +
		"	INNER JOIN batch_revtr brtr ON brtr.batch_id = b.id " +
		"	INNER JOIN reverse_traceroutes rt ON rt.id = brtr.revtr_id " +
		"	WHERE " +
		"	u.`key` = ? AND b.created >= DATE_SUB(NOW(), INTERVAL u.delay MINUTE) " +
		"	GROUP BY " +
		"		u.max "
	revtrAddBatch         = "INSERT INTO batch(user_id) SELECT id FROM users WHERE users.`key` = ?"
	revtrAddBatchRevtr    = "INSERT INTO batch_revtr(batch_id, revtr_id) VALUES (?, ?)"
	revtrGetRevtrsInBatch = "SELECT rt.id, rt.src, rt.dst, rt.runtime, rt.rr_issued, rt.ts_issued, rt.stop_reason, rt.status, rt.date " +
		"FROM users u INNER JOIN batch b ON u.id = b.user_id INNER JOIN batch_revtr brt ON b.id = brt.batch_id " +
		"INNER JOIN reverse_traceroutes rt ON brt.revtr_id = rt.id WHERE u.id = ? AND b.id = ?"
	revtrGetHopsForRevtr = "SELECT hop, hop_type FROM reverse_traceroute_hops rth WHERE rth.reverse_traceroute_id = ? ORDER BY rth.`order`"
	revtrUpdateRevtr     = `UPDATE reverse_traceroutes 
	SET 
		runtime = ?,
		rr_issued = ?,
		ts_issued = ?,
		stop_reason = ?,
		status = ?
	WHERE
		reverse_traceroutes.id = ?;`
)

var (
	// ErrInvalidUserID is returned when the user id provided is not in the system
	ErrInvalidUserID = fmt.Errorf("Invalid User Id")
	// ErrNoRow is returned when a query that should return row doesn't
	ErrNoRow = fmt.Errorf("No rows returned when one should have been")
	// ErrCannotAddRevtrBatch is returned if the user is not allowed to add more revtrs
	ErrCannotAddRevtrBatch = fmt.Errorf("Cannot add more revtrs")
)

// StoreBatchedRevtrs stores the results of a batch of revtrs
// this means updating the initial entries and adding in hops
func (db *DB) StoreBatchedRevtrs(batch []dm.ReverseTraceroute) error {
	con := db.getWriter()
	tx, err := con.Begin()
	if err != nil {
		return err
	}
	for _, rt := range batch {
		_, err = tx.Exec(revtrUpdateRevtr, rt.Runtime, rt.RrIssued, rt.TsIssued, rt.StopReason, rt.Status.String(), rt.Id)
		if err != nil {
			log.Error(err)
			tx.Rollback()
			return err
		}
		for i, hop := range rt.Path {
			hopi, _ := util.IPStringToInt32(hop.Hop)
			_, err = tx.Exec(revtrStoreRevtrHop, rt.Id, hopi, uint32(hop.Type), i)
			if err != nil {
				log.Error(err)
				tx.Rollback()
				return err
			}
		}
	}
	err = tx.Commit()
	if err != nil {
		log.Error(err)
		return tx.Rollback()
	}
	return nil
}

type rtid struct {
	rt dm.ReverseTraceroute
	id uint32
}

// GetRevtrsInBatch gets the reverse traceroutes in batch bid
func (db *DB) GetRevtrsInBatch(uid, bid uint32) ([]*dm.ReverseTraceroute, error) {
	con := db.getReader()
	res, err := con.Query(revtrGetRevtrsInBatch, uid, bid)
	defer res.Close()
	if err != nil {
		return nil, err
	}
	var ret []rtid
	var final []*dm.ReverseTraceroute
	for res.Next() {
		var r dm.ReverseTraceroute
		var src, dst, id uint32
		var t time.Time
		var status string
		err = res.Scan(&id, &src, &dst, &r.Runtime, &r.RrIssued, &r.TsIssued, &r.StopReason, &status, &t)
		if err != nil {
			return nil, err
		}
		r.Src, _ = util.Int32ToIPString(src)
		r.Dst, _ = util.Int32ToIPString(dst)
		r.Date = t.String()
		r.Status = dm.RevtrStatus(dm.RevtrStatus_value[status])
		if r.Status == dm.RevtrStatus_RUNNING {
			r.Runtime = time.Since(t).Nanoseconds()
		}
		ret = append(ret, rtid{rt: r, id: id})
	}
	log.Debug(ret)
	if err := res.Err(); err != nil {
		return nil, err
	}
	for _, rt := range ret {
		use := rt.rt
		log.Debug(rt)
		if use.Status == dm.RevtrStatus_COMPLETED {
			res2, err := con.Query(revtrGetHopsForRevtr, rt.id)
			if err != nil {
				return nil, err
			}
			for res2.Next() {
				h := dm.RevtrHop{}
				var hop, hopType uint32
				err = res2.Scan(&hop, &hopType)
				h.Hop, _ = util.Int32ToIPString(hop)
				h.Type = dm.RevtrHopType(hopType)
				use.Path = append(use.Path, &h)
				log.Debug(h)
			}
			if err := res2.Err(); err != nil {
				return nil, err
			}
			res2.Close()
		}
		final = append(final, &(use))
	}
	log.Debug(final)
	return final, nil
}

// CreateRevtrBatch creatse a batch of revtrs if the user identified by id
// is allowed to issue more reverse traceroutes
func (db *DB) CreateRevtrBatch(batch []dm.RevtrMeasurement, id string) ([]dm.RevtrMeasurement, uint32, error) {
	con := db.getWriter()
	tx, err := con.Begin()
	if err != nil {
		return nil, 0, err
	}
	var canDo bool
	err = tx.QueryRow(revtrCanAddTraces, len(batch), id).Scan(&canDo)
	switch {
	// This requires the assumption that I'm already authorized
	case err == sql.ErrNoRows:
		canDo = true
	case err != nil:
		log.Error(err)
		tx.Rollback()
		return nil, 0, ErrCannotAddRevtrBatch
	}
	if !canDo {
		tx.Rollback()
		return nil, 0, ErrCannotAddRevtrBatch
	}
	res, err := tx.Exec(revtrAddBatch, id)
	if err != nil {
		log.Error(err)
		tx.Rollback()
		return nil, 0, ErrCannotAddRevtrBatch
	}
	bID, err := res.LastInsertId()
	if err != nil {
		log.Error(err)
		tx.Rollback()
		return nil, 0, ErrCannotAddRevtrBatch
	}
	batchID := uint32(bID)
	var added []dm.RevtrMeasurement
	for _, rm := range batch {
		src, _ := util.IPStringToInt32(rm.Src)
		dst, _ := util.IPStringToInt32(rm.Dst)
		res, err := tx.Exec(revtrInitRevtr, src, dst)
		if err != nil {
			tx.Rollback()
			log.Error(err)
			return nil, 0, ErrCannotAddRevtrBatch
		}
		id, err := res.LastInsertId()
		if err != nil {
			tx.Rollback()
			log.Error(err)
			return nil, 0, ErrCannotAddRevtrBatch
		}
		_, err = tx.Exec(revtrAddBatchRevtr, batchID, uint32(id))
		if err != nil {
			tx.Rollback()
			log.Error(err)
			return nil, 0, ErrCannotAddRevtrBatch
		}
		rm.Id = uint32(id)
		added = append(added, rm)
	}
	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		log.Error(err)
		return nil, 0, ErrCannotAddRevtrBatch
	}
	return added, batchID, nil
}

// StoreRevtr stores a Revtr
func (db *DB) StoreRevtr(r dm.ReverseTraceroute) error {
	con := db.getWriter()
	tx, err := con.Begin()
	if err != nil {
		log.Error(err)
		return err
	}
	src, _ := util.IPStringToInt32(r.Src)
	dst, _ := util.IPStringToInt32(r.Dst)
	res, err := tx.Exec(revtrStoreRevtr, src, dst, r.Runtime, r.RrIssued, r.TsIssued, r.StopReason)
	if err != nil {
		log.Error(err)
		tx.Rollback()
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Error(err)
		tx.Rollback()
		return err
	}
	for i, h := range r.Path {
		hop, _ := util.IPStringToInt32(h.Hop)
		_, err := tx.Exec(revtrStoreRevtrHop, id, hop, h.Type, i)
		if err != nil {
			log.Error(err)
			tx.Rollback()
			return err
		}
	}
	err = tx.Commit()
	if err != nil {
		log.Error(err)
		tx.Rollback()
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
