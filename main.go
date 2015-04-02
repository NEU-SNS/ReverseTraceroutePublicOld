package main

import (
	"flag"
	"github.com/NEU-SNS/ReverseTraceroute/mproc"
	"github.com/NEU-SNS/ReverseTraceroute/mproc/proc"
)

func main() {
	flag.Parse()
	mp := mproc.New()
	proc := proc.New("/bin/sleep", nil, "10")
	mp.ManageProcess(proc, true)
	select {}
}
