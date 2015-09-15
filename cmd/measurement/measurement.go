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
package main

import (
	"bufio"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"runtime"
	"strings"
	"time"

	"google.golang.org/grpc"

	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	plc "github.com/NEU-SNS/ReverseTraceroute/plcontrollerapi"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	ctx "golang.org/x/net/context"
)

import _ "github.com/go-sql-driver/mysql"

var (
	srcPath   string
	dstPath   string
	rr        bool
	uname     string
	passwd    string
	conFmt    string = "%s:%s@tcp(localhost:3306)/record_routes?parseTime=true"
	outDir    string
	writeDb   bool
	runId     string
	version   string = "0.0.3"
	showV     bool
	batchSize int
	delay     time.Duration
)

const (
	insertPing = `
INSERT INTO record_routes.pings(src, dst, start, version, type, ping_sent, probe_size, ttl, wait, timeout, flags, runId, error)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`
	insertResponse = "INSERT INTO record_routes.ping_responses(src, dst, start, `from`, seq, reply_size, reply_ttl, reply_proto, tx, rx, rtt, probe_ipid, reply_ipid, icmp_type, icmp_code, response) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	insertHop      = "INSERT INTO record_routes.routes(src, dst, start,`from`,hop, ip, ping_response) VALUES(?, ?, ?, ?, ?, ?, ?)"
	insertStats    = "INSERT INTO record_routes.ping_stats(src, dst, start, replies, loss, min, max, avg, stddev) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)"
)

func init() {
	flag.StringVar(&srcPath, "s", "", "Path to the file containing src addresses")
	flag.StringVar(&dstPath, "d", "", "Path to the file containing dst addresses")
	flag.BoolVar(&rr, "r", true, "Option to run with record route")
	flag.StringVar(&uname, "uname", "", "Username for the db")
	flag.StringVar(&passwd, "passwd", "", "Password for the db")
	flag.StringVar(&runId, "id", "", "Id to associate with the measurement set")
	flag.BoolVar(&writeDb, "db", false, "Set if it is desired to write to the db")
	flag.StringVar(&outDir, "out", "", "Directory to write output to")
	flag.IntVar(&batchSize, "b", 0, "Number of measurements to run in each batch")
	flag.DurationVar(&delay, "delay", 0, "How long to wait between measurement sets")
	flag.BoolVar(&showV, "version", false, "Show the version number")
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
	if showV {
		fmt.Println(version)
		os.Exit(0)
	}
	if batchSize == 0 {
		fmt.Fprintln(os.Stderr, "BatchSize not provided")
		os.Exit(1)
	}
	if runId == "" {
		fmt.Fprintln(os.Stderr, "RunId not provided")
		os.Exit(1)
	}
	if outDir == "" {
		fmt.Fprintln(os.Stderr, "Outdir not provided")
		os.Exit(1)
	}
	if (passwd == "" || uname == "") && writeDb {
		fmt.Fprintln(os.Stderr, "Missing db args")
		os.Exit(1)
	}
	err := os.Mkdir(outDir, os.ModeDir|0755)
	if err != nil && !os.IsExist(err) {
		fmt.Fprintln(os.Stderr, "Error creating outdir: %v", err)
		os.Exit(1)
	}
	db, err := sql.Open("mysql", fmt.Sprintf(conFmt, uname, passwd))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to open db: %v", err)
		os.Exit(1)
	}
	defer db.Close()
	if err = db.Ping(); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to ping db: %v", err)
		os.Exit(1)
	}
	if srcPath == "" {
		fmt.Fprintln(os.Stderr, "Src path is a required argument")
		db.Close()
		os.Exit(1)
	}

	if dstPath == "" {
		fmt.Fprintln(os.Stderr, "Dst path is a required argument")
		db.Close()
		os.Exit(1)
	}

	srcFile, err := os.Open(srcPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open %s: %v\n", srcPath, err)
		db.Close()
		os.Exit(1)
	}
	defer srcFile.Close()
	srcs := make([]string, 0)
	scanner := bufio.NewScanner(srcFile)
	scanner.Split(bufio.ScanLines)
	line := 1
	for scanner.Scan() {
		ip := scanner.Text()
		ip1 := net.ParseIP(ip)
		if ip1 == nil {
			fmt.Fprintf(os.Stderr, "Invalid ip at line: %d, %s\n", line, ip)
			srcFile.Close()
			db.Close()
			os.Exit(1)
		}
		srcs = append(srcs, ip1.String())
		line++
	}

	dstFile, err := os.Open(dstPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open %s: %v\n", dstPath, err)
		srcFile.Close()
		db.Close()
		os.Exit(1)
	}
	defer dstFile.Close()

	line = 0
	read := bufio.NewReader(dstFile)
	dsts := make([]string, 0)
	for {
		ip, err := read.ReadString('\n')
		if err != nil && err != io.EOF {
			fmt.Fprintf(os.Stderr, "Failed to open %s: %v\n", dstPath, err)
			srcFile.Close()
			dstFile.Close()
			db.Close()
			os.Exit(1)
		}
		if err == io.EOF && line == 0 {
			break
		}
		ip1 := net.ParseIP(strings.TrimSpace(ip))
		if ip1 == nil {
			fmt.Fprintf(os.Stderr, "Invalid ip at line: %d, %s\n", line, ip)
			srcFile.Close()
			dstFile.Close()
			db.Close()
			os.Exit(1)
		}
		dsts = append(dsts, ip1.String())
		line++
		if line == batchSize {
			runMeasurements(srcs, dsts, db)
			line = 0
			dsts = make([]string, 0)
			<-time.After(time.Second)
			continue
		}
		if err == io.EOF {
			if len(dsts) == 0 {
				break
			}
			runMeasurements(srcs, dsts, db)
			break
		}

	}

}

func runMeasurements(srcs, dsts []string, db *sql.DB) error {
	pingReq := &dm.PingArg{
		Pings: make([]*dm.PingMeasurement, 0),
	}
	for _, src := range srcs {
		for _, dst := range dsts {
			pingReq.Pings = append(pingReq.Pings, &dm.PingMeasurement{
				Src:     src,
				Dst:     dst,
				RR:      rr,
				Timeout: 60,
				Count:   "1",
			})
		}
	}
	conn, err := grpc.Dial("129.10.113.189:4380")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to controller: %v", err)
		os.Exit(1)
	}
	defer conn.Close()
	cl := plc.NewPLControllerClient(conn)
	fmt.Println("Num of requests:", len(pingReq.Pings))
	start := time.Now()
	fmt.Println("Starting:", start)
	st, err := cl.Ping(ctx.Background(), pingReq)
	if err != nil {
		conn.Close()
		fmt.Fprintf(os.Stderr, "Failed to run ping: %v", err)
		os.Exit(1)
	}
	ps := make([]*dm.Ping, 0)
	for {
		pr, err := st.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			conn.Close()
			fmt.Fprintf(os.Stderr, "Error while running measurement: %v", err)
			return err
		}
		err = os.Mkdir(fmt.Sprintf("%s/%s", outDir, pr.Src), os.ModeDir|0755)
		if err != nil && !os.IsExist(err) {
			fmt.Fprintln(os.Stderr, "Error creating dir: %v", pr.Src)
		}
		err = ioutil.WriteFile(fmt.Sprintf("%s/%s/%s_%s", outDir, pr.Src, pr.Src, pr.Dst), []byte(pr.String()), 0666)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not write file: %v", err)
		}
		err = storePing(pr, db)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error storing ping: %v", err)
		}
		ps = append(ps, pr)
	}
	end := time.Now()
	fmt.Println("Finished:", end, "Took:", time.Since(start), "Received:", len(ps), "responses")
	return nil
}

func storePing(p *dm.Ping, db *sql.DB) error {
	tran, err := db.Begin()
	if err != nil {
		return err
	}
	src, err := util.IPStringToInt32(p.Src)
	if err != nil {
		return err
	}
	dst, err := util.IPStringToInt32(p.Dst)
	if err != nil {
		return err
	}
	start := time.Unix(p.Start.Sec, util.MicroToNanoSec(p.Start.Usec))
	_, err = tran.Exec(insertPing, src, dst, start.UnixNano(), p.Version, p.Type, p.PingSent, p.ProbeSize, p.Ttl, p.Wait, p.Timeout, fmt.Sprintf("%v", p.Flags), runId, p.Error)
	if err != nil {
		tran.Rollback()
		return err
	}
	stats := p.GetStatistics()
	if stats == nil {
		err = tran.Commit()
		if err != nil {
			tran.Rollback()
			return err
		}
		return nil
	}
	_, err = tran.Exec(insertStats, src, dst, start.UnixNano(), stats.Replies, stats.Loss, stats.Min, stats.Max, stats.Avg, stats.Stddev)
	if err != nil {
		tran.Rollback()
		return err
	}
	responses := p.GetResponses()
	if responses == nil {
		err = tran.Commit()
		if err != nil {
			tran.Rollback()
			return err
		}
		return nil
	}
	for j, response := range responses {
		from, err := util.IPStringToInt32(response.From)
		if err != nil {
			tran.Rollback()
			return err
		}
		tx := time.Unix(response.Tx.Sec, util.MicroToNanoSec(response.Tx.Usec))
		rx := time.Unix(response.Rx.Sec, util.MicroToNanoSec(response.Rx.Usec))
		_, err = tran.Exec(
			insertResponse,
			src,
			dst,
			start.UnixNano(),
			from,
			response.Seq,
			response.ReplySize,
			response.ReplyTtl,
			response.ReplyProto,
			tx.UnixNano(),
			rx.UnixNano(),
			response.Rtt,
			response.ProbeIpid,
			response.ReplyIpid,
			response.IcmpType,
			response.IcmpCode,
			j,
		)
		if err != nil {
			tran.Rollback()
			return err
		}
		rr := response.RR
		if rr == nil {
			continue
		}
		for i, hop := range rr {
			hip, err := util.IPStringToInt32(hop)
			if err != nil {
				tran.Rollback()
				return err
			}
			tran.Exec(insertHop, src, dst, start.UnixNano(), from, i+1, hip, j)
		}
	}
	return tran.Commit()
}
