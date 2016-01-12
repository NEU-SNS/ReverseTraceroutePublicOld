package main

import (
	"bufio"
	"compress/bzip2"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/NEU-SNS/ReverseTraceroute/dataaccess"
)

var filePath string
var dbName string

const usage = `loadclusters -f <file> -d <dbname>`

func init() {
	flag.StringVar(&filePath, "f", "", "The path to the file that contains the alias data")
	flag.StringVar(&dbName, "d", "", "The name of the db to use")
}

func main() {
	flag.Parse()
	if filePath == "" {
		fmt.Println(usage)
		os.Exit(1)
	}

	if dbName == "" {
		fmt.Println(usage)
		os.Exit(1)
	}

	_, pref, err := net.ParseCIDR("224.0.0.0/3")
	if err != nil {
		fmt.Println("Failed to parse 224.0.0.0/3")
		os.Exit(1)
	}
	var scan *bufio.Scanner
	f, err := os.Open(filePath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer f.Close()
	if filepath.Ext(filePath) == "bz2" {
		scan = bufio.NewScanner(bzip2.NewReader(f))
	} else {
		scan = bufio.NewScanner(f)
	}
	var dc dataaccess.DbConfig
	var conf dataaccess.Config
	conf.Host = "localhost"
	conf.Db = dbName
	conf.Password = "password"
	conf.Port = "3306"
	conf.User = "revtr"
	dc.ReadConfigs = append(dc.ReadConfigs, conf)
	dc.WriteConfigs = append(dc.WriteConfigs, conf)
	da, err := dataaccess.New(dc)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer da.Close()
	for scan.Scan() {
		line := scan.Text()
		// Skip comment
		if line[0] == '#' {
			continue
		}
		// line format node N<node #>: <ip1> <ip2> ... <ipN>
		initial := strings.Split(line, ":")
		if len(initial) != 2 {
			fmt.Println("Invalid line: ", line)
			os.Exit(1)
		}
		header := initial[0]
		nodeSplit := strings.Split(header, " ")
		if len(nodeSplit) != 2 {
			fmt.Println("Invalid line header: ", header)
			os.Exit(1)
		}
		num := nodeSplit[1][1:]
		numAsInt, err := strconv.Atoi(num)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		ips := strings.Split(strings.TrimSpace(initial[1]), " ")
		var ipsSanitize []net.IP
		for _, ip := range ips {
			netIP := net.ParseIP(ip)
			if pref.Contains(netIP) {
				continue
			}
			ipsSanitize = append(ipsSanitize, netIP)
		}
		err = da.StoreAlias(numAsInt, ipsSanitize)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
}
