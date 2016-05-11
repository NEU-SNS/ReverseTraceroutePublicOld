package repo

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/atlas/pb"
	"github.com/NEU-SNS/ReverseTraceroute/atlas/types"
	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/repository"
)

// Repo is a respository for storing and querying traceroutes
type Repo struct {
	repo *repository.DB
}

// Configs is a group of DB Configs
type Configs struct {
	WriteConfigs []Config
	ReadConfigs  []Config
}

// Config is a database configuration
type Config repository.Config

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

type errorf func() error

func logError(e errorf) {
	if err := e(); err != nil {
		log.Error(err)
	}
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
func (r *Repo) GetAtlasSources(dst uint32, stale time.Duration) ([]uint32, error) {
	rows, err := r.repo.GetReader().Query(getSources, dst, int64(stale.Minutes()))
	var srcs []uint32
	if err != nil {
		log.Error(err)
		return nil, err
	}
	defer logError(rows.Close)
	for rows.Next() {
		var curr uint32
		err := rows.Scan(&curr)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		srcs = append(srcs, curr)
	}
	if err = rows.Err(); err != nil {
		log.Error(err)
		return nil, err
	}
	return srcs, nil
}

// FindIntersectingTraceroute finds a traceroute that intersects hop towards the dst
func (r *Repo) FindIntersectingTraceroute(iq types.IntersectionQuery) (*pb.Path, error) {
	log.Debug("Finding intersecting traceroute ", iq)
	var rows *sql.Rows
	var err error
	if iq.IgnoreSource {
		rows, err = r.repo.GetReader().Query(findIntersectingIgnoreSource, iq.Addr, iq.Src, iq.Dst, int64(iq.Stale.Minutes()), iq.Addr, iq.Addr)
		if err != nil {
			log.Error(err)
			return nil, err
		}
	} else {
		rows, err = r.repo.GetReader().Query(findIntersecting, iq.Addr, iq.Dst, int64(iq.Stale.Minutes()), iq.Addr, iq.Addr)
		if err != nil {
			log.Error(err)
			return nil, err
		}
	}
	defer logError(rows.Close)
	ret := pb.Path{}
	for rows.Next() {
		row := hopRow{}
		err := rows.Scan(&row.src, &row.dest, &row.hop, &row.ttl)
		if err != nil {
			return nil, err
		}
		ret.Hops = append(ret.Hops, &pb.Hop{
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
	return &ret, nil
}

const (
	insertAtlasTrace = `INSERT INTO atlas_traceroutes(dest, src) VALUES(?, ?)`
	insertAtlasHop   = `
	INSERT INTO atlas_traceroute_hops(trace_id, hop, ttl) 
	VALUES (?, ?, ?)`
)

// StoreAtlasTraceroute stores a traceroute in a form that the Atlas requires
func (r *Repo) StoreAtlasTraceroute(trace *datamodel.Traceroute) error {
	conn := r.repo.GetWriter()
	tx, err := conn.Begin()
	if err != nil {
		return err
	}
	res, err := tx.Exec(insertAtlasTrace, trace.Dst, trace.Src)
	if err != nil {
		logError(tx.Rollback)
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		logError(tx.Rollback)
		return err
	}
	stmt, err := tx.Prepare(insertAtlasHop)
	if err != nil {
		logError(tx.Rollback)
		return err
	}
	_, err = stmt.Exec(int32(id), trace.Src, 0)
	if err != nil {
		logError(tx.Rollback)
		return err
	}
	for _, hop := range trace.GetHops() {
		_, err := stmt.Exec(int32(id), hop.Addr, hop.ProbeTtl)
		if err != nil {
			logError(tx.Rollback)
			return err
		}
	}
	err = stmt.Close()
	if err != nil {
		logError(tx.Rollback)
		return err
	}
	return tx.Commit()
}
