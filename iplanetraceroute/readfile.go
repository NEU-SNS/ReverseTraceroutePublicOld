package iplane

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"github.com/NEU-SNS/ReverseTraceroute/util"
)

// Traceroute is the traceroute as stored in iPlane data outputs
type Traceroute struct {
	Dest    net.IP
	NumHops int32
	Hops    []Hop
}

func (tr Traceroute) String() string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("destination: %v, hops: %d\n", tr.Dest, tr.NumHops))
	for i, hop := range tr.Hops {
		buf.WriteString(fmt.Sprintf("%d: %v %f %d\n", i, hop.IP, hop.Lat, hop.TTL))
	}
	return buf.String()
}

// Hop represents a hop in the traceroute
type Hop struct {
	Lat float32
	IP  net.IP
	TTL int32
}

// TracerouteScanner scans a file for Traceroutes
type TracerouteScanner struct {
	err         error
	f           io.Reader
	tr          *Traceroute
	size        int32
	curr        int32
	initialized bool
	clientID    int32
	uniqueID    int32
	length      int32
}

func (tr *TracerouteScanner) initialize() error {
	tr.initialized = true
	tr.curr = 0
	err := binary.Read(tr.f, binary.LittleEndian, &tr.clientID)
	if err != nil {
		return err
	}
	err = binary.Read(tr.f, binary.LittleEndian, &tr.uniqueID)
	if err != nil {
		return err
	}
	err = binary.Read(tr.f, binary.LittleEndian, &tr.size)
	if err != nil {
		return err
	}
	err = binary.Read(tr.f, binary.LittleEndian, &tr.length)
	if err != nil {
		return err
	}
	return nil
}

// NewTracerouteScanner creates a new TracerouteScanner using the reader f
func NewTracerouteScanner(f io.Reader) *TracerouteScanner {
	return &TracerouteScanner{
		f: f,
	}
}

// Scan advances to the next traceroute
func (tr *TracerouteScanner) Scan() bool {
	if !tr.initialized {
		err := tr.initialize()
		if err == io.EOF {
			tr.err = nil
			return false
		}
		if err != nil {
			tr.err = err
			return false
		}
	}
	if tr.curr >= tr.size {
		tr.initialized = false
		err := tr.initialize()
		if err == io.EOF {
			tr.err = nil
			return false
		}
		if err != nil {
			tr.err = err
			return false
		}
	}
	trace := &Traceroute{}
	var hops, ttl int32
	var ip uint32
	err := binary.Read(tr.f, binary.LittleEndian, &ip)
	if err != nil {
		tr.err = err
		return false
	}
	trace.Dest = util.Int32ToIP(ip)
	err = binary.Read(tr.f, binary.LittleEndian, &hops)
	if err != nil {
		tr.err = err
		return false
	}
	trace.NumHops = hops
	for j := int32(0); j < hops; j++ {
		var lat float32
		var hip uint32
		hop := Hop{}
		err = binary.Read(tr.f, binary.LittleEndian, &hip)
		if err != nil {
			tr.err = err
			return false
		}
		hop.IP = util.Int32ToIP(hip)
		err = binary.Read(tr.f, binary.LittleEndian, &lat)
		if err != nil {
			tr.err = err
			return false
		}
		hop.Lat = lat
		err = binary.Read(tr.f, binary.LittleEndian, &ttl)
		if err != nil {
			tr.err = err
			return false
		}
		hop.TTL = ttl
		if hop.TTL > 512 {
			tr.err = fmt.Errorf("TTL too high, possible file corruption")
			return false
		}
		trace.Hops = append(trace.Hops, hop)
	}
	tr.tr = trace
	tr.curr++
	return true
}

// Traceroute gets the Traceroute from the last scan
func (tr *TracerouteScanner) Traceroute() Traceroute {
	return *tr.tr
}

// Err returns any error that might have occured while scanning
func (tr *TracerouteScanner) Err() error {
	return tr.err
}
