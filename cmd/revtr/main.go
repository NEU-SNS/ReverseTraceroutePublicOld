package main

import (
	"flag"
	"net/http"

	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/revtr"
)

func main() {
	flag.Parse()
	/*
		dsts := []string{

			"4.15.166.15",
			"4.15.35.143",
			"4.34.58.15",
			"4.34.58.28",
			"4.34.58.41",
			"4.34.58.54",
			"4.35.94.15",
			"4.35.238.207",
			"4.35.238.220",
			"4.35.238.233",
			"4.35.238.246",
			"4.71.157.143",
			"4.71.157.156",
			"4.71.157.169",
			"4.71.157.182",
			"4.71.210.205",
			"23.228.128.169",
			"23.228.128.182",
			"38.65.210.207",
			"38.90.140.143",
			"38.98.51.13",
			"38.102.0.77",
			"38.102.163.143",
			"38.106.70.139",
			"38.107.216.18",
			"38.109.21.15",
			"41.231.21.15",
			"41.231.21.28",
			"41.231.21.41",
			"61.7.252.15",
			"61.7.252.41",
			"63.243.224.15",
			"63.243.240.79",
			"64.86.132.79",
			"64.86.148.143",
			"64.86.200.207",
			"65.46.46.143",
			"65.46.46.156",
			"65.46.46.169",
			"65.46.46.182",
			"66.110.32.79",
			"66.110.73.41",
			"66.110.73.54",
			"66.198.10.143",
			"66.198.24.79",
			"66.198.24.92",
			"66.198.24.105",
			"66.198.24.118",
			"67.106.215.207",
			"67.106.215.220",
		}
		src := "4.15.166.15"

		_, srvs, err := net.LookupSRV("controller", "tcp", "revtr.ccs.neu.edu")
		if err != nil {
			panic(err)
		}
		connstr := fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port)
		cc, err := grpc.Dial(connstr, grpc.WithInsecure())
		if err != nil {
			panic(err)
		}
		defer cc.Close()
		cli := client.New(context.Background(), cc)
		_, srvs, err = net.LookupSRV("atlas", "tcp", "revtr.ccs.neu.edu")
		if err != nil {
			panic(err)
		}
		connstrat := fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port)
		c2, err := grpc.Dial(connstrat, grpc.WithInsecure())
		if err != nil {
			panic(err)
		}
		defer c2.Close()
		atl := at.New(context.Background(), c2)
		_, srvs, err = net.LookupSRV("vpservice", "tcp", "revtr.ccs.neu.edu")
		if err != nil {
			panic(err)
		}
		connvp := fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port)
		c3, err := grpc.Dial(connvp, grpc.WithInsecure())
		if err != nil {
			panic(err)
		}
		defer c3.Close()
		vps := vpservice.New(context.Background(), c3)

		var dc dataaccess.DbConfig
		var conf dataaccess.Config
		conf.Host = "localhost"
		conf.Db = "revtr"
		conf.Password = "password"
		conf.Port = "3306"
		conf.User = "revtr"
		dc.ReadConfigs = append(dc.ReadConfigs, conf)
		dc.WriteConfigs = append(dc.WriteConfigs, conf)
		da, err := dataaccess.New(dc)
		if err != nil {
			panic(err)
		}
		var revtrs []revtr.ReverseTracerouteReq
		for _, d := range dsts {
			revtrs = append(revtrs, revtr.ReverseTracerouteReq{Src: src, Dst: d})
		}

		var res []*revtr.ReverseTraceroute
		var wg sync.WaitGroup
		for _, rt := range revtrs {
			wg.Add(1)
			go func(r revtr.ReverseTracerouteReq) {
				rr, err := revtr.RunReverseTraceroute(r, true, cli, atl, vps, da, da)
				wg.Done()
				if err != nil {
					fmt.Println(err)
					return
				}
				res = append(res, rr)
			}(rt)
		}
		wg.Wait()
		for _, rev := range res {
			fmt.Println(rev)
		}
	*/
	sr := revtr.NewV1Revtr(nil)

	http.HandleFunc(sr.Route, sr.Handle)
	for {
		log.Error(http.ListenAndServe(":8080", nil))
	}
}
