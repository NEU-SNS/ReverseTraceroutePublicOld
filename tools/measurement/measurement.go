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
	"fmt"
	"io"
	"net"
	"os"
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
	conFmt = "%s:%s@tcp(localhost:3306)/record_routes?parseTime=true"
)

var errDone = fmt.Errorf("Done")

func makeOut(outDir string) error {
	err := os.Mkdir(outDir, os.ModeDir|0755)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("Error creating outdir: %v", err)
	}
	return nil
}

func openOutFile(fname string) (*os.File, error) {
	return os.Create(fname)
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
	f, err := openOutFile(fmt.Sprintf("%s/%s.txt", ctx.String("out"), ctx.GlobalString("id")))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Got error: %v", err)
		return
	}
	defer f.Close()
	rr := ctx.GlobalBool("rr")
	src, err := openSource(ctx.String("src"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Got error opening source, %s: %v", ctx.String("src"), err)
		return
	}
	defer src.Close()
	var srcs []string
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
	var dsts []string
	line = 0
	for {
		ip, err := read.ReadString('\n')
		if err != nil && err != io.EOF {
			fmt.Fprintf(os.Stderr, "Got error: %v", err)
			return
		}
		if err == io.EOF {
			if len(dsts) == 0 {
				return
			}
			_, err := runMeasurements(srcs, dsts, rr, f)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error running measurements: %s\n", err)
				return
			}
			return
		}
		ip1 := net.ParseIP(strings.TrimSpace(ip))
		if ip1 == nil {
			fmt.Fprintf(os.Stderr, "Invalid ip at line: %d, %s\n", line+1, ip)
			return
		}
		dsts = append(dsts, ip1.String())
		line++
		if line == bs {
			_, err := runMeasurements(srcs, dsts, rr, f)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error running measurements: %s\n", err)
				return
			}
			line = 0
			dsts = make([]string, 0)
			continue
		}

	}
}

func runMeasurements(srcs, dsts []string, rr bool, out *os.File) ([]uint32, error) {
	pingReq := &dm.PingArg{
		Pings: make([]*dm.PingMeasurement, 0),
	}
	for _, src := range srcs {
		for _, dst := range dsts {
			s, _ := util.IPStringToInt32(src)
			d, _ := util.IPStringToInt32(dst)
			pingReq.Pings = append(pingReq.Pings, &dm.PingMeasurement{
				Src:     s,
				Dst:     d,
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
	st, err := cl.Ping(ctx.Background())
	if err != nil {
		return nil, fmt.Errorf("Failed to run ping: %v", err)
	}
	st.Send(pingReq)
	st.CloseSend()
	var ps []*dm.Ping
	var succ []uint32
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
		_, err = out.WriteString(pr.String() + "\n")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not write file: %v", err)
		}
		ps = append(ps, pr)
	}
	end := time.Now()
	fmt.Println("Finished:", end, "Took:", time.Since(start), "Received:", len(ps), "responses")
	return succ, nil
}

func main() {
	app := cli.NewApp()
	app.Name = "measurement"
	app.Usage = "Run ping measurements using the revtr measurement infrastructure"
	app.Version = "0.0.5"
	app.Flags = []cli.Flag{
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
