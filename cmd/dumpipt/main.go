package main

import (
	"fmt"
	"os"

	"github.com/NEU-SNS/ReverseTraceroute/iplanetraceroute"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: dumpipt <file>")
		os.Exit(1)
	}
	var file = os.Args[1]
	f, err := os.Open(file)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer f.Close()
	scan := iplane.NewTracerouteScanner(f)
	for scan.Scan() {
		fmt.Println(scan.Traceroute())
	}
	if err = scan.Err(); err != nil {
		fmt.Println(err)
		f.Close()
		os.Exit(1)
	}

}
