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
	"container/list"
	"database/sql"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"math/rand"
	"net"
	"os"
	"runtime"
	"strings"
	"time"

	"google.golang.org/grpc"

	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	plc "github.com/NEU-SNS/ReverseTraceroute/plcontrollerapi"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	"github.com/codegangsta/cli"
	ctx "golang.org/x/net/context"
)

import _ "github.com/go-sql-driver/mysql"

var (
	conFmt string = "%s:%s@tcp(localhost:3306)/record_routes?parseTime=true"
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

type Prefix struct {
	net   *net.IPNet
	ones  int
	bits  int
	order []int
}

var done = fmt.Errorf("Done")

func (p *Prefix) GetRandom() (string, error) {
	if p.order == nil {
		rand.Seed(time.Now().UnixNano())
		max := p.bits - p.ones
		maxVal := math.Pow(2, float64(max))
		p.order = rand.Perm(int(maxVal))
	}
	if len(p.order) == 0 {
		return "", done
	}
	val := p.order[0]
	p.order = p.order[1:]
	ips := p.net.IP.String()
	splits := strings.Split(ips, ".")
	final := splits[0:3]
	final = append(final, fmt.Sprintf("%d", val))
	return strings.Join(final, "."), nil
}

var pr *bufio.Reader

func getPrefixes(limit int) (*list.List, error) {
	li := list.New()
	for i := 0; i < limit; i++ {
		ip, err := pr.ReadString('\n')
		if err != nil && err != io.EOF {
			return li, err
		}
		if err == io.EOF {
			return li, io.EOF
		}
		_, cidr, err := net.ParseCIDR(strings.TrimSpace(ip))
		if err != nil {
			return nil, err
		}
		o, l := cidr.Mask.Size()
		li.PushBack(&Prefix{net: cidr, ones: o, bits: l})
	}
	return li, nil
}

func makeOut(outDir string) error {
	err := os.Mkdir(outDir, os.ModeDir|0755)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("Error creating outdir: %v", err)
	}
	return nil
}

func openSource(src string) (io.ReadCloser, error) {
	srcFile, err := os.Open(src)
	if err != nil {
		return nil, fmt.Errorf("Failed to open src, %s: %v", src, err)
	}
	return srcFile, nil
}

func openDest(dst string) (io.ReadCloser, error) {
	dstFile, err := os.Open(dst)
	if err != nil {
		return nil, fmt.Errorf("Failed to open %s: %v", dst, err)
	}
	return dstFile, nil
}

func addrScan(ctx *cli.Context) {
	if !(ctx.IsSet("src") && ctx.IsSet("dst")) {
		fmt.Fprintf(os.Stderr, "Error: Missing argument from command prefix")
		return
	}
	bs := ctx.GlobalInt("b")
	if bs == 0 {
		fmt.Fprintf(os.Stderr, "Error: Batch size must be > 0")
		return
	}
	if ctx.GlobalString("id") == "" {
		fmt.Fprintf(os.Stderr, "Error: id must be set")
		return
	}
	var out string
	if out = ctx.String("out"); out == "" {
		fmt.Fprintf(os.Stderr, "Error: out must be set")
		return
	}
	if err := makeOut(out); err != nil {
		fmt.Fprintf(os.Stderr, "Got error: %v", err)
		return
	}
	rr := ctx.GlobalBool("rr")
	src, err := openSource(ctx.String("src"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Got error opening source, %s: %v", ctx.String("src"), err)
		return
	}
	defer src.Close()
	srcs := make([]string, 0)
	scanner := bufio.NewScanner(src)
	scanner.Split(bufio.ScanLines)
	line := 1
	for scanner.Scan() {
		ip := scanner.Text()
		ip1 := net.ParseIP(ip)
		if ip1 == nil {
			fmt.Fprintf(os.Stderr, "Invalid ip at line: %d, %s", line, ip)
			return
		}
		srcs = append(srcs, ip1.String())
		line++
	}
	dst, err := openDest(ctx.String("dst"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Got error: %v", err)
		return
	}

	defer dst.Close()
	read := bufio.NewReader(dst)
	dsts := make([]string, 0)
	for {
		ip, err := read.ReadString('\n')
		if err != nil && err != io.EOF {
			return
		}
		if err == io.EOF && line == 0 {
			break
		}
		ip1 := net.ParseIP(strings.TrimSpace(ip))
		if ip1 == nil {
			fmt.Fprintf(os.Stderr, "Invalid ip at line: %d, %s\n", line, ip)
			return
		}
		dsts = append(dsts, ip1.String())
		line++
		if line == bs {
			_, err := runMeasurements(srcs, dsts, ctx.GlobalString("id"), ctx.String("out"), rr, ctx.BoolT("db"), nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error running measurements: %s\n", err)
				return
			}
			line = 0
			dsts = make([]string, 0)
			<-time.After(time.Second)
			continue
		}
		if err == io.EOF {
			if len(dsts) == 0 {
				break
			}
			_, err := runMeasurements(srcs, dsts, ctx.GlobalString("id"), ctx.String("out"), rr, ctx.BoolT("db"), nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error running measurements: %s\n", err)
				return
			}
			break
		}

	}
}

func prefixScan(ctx *cli.Context) {
	if !(ctx.IsSet("src") && ctx.IsSet("dst")) {
		fmt.Fprintf(os.Stderr, "Error: Missing argument from command prefix")
		return
	}
	bs := ctx.GlobalInt("b")
	if bs == 0 {
		fmt.Fprintf(os.Stderr, "Error: Batch size must be > 0")
		return
	}
	if ctx.GlobalString("id") == "" {
		fmt.Fprintf(os.Stderr, "Error: id must be set")
		return
	}
	var out string
	if out = ctx.String("out"); out == "" {
		fmt.Fprintf(os.Stderr, "Error: out must be set")
		return
	}
	if err := makeOut(out); err != nil {
		fmt.Fprintf(os.Stderr, "Got error: %v", err)
		return
	}
	rr := ctx.GlobalBool("rr")
	src, err := openSource(ctx.String("src"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Got error opening source, %s: %v", ctx.String("src"), err)
		return
	}
	defer src.Close()
	srcs := make([]string, 0)
	scanner := bufio.NewScanner(src)
	scanner.Split(bufio.ScanLines)
	line := 1
	for scanner.Scan() {
		ip := scanner.Text()
		ip1 := net.ParseIP(ip)
		if ip1 == nil {
			fmt.Fprintf(os.Stderr, "Invalid ip at line: %d, %s", line, ip)
			return
		}
		srcs = append(srcs, ip1.String())
		line++
	}
	dst, err := openDest(ctx.String("dst"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Got error: %v", err)
		return
	}
	defer dst.Close()
	pr = bufio.NewReader(dst)
	prefixes, err := getPrefixes(bs)
	if err != nil && err != io.EOF {
		fmt.Fprintf(os.Stderr, "Got error: %v", err)
		return
	}
	var npre *list.List
	npre = prefixes
	for {
		prefixes = npre
		npre = list.New()
		if prefixes.Len() == 0 {
			return
		}

		dsts := make([]string, 0)
		for e := prefixes.Front(); e != nil; e = e.Next() {
			pre := e.Value.(*Prefix)
			ip, err := pre.GetRandom()
			if err == done {
				px, err := getPrefixes(1)
				if err != nil && err == io.EOF {
					continue
				}
				if err != io.EOF && err != nil {
					return
				}
				npre.PushBackList(px)
				continue
			}
			npre.PushBack(e.Value)
			dsts = append(dsts, ip)
		}
		succ, err := runMeasurements(srcs, dsts, ctx.GlobalString("id"), ctx.String("out"), rr, ctx.BoolT("db"), nil)
		if err != nil {
			return
		}
		for _, s := range succ {
			for e := npre.Front(); e != nil; e = e.Next() {
				pre := e.Value.(*Prefix)
				ip := net.ParseIP(s)
				if pre.net.Contains(ip) {
					curr := e
					e = e.Prev()
					npre.Remove(curr)
					px, err := getPrefixes(1)
					if err != nil && err != io.EOF {
						fmt.Fprintf(os.Stderr, "Failed to get prefix: %s", err)
						return
					}
					if err == io.EOF {
						continue
					}
					npre.PushBackList(px)
				}
			}
		}
	}
}

func runMeasurements(srcs, dsts []string, id, outDir string, rr, wdb bool, db *sql.DB) ([]string, error) {
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
	opts := []grpc.DialOption{grpc.WithInsecure()}
	conn, err := grpc.Dial("rhansen2.revtr.ccs.neu.edu:4380", opts...)
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to controller: %v", err)
	}
	defer conn.Close()
	cl := plc.NewPLControllerClient(conn)
	fmt.Println("Num of requests:", len(pingReq.Pings))
	start := time.Now()
	fmt.Println("Starting:", start)
	st, err := cl.Ping(ctx.Background(), pingReq)
	if err != nil {
		return nil, fmt.Errorf("Failed to run ping: %v", err)
	}
	ps := make([]*dm.Ping, 0)
	succ := make([]string, 0)
	for {
		pr, err := st.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Error while running measurement: %v", err)
		}
		if pr.Responses != nil && len(pr.Responses) > 0 {
			succ = append(succ, pr.Dst)
		}
		err = os.Mkdir(fmt.Sprintf("%s/%s", outDir, pr.Src), os.ModeDir|0755)
		if err != nil && !os.IsExist(err) {
			fmt.Fprintln(os.Stderr, "Error creating dir: %v", pr.Src)
		}
		err = ioutil.WriteFile(fmt.Sprintf("%s/%s/%s_%s", outDir, pr.Src, pr.Src, pr.Dst), []byte(pr.String()), 0666)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not write file: %v", err)
		}
		if wdb && db != nil {
			err = storePing(pr, db, id)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error storing ping: %v", err)
			}
		}
		ps = append(ps, pr)
	}
	end := time.Now()
	fmt.Println("Finished:", end, "Took:", time.Since(start), "Received:", len(ps), "responses")
	return succ, nil
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	app := cli.NewApp()
	app.Name = "measurement"
	app.Usage = "Run ping measurements using the revtr measurement infrastructure"
	app.Version = "0.0.4"
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "db",
			Usage: "Write Results to the DB",
		},
		cli.StringFlag{
			Name:  "user",
			Usage: "User name for the db",
		},
		cli.StringFlag{
			Name:  "password",
			Usage: "Password for the db",
		},
		cli.StringFlag{
			Name:  "id",
			Usage: "Id to use for the measurement set",
		},
		cli.BoolTFlag{
			Name:  "rr",
			Usage: "Use the Record Route option",
		},
		cli.IntFlag{
			Name:  "b",
			Usage: "Size of batches",
		},
	}
	app.Commands = []cli.Command{
		cli.Command{
			Name:  "prefix",
			Usage: "Scan a prefix randomly to find a responding address",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "src",
					Usage: "File containing source addresses",
				},
				cli.StringFlag{
					Name:  "dst",
					Usage: "File containing list of prefixes",
				},
				cli.StringFlag{
					Name:  "out",
					Usage: "Directory to write results to",
				},
			},
			Action: prefixScan,
		},
		cli.Command{
			Name:  "addr",
			Usage: "Ping all IPs in file",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "src",
					Usage: "File containing source addresses",
				},
				cli.StringFlag{
					Name:  "dst",
					Usage: "File containing list of prefixes",
				},
				cli.StringFlag{
					Name:  "out",
					Usage: "Directory to write results to",
				},
			},
			Action: addrScan,
		},
	}
	app.Run(os.Args)
}

func storePing(p *dm.Ping, db *sql.DB, runId string) error {
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
