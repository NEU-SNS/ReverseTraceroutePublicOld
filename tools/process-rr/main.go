package main

import (
	"os"

	"github.com/NEU-SNS/ReverseTraceroute/tools/process-rr/processrr"
)

func main() {
	os.Exit(processrr.Main())
}
