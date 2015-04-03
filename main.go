package main

import (
	"flag"
	"github.com/NEU-SNS/ReverseTraceroute/controller"
	"runtime"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
	controller.Start("tcp", "45000")
}
