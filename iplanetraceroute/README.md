# iplane
--
    import "github.com/NEU-SNS/ReverseTraceroute/iplanetraceroute"


## Usage

#### type Hop

```go
type Hop struct {
	Lat float32
	IP  net.IP
	TTL int32
}
```

Hop represents a hop in the traceroute

#### type Traceroute

```go
type Traceroute struct {
	Dest    net.IP
	NumHops int32
	Hops    []Hop
}
```

Traceroute is the traceroute as stored in iPlane data outputs

#### func (Traceroute) String

```go
func (tr Traceroute) String() string
```

#### type TracerouteScanner

```go
type TracerouteScanner struct {
}
```

TracerouteScanner scans a file for Traceroutes

#### func  NewTracerouteScanner

```go
func NewTracerouteScanner(f io.Reader) *TracerouteScanner
```
NewTracerouteScanner creates a new TracerouteScanner using the reader f

#### func (*TracerouteScanner) Err

```go
func (tr *TracerouteScanner) Err() error
```
Err returns any error that might have occured while scanning

#### func (*TracerouteScanner) Scan

```go
func (tr *TracerouteScanner) Scan() bool
```
Scan advances to the next traceroute

#### func (*TracerouteScanner) Traceroute

```go
func (tr *TracerouteScanner) Traceroute() Traceroute
```
Traceroute gets the Traceroute from the last scan
