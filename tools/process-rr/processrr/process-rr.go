package processrr

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"sync"

	"log"
)

// import mysql driver
import _ "github.com/go-sql-driver/mysql"

type dbConfig struct {
	url  string
	user string
	pass string
	name string
	port string
}

// Ping is a ping
type Ping struct {
	Src, Dst, SpoofedFrom uint32
	ID                    int64
}

// PingResponse is a ping response
type PingResponse struct {
	ID   int64
	From uint32
	Src  uint32
	RRS  []RR
}

// RR is an RR hop
type RR struct {
	Addr uint32
	Hop  int
}

// Result is the result of processning a ping
type Result struct {
	Src, Dst, Slash24 uint32
	Dist              int
}

var conFmt = "%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Local"
var dryRun bool

// Main runs the processrr procedure
func Main() int {
	var (
		rCon, wCon dbConfig
	)
	log.SetFlags(log.Lshortfile)
	fs := flag.NewFlagSet("processrr", flag.ExitOnError)
	fs.StringVar(&wCon.url, "db-write-url", "", "The url for the write database")
	fs.StringVar(&wCon.port, "db-write-port", "3306", "The port for the write database")
	fs.StringVar(&wCon.user, "db-write-user", "", "The username for the write database")
	fs.StringVar(&wCon.pass, "db-write-pass", "", "The password to use for the write database")
	fs.StringVar(&wCon.name, "db-write-name", "", "The name of the write db to use")
	fs.StringVar(&rCon.url, "db-read-url", "", "The url for the read database")
	fs.StringVar(&rCon.port, "db-read-port", "3306", "The port for the read database")
	fs.StringVar(&rCon.user, "db-read-user", "", "The username for the read database")
	fs.StringVar(&rCon.pass, "db-read-pass", "", "The password to use for the read database")
	fs.StringVar(&rCon.name, "db-read-name", "", "The name of the read db to use")
	fs.BoolVar(&dryRun, "dry-run", false, "If set, do not write results to the database, print them to stdout instead.")
	err := fs.Parse(os.Args[1:])
	if err != nil {
		log.Println(err)
		return 1
	}
	wconString := fmt.Sprintf(conFmt, wCon.user, wCon.pass, wCon.url, wCon.port, wCon.name)
	rconString := fmt.Sprintf(conFmt, rCon.user, rCon.pass, rCon.url, rCon.port, rCon.name)
	var wdb *sql.DB
	if !dryRun {
		wdb, err = sql.Open("mysql", wconString)
		if err != nil {
			log.Println(err)
			return 1
		}
		if err = wdb.Ping(); err != nil {
			log.Println(err)
			return 1
		}
		if err := copyOld(wdb); err != nil {
			log.Println(err)
			return 1
		}
		wdb.SetMaxIdleConns(24)
		wdb.SetMaxOpenConns(24)
	}
	var rdb *sql.DB
	rdb, err = sql.Open("mysql", rconString)
	if err != nil {
		log.Println(err)
		return 1
	}
	if err = rdb.Ping(); err != nil {
		log.Println(err)
		return 1
	}
	rdb.SetMaxIdleConns(24)
	rdb.SetMaxOpenConns(24)
	defer func() {
		if rdb != nil {
			err := rdb.Close()
			if err != nil {
				log.Println(err)
			}
		}
	}()
	defer func() {
		if wdb != nil {
			err := wdb.Close()
			if err != nil {
				log.Println(err)
			}
		}
	}()
	errch := make(chan error)
	done := processPings(rdb, wdb, errch)
	for {
		select {
		case err := <-errch:
			log.Println(err)
		case <-done:
			if !dryRun {
				if err := finalize(wdb); err != nil {
					log.Println(err)
					return 1
				}
			}
			return 0
		}
	}
}

const (
	insertRes          = `insert into dist_to_dest_temp(src, dst, dist, slash_24) values(?, ?, ?, ?)`
	moveOld            = `insert into dist_to_dest_temp(src, dst, dist, slash_24) select src, dst, dist, slash_24 from dist_to_dest`
	getPings           = `select id, src, dst, spoofed_from from pings where record_route = 1`
	getResponseForPing = "select id, `from` from ping_responses where ping_id = ?"
	getRRForResponse   = `select hop, ip from record_routes where response_id = ?`
	insertNew          = `insert into dist_to_dest(src, dst, dist, slash_24) select src, dst, min(dist) dist, slash_24 from dist_to_dest_temp group by src, dst, slash_24 on duplicate key update dist = values(dist)`
	clearTemp          = `truncate table dist_to_dest_temp;`
)

func finalize(db *sql.DB) error {
	_, err := db.Exec(insertNew)
	if err != nil {
		return err
	}
	_, err = db.Exec(clearTemp)
	if err != nil {
		return err
	}
	return nil
}

func copyOld(db *sql.DB) error {
	_, err := db.Exec(moveOld)
	if err != nil {
		return err
	}
	return nil
}

func processPing(wg *sync.WaitGroup, wdb, db *sql.DB, inp <-chan *Ping, resp, rrs, res *sql.Stmt) {
	defer wg.Done()
	for p := range inp {
		var responses []*PingResponse
		rows, err := resp.Query(p.ID)
		if err != nil {
			log.Println(err)
			return
		}
		for rows.Next() {
			resp := new(PingResponse)
			err := rows.Scan(&resp.ID, &resp.From)
			if err != nil {
				log.Println(err)
				rows.Close()
				return
			}
			if resp.From == p.Dst {
				r, err := rrs.Query(resp.ID)
				if err != nil {
					log.Println(err)
					rows.Close()
					return
				}
				for r.Next() {
					rr := new(RR)
					err = r.Scan(&rr.Hop, &rr.Addr)
					if err != nil {
						log.Println(err)
						rows.Close()
						r.Close()
						return
					}
					resp.RRS = append(resp.RRS, *rr)
					if p.SpoofedFrom != 0 {
						resp.Src = p.SpoofedFrom
					} else {
						resp.Src = p.Src
					}
				}
				if err = r.Err(); err != nil {
					log.Println(err)
					r.Close()
					rows.Close()
					return
				}
				responses = append(responses, resp)
			}
		}
		if err = rows.Err(); err != nil {
			rows.Close()
			log.Println(err)
			return
		}
		rows.Close()
		for _, r := range responses {
			results := getResults(r)
			for _, result := range results {
				if dryRun {
					fmt.Printf("Src: %d Dst: %d Dist: %d /24: %d\n", result.Src, result.Dst, result.Dist, result.Slash24)
					continue
				}
				_, err := res.Exec(result.Src, result.Dst, result.Dist, result.Slash24)
				if err != nil {
					log.Println(err)
					return
				}
			}
		}
	}
}

func getResults(p *PingResponse) []Result {
	var results []Result
	for i, rr := range p.RRS {
		var res Result
		res.Src = p.Src
		res.Slash24 = rr.Addr >> 8
		res.Dst = rr.Addr
		res.Dist = i + 1
		results = append(results, res)
		if rr.Addr == p.From {
			break
		}
	}
	return results
}

const routineCount = 12

func processPings(db, wdb *sql.DB, ec chan error) chan struct{} {
	var wg sync.WaitGroup
	done := make(chan struct{})
	pc := make(chan *Ping)
	rows, err := db.Query(getPings)
	if err != nil {
		select {
		case <-done:
		case ec <- err:
		}
	}
	respState, err := db.Prepare(getResponseForPing)
	if err != nil {
		log.Println(err)
		select {
		case <-done:
		case ec <- err:
		}
	}
	rrsState, err := db.Prepare(getRRForResponse)
	if err != nil {
		log.Println(err)
		select {
		case <-done:
		case ec <- err:
		}
	}
	var resState *sql.Stmt
	if !dryRun {
		resState, err = wdb.Prepare(insertRes)
		if err != nil {
			log.Println(err)
			select {
			case <-done:
			case ec <- err:
			}
		}
	}
	wg.Add(routineCount)
	for i := 0; i < routineCount; i++ {
		go processPing(&wg, wdb, db, pc, respState, rrsState, resState)
	}
	go func() {
		wg.Wait()
		close(done)
	}()
	go func() {
		defer rows.Close()
		for rows.Next() {
			p := new(Ping)
			err = rows.Scan(&p.ID, &p.Src, &p.Dst, &p.SpoofedFrom)
			if err != nil {
				select {
				case <-done:
				case ec <- err:
				}
				close(pc)
				return
			}
			select {
			case <-done:
				return
			case pc <- p:
			}
		}
		if err = rows.Err(); err != nil {
			select {
			case <-done:
			case ec <- err:
			}
		}
		close(pc)
	}()
	return done
}
