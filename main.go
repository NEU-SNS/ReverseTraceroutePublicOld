// Package main is a dummy for testing
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"golang.org/x/net/context"

	"github.com/NEU-SNS/ReverseTraceroute/datamodel"

	"github.com/NEU-SNS/ReverseTraceroute/plcontroller/pb"
	"google.golang.org/grpc"
)

func main() {
	cc, err := grpc.Dial(fmt.Sprintf("%s:%d", "plcontroller.revtr.ccs.neu.edu", 4380), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer cc.Close()
	cl := pb.NewPLControllerClient(cc)
	f, err := os.Open("/home/rhansen2/test_addresses.txt")
	if err != nil {
		panic(err)
	}
	var pings []*datamodel.PingMeasurement
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		ip, err := strconv.Atoi(line)
		if err != nil {
			panic(err)
		}
		pings = append(pings, &datamodel.PingMeasurement{
			Src:     2150272554,
			Dst:     uint32(ip),
			Count:   "1",
			Timeout: 240,
		})
		pings = append(pings, &datamodel.PingMeasurement{
			Src:     2161960706,
			Dst:     uint32(ip),
			Count:   "1",
			Timeout: 240,
		})
		pings = append(pings, &datamodel.PingMeasurement{
			Src:     2154859361,
			Dst:     uint32(ip),
			Count:   "1",
			Timeout: 240,
		})
		pings = append(pings, &datamodel.PingMeasurement{
			Src:     2168430437,
			Dst:     uint32(ip),
			Count:   "1",
			Timeout: 240,
		})
		pings = append(pings, &datamodel.PingMeasurement{
			Src:     2197624185,
			Dst:     uint32(ip),
			Count:   "1",
			Timeout: 240,
		})
		pings = append(pings, &datamodel.PingMeasurement{
			Src:     2389115394,
			Dst:     uint32(ip),
			Count:   "1",
			Timeout: 240,
		})
		pings = append(pings, &datamodel.PingMeasurement{
			Src:     2502643732,
			Dst:     uint32(ip),
			Count:   "1",
			Timeout: 240,
		})
		pings = append(pings, &datamodel.PingMeasurement{
			Src:     2572812579,
			Dst:     uint32(ip),
			Count:   "1",
			Timeout: 240,
		})
		/*
			pings = append(pings, &datamodel.PingMeasurement{
				Src:     3093894991,
				Dst:     uint32(ip),
				Count:   "1",
				Timeout: 240,
			})
			pings = append(pings, &datamodel.PingMeasurement{
				Src:     3025625449,
				Dst:     uint32(ip),
				Count:   "1",
				Timeout: 240,
			})
			pings = append(pings, &datamodel.PingMeasurement{
				Src:     3025625436,
				Dst:     uint32(ip),
				Count:   "1",
				Timeout: 240,
			})
			pings = append(pings, &datamodel.PingMeasurement{
				Src:     3025625423,
				Dst:     uint32(ip),
				Count:   "1",
				Timeout: 240,
			})
			pings = append(pings, &datamodel.PingMeasurement{
				Src:     2938982172,
				Dst:     uint32(ip),
				Count:   "1",
				Timeout: 240,
			})
			pings = append(pings, &datamodel.PingMeasurement{
				Src:     2938982159,
				Dst:     uint32(ip),
				Count:   "1",
				Timeout: 240,
			})
			pings = append(pings, &datamodel.PingMeasurement{
				Src:     2915894313,
				Dst:     uint32(ip),
				Count:   "1",
				Timeout: 240,
			})
			pings = append(pings, &datamodel.PingMeasurement{
				Src:     2915894300,
				Dst:     uint32(ip),
				Count:   "1",
				Timeout: 240,
			})
			pings = append(pings, &datamodel.PingMeasurement{
				Src:     2915894287,
				Dst:     uint32(ip),
				Count:   "1",
				Timeout: 240,
			})
			pings = append(pings, &datamodel.PingMeasurement{
				Src:     2915894185,
				Dst:     uint32(ip),
				Count:   "1",
				Timeout: 240,
			})
			pings = append(pings, &datamodel.PingMeasurement{
				Src:     2915894070,
				Dst:     uint32(ip),
				Count:   "1",
				Timeout: 240,
			})
			pings = append(pings, &datamodel.PingMeasurement{
				Src:     2915894057,
				Dst:     uint32(ip),
				Count:   "1",
				Timeout: 240,
			})
			pings = append(pings, &datamodel.PingMeasurement{
				Src:     2915894031,
				Dst:     uint32(ip),
				Count:   "1",
				Timeout: 240,
			})
			pings = append(pings, &datamodel.PingMeasurement{
				Src:     2915893494,
				Dst:     uint32(ip),
				Count:   "1",
				Timeout: 240,
			})
			pings = append(pings, &datamodel.PingMeasurement{
				Src:     2915893481,
				Dst:     uint32(ip),
				Count:   "1",
				Timeout: 240,
			})
			pings = append(pings, &datamodel.PingMeasurement{
				Src:     2915893455,
				Dst:     uint32(ip),
				Count:   "1",
				Timeout: 240,
			})
		*/
	}
	pings = append(pings, pings...)
	pings = append(pings, pings...)
	pings = append(pings, pings...)
	pings = append(pings, pings...)
	pings = append(pings, pings...)
	pings = append(pings, pings...)
	pings = append(pings, pings...)
	pm := datamodel.PingArg{
		Pings: pings,
	}
	st, err := cl.Ping(context.Background())
	if err != nil {
		panic(err)
	}
	start := time.Now()
	st.Send(&pm)
	st.CloseSend()
	var res []*datamodel.Ping
	for {
		p, err := st.Recv()
		if err == io.EOF {
			break
		}

		if err != nil {
			panic(err)
		}
		if p.Error == "" {
			res = append(res, p)
		}
	}
	fmt.Println("Ran", len(pings), "got", len(res), "in", time.Since(start))
}
