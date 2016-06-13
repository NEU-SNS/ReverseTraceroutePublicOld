package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/NEU-SNS/ReverseTraceroute/log"
)

var (
	path string
	src  string
)

func init() {
	flag.StringVar(&path, "p", "", "The path to the destinations file")
	flag.StringVar(&src, "s", "", "The src to send the revtrs from")
	flag.Usage = func() {
		fmt.Println("revtrtest -p <path to dsts> -s <src>")
	}
}

// Revtr ...
type Revtr struct {
	Src       string `json:"src"`
	Dst       string `json:"dst"`
	Staleness uint32 `json:"staleness"`
}

func main() {
	flag.Parse()
	if path == "" || src == "" {
		flag.Usage()
		os.Exit(1)
	}
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	scan := bufio.NewScanner(f)
	var revs []Revtr
	for scan.Scan() {
		dst := scan.Text()
		if dst == "" {
			continue
		}
		revs = append(revs, Revtr{
			Src:       src,
			Dst:       dst,
			Staleness: 10000,
		})
	}
	if err := scan.Err(); err != nil {
		f.Close()
		log.Error(err)
		os.Exit(1)
	}
	b, err := json.Marshal(struct {
		Revtrs []Revtr `json:"revtrs"`
	}{
		Revtrs: revs,
	})
	if err != nil {
		f.Close()
		log.Error(err)
		os.Exit(1)
	}
	buf := bytes.NewBuffer(b)
	fmt.Printf(buf.String())
	req, err := http.NewRequest("POST", "http://localhost:8080/api/v1/revtr", buf)
	if err != nil {
		f.Close()
		log.Error(err)
		os.Exit(1)
	}
	req.Header.Add("Revtr-Key", "aaaaaa")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		f.Close()
		log.Error(err)
		os.Exit(1)
	}
	bb := make([]byte, 500)
	_, err = resp.Body.Read(bb)
	if err != nil && err != io.EOF {
		f.Close()
		log.Error(err)
		os.Exit(1)
	}
	resp.Body.Close()
	fmt.Println(string(bb))
}
