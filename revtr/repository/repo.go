package repo

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/repository"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/pb"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/types"
	"github.com/NEU-SNS/ReverseTraceroute/util"
)

const (
	revtrStoreRevtr = `INSERT INTO reverse_traceroutes(src, dst, runtime, rr_issued, ts_issued, stop_reason, status) VALUES
	(?, ?, ?, ?, ?, ?, ?)`
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
	// ErrFailedToStoreBatch is returned when storing a batch of revtrs failed
	ErrFailedToStoreBatch = fmt.Errorf("Failed to store batch of revtrs")
	// ErrFailedToGetBatch is returned when a batch cannot be fetched
	ErrFailedToGetBatch = fmt.Errorf("Failed to get batch of revtrs")
)

// Repo is a repository for storing and retreiving reverse traceroutes
type Repo struct {
	repo *repository.DB
}

// Configs is the configuration for the repo
type Configs struct {
	WriteConfigs []Config
	ReadConfigs  []Config
}

// Config is an individual reader/writer config
type Config struct {
	User     string
	Password string
	Host     string
	Port     string
	Db       string
}

type repoOptions struct {
	writeConfigs []Config
	readConfigs  []Config
}

// Option sets up the Repo
type Option func(*repoOptions)

// WithWriteConfig configures the repo with the given config used as a writer
// multiples may be provided
func WithWriteConfig(c Config) Option {
	return func(ro *repoOptions) {
		ro.writeConfigs = append(ro.writeConfigs, c)
	}
}

// WithReadConfig configures the repo with the given config used as a reader
// multiples may be provided
func WithReadConfig(c Config) Option {
	return func(ro *repoOptions) {
		ro.readConfigs = append(ro.readConfigs, c)
	}
}

// NewRepo creates a new Repo configured with the given options
func NewRepo(options ...Option) (*Repo, error) {
	ro := &repoOptions{}
	for _, opt := range options {
		opt(ro)
	}
	var dbc repository.DbConfig
	for _, wc := range ro.writeConfigs {
		var c repository.Config
		c.User = wc.User
		c.Password = wc.Password
		c.Host = wc.Host
		c.Port = wc.Port
		c.Db = wc.Db
		dbc.WriteConfigs = append(dbc.WriteConfigs, c)
	}
	for _, rc := range ro.readConfigs {
		var c repository.Config
		c.User = rc.User
		c.Password = rc.Password
		c.Host = rc.Host
		c.Port = rc.Port
		c.Db = rc.Db
		dbc.ReadConfigs = append(dbc.ReadConfigs, c)
	}
	db, err := repository.NewDB(dbc)
	if err != nil {
		return nil, err
	}
	return &Repo{repo: db}, nil
}

// StoreBatchedRevtrs stores a batch of Revtrs
func (r *Repo) StoreBatchedRevtrs(batch []pb.ReverseTraceroute) error {
	con := r.repo.GetWriter()
	tx, err := con.Begin()
	if err != nil {
		log.Error(err)
		return ErrFailedToStoreBatch
	}
	for _, rt := range batch {
		_, err = tx.Exec(revtrUpdateRevtr, rt.Runtime, rt.RrIssued, rt.TsIssued, rt.StopReason, rt.Status.String(), rt.Id)
		if err != nil {
			log.Error(err)
			if err := tx.Rollback(); err != nil {
				log.Error(err)
			}
			return ErrFailedToStoreBatch
		}
		for i, hop := range rt.Path {
			hopi, _ := util.IPStringToInt32(hop.Hop)
			_, err = tx.Exec(revtrStoreRevtrHop, rt.Id, hopi, uint32(hop.Type), i)
			if err != nil {
				log.Error(err)
				if err := tx.Rollback(); err != nil {
					log.Error(err)
				}
				return ErrFailedToStoreBatch
			}
		}
	}
	err = tx.Commit()
	if err != nil {
		log.Error(err)
		if err := tx.Rollback(); err != nil {
			log.Error(err)
		}
		return ErrFailedToStoreBatch
	}
	return nil
}

type rtid struct {
	rt pb.ReverseTraceroute
	id uint32
}

// GetRevtrsInBatch gets the reverse traceroutes in batch bid
func (r *Repo) GetRevtrsInBatch(uid, bid uint32) ([]*pb.ReverseTraceroute, error) {
	con := r.repo.GetReader()
	res, err := con.Query(revtrGetRevtrsInBatch, uid, bid)
	defer func() {
		if err := res.Close(); err != nil {
			log.Error(err)
		}
	}()
	if err != nil {
		log.Error(err)
		return nil, ErrFailedToGetBatch
	}
	var ret []rtid
	var final []*pb.ReverseTraceroute
	for res.Next() {
		var r pb.ReverseTraceroute
		var src, dst, id uint32
		var t time.Time
		var status string
		err = res.Scan(&id, &src, &dst, &r.Runtime, &r.RrIssued, &r.TsIssued, &r.StopReason, &status, &t)
		if err != nil {
			log.Error(err)
			return nil, ErrFailedToGetBatch
		}
		r.Src, _ = util.Int32ToIPString(src)
		r.Dst, _ = util.Int32ToIPString(dst)
		r.Date = t.String()
		r.Status = pb.RevtrStatus(pb.RevtrStatus_value[status])
		if r.Status == pb.RevtrStatus_RUNNING {
			r.Runtime = time.Since(t).Nanoseconds()
		}
		ret = append(ret, rtid{rt: r, id: id})
	}
	if err := res.Err(); err != nil {
		log.Error(err)
		return nil, ErrFailedToGetBatch
	}
	for _, rt := range ret {
		use := rt.rt
		log.Debug(rt)
		if use.Status == pb.RevtrStatus_COMPLETED {
			res2, err := con.Query(revtrGetHopsForRevtr, rt.id)
			if err != nil {
				log.Error(err)
				return nil, ErrFailedToGetBatch
			}
			for res2.Next() {
				h := pb.RevtrHop{}
				var hop, hopType uint32
				err = res2.Scan(&hop, &hopType)
				h.Hop, _ = util.Int32ToIPString(hop)
				h.Type = pb.RevtrHopType(hopType)
				use.Path = append(use.Path, &h)
				log.Debug(h)
			}
			if err := res2.Err(); err != nil {
				log.Error(err)
				return nil, ErrFailedToGetBatch
			}
			if err := res2.Close(); err != nil {
				log.Error(err)
			}
		}
		final = append(final, &(use))
	}
	log.Debug(final)
	return final, nil
}

type errorf func() error

func logError(e errorf) {
	if err := e(); err != nil {
		log.Error(err)
	}
}

// CreateRevtrBatch creatse a batch of revtrs if the user identified by id
// is allowed to issue more reverse traceroutes
func (r *Repo) CreateRevtrBatch(batch []*pb.RevtrMeasurement, id string) ([]*pb.RevtrMeasurement, uint32, error) {
	con := r.repo.GetWriter()
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
		logError(tx.Rollback)
		return nil, 0, ErrCannotAddRevtrBatch
	}
	if !canDo {
		logError(tx.Rollback)
		return nil, 0, ErrCannotAddRevtrBatch
	}
	res, err := tx.Exec(revtrAddBatch, id)
	if err != nil {
		log.Error(err)
		logError(tx.Rollback)
		return nil, 0, ErrCannotAddRevtrBatch
	}
	bID, err := res.LastInsertId()
	if err != nil {
		log.Error(err)
		logError(tx.Rollback)
		return nil, 0, ErrCannotAddRevtrBatch
	}
	batchID := uint32(bID)
	var added []*pb.RevtrMeasurement
	for _, rm := range batch {
		src, _ := util.IPStringToInt32(rm.Src)
		dst, _ := util.IPStringToInt32(rm.Dst)
		res, err := tx.Exec(revtrInitRevtr, src, dst)
		if err != nil {
			logError(tx.Rollback)
			log.Error(err)
			return nil, 0, ErrCannotAddRevtrBatch
		}
		id, err := res.LastInsertId()
		if err != nil {
			logError(tx.Rollback)
			log.Error(err)
			return nil, 0, ErrCannotAddRevtrBatch
		}
		_, err = tx.Exec(revtrAddBatchRevtr, batchID, uint32(id))
		if err != nil {
			logError(tx.Rollback)
			log.Error(err)
			return nil, 0, ErrCannotAddRevtrBatch
		}
		rm.Id = uint32(id)
		added = append(added, rm)
	}
	err = tx.Commit()
	if err != nil {
		logError(tx.Rollback)
		log.Error(err)
		return nil, 0, ErrCannotAddRevtrBatch
	}
	return added, batchID, nil
}

// StoreRevtr stores a Revtr
func (r *Repo) StoreRevtr(rt pb.ReverseTraceroute) error {
	con := r.repo.GetWriter()
	tx, err := con.Begin()
	if err != nil {
		log.Error(err)
		return err
	}
	src, _ := util.IPStringToInt32(rt.Src)
	dst, _ := util.IPStringToInt32(rt.Dst)
	res, err := tx.Exec(revtrStoreRevtr, src, dst, rt.Runtime, rt.RrIssued, rt.TsIssued, rt.StopReason, rt.Status.String())
	if err != nil {
		log.Error(err)
		logError(tx.Rollback)
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Error(err)
		logError(tx.Rollback)
		return err
	}
	for i, h := range rt.Path {
		hop, _ := util.IPStringToInt32(h.Hop)
		_, err := tx.Exec(revtrStoreRevtrHop, id, hop, h.Type, i)
		if err != nil {
			log.Error(err)
			logError(tx.Rollback)
			return err
		}
	}
	err = tx.Commit()
	if err != nil {
		log.Error(err)
		logError(tx.Rollback)
		return err
	}
	return nil
}

var (
	// ErrNoRevtrUserFound is returned when no user is found with the given key
	ErrNoRevtrUserFound = fmt.Errorf("No user found")
)

// GetUserByKey gets a reverse traceroute user with the given key
func (r *Repo) GetUserByKey(key string) (pb.RevtrUser, error) {
	con := r.repo.GetReader()
	res := con.QueryRow(revtrGetUserByKey, key)
	var ret pb.RevtrUser
	err := res.Scan(&ret.Id, &ret.Name, &ret.Email, &ret.Max, &ret.Delay, &ret.Key)
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

const (
	selectByAddressAndDest24AdjDstQuery = `
	SELECT dest24, address, adjacent, cnt
	FROM adjacencies_to_dest 
	WHERE address = ? AND dest24 = ?
	ORDER BY cnt DESC LIMIT 500`
	selectByIP1AdjQuery = `SELECT ip1, ip2, cnt from adjacencies WHERE ip1 = ?
							ORDER BY cnt DESC LIMIT 500`
	selectByIP2AdjQuery = `SELECT ip1, ip2, cnt from adjacencies WHERE ip2 = ?
							ORDER BY cnt DESC LIMIT 500`

	aliasGetByIP          = `SELECT cluster_id FROM ip_aliases WHERE ip_address = ? LIMIT 1`
	aliasGetIPsForCluster = `SELECT ip_address FROM ip_aliases WHERE cluster_id = ? LIMIT 2000`
)

// GetAdjacenciesByIP1 gets ajds by ip1
func (r *Repo) GetAdjacenciesByIP1(ip uint32) ([]types.Adjacency, error) {
	con := r.repo.GetReader()
	res, err := con.Query(selectByIP1AdjQuery, ip)
	if err != nil {
		return nil, err
	}
	defer logError(res.Close)
	var adjs []types.Adjacency
	for res.Next() {
		var adj types.Adjacency
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
func (r *Repo) GetAdjacenciesByIP2(ip uint32) ([]types.Adjacency, error) {
	con := r.repo.GetReader()
	res, err := con.Query(selectByIP2AdjQuery, ip)
	if err != nil {
		return nil, err
	}
	defer logError(res.Close)
	var adjs []types.Adjacency
	for res.Next() {
		var adj types.Adjacency
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

// GetAdjacencyToDestByAddrAndDest24 does what it says
func (r *Repo) GetAdjacencyToDestByAddrAndDest24(dest24, addr uint32) ([]types.AdjacencyToDest, error) {
	con := r.repo.GetReader()
	res, err := con.Query(selectByAddressAndDest24AdjDstQuery, addr, dest24)
	if err != nil {
		return nil, err

	}
	defer logError(res.Close)
	var adjs []types.AdjacencyToDest
	for res.Next() {
		var adj types.AdjacencyToDest
		err = res.Scan(&adj.Dest24, &adj.Address, &adj.Adjacent, &adj.Cnt)
		if err != nil {
			return nil, err

		}
		adjs = append(adjs, adj)

	}
	return adjs, nil

}

var (
	// ErrNoAlias is returned when no alias is found for an ip
	ErrNoAlias = fmt.Errorf("No alias found")
)

// GetClusterIDByIP gets a the cluster ID for a give ip
func (r *Repo) GetClusterIDByIP(ip uint32) (int, error) {
	con := r.repo.GetReader()
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
func (r *Repo) GetIPsForClusterID(id int) ([]uint32, error) {
	con := r.repo.GetReader()
	var scan uint32
	var ret []uint32
	res, err := con.Query(aliasGetByIP, id)
	defer logError(res.Close)
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
