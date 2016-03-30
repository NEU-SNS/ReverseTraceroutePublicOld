package revtr

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"testing"

	"google.golang.org/grpc/metadata"

	"golang.org/x/net/context"

	at "github.com/NEU-SNS/ReverseTraceroute/atlas/client"
	am "github.com/NEU-SNS/ReverseTraceroute/atlas/mocks"
	"github.com/NEU-SNS/ReverseTraceroute/controller/client"
	cm "github.com/NEU-SNS/ReverseTraceroute/controller/mocks"
	"github.com/NEU-SNS/ReverseTraceroute/controller/pb"
	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
	mocks "github.com/NEU-SNS/ReverseTraceroute/revtr/mocks"
	vpservice "github.com/NEU-SNS/ReverseTraceroute/vpservice/client"
	vpm "github.com/NEU-SNS/ReverseTraceroute/vpservice/mocks"
	"github.com/stretchr/testify/mock"
)

var vp = []*datamodel.VantagePoint{
	&datamodel.VantagePoint{
		Hostname:     "test1.fake.com",
		Ip:           1239139955,
		Timestamp:    true,
		RecordRoute:  true,
		CanSpoof:     true,
		ReceiveSpoof: true,
		Site:         "fake.com",
	},
	&datamodel.VantagePoint{
		Hostname:     "test2.fake.com",
		Ip:           1239139958,
		Timestamp:    true,
		RecordRoute:  true,
		CanSpoof:     true,
		ReceiveSpoof: true,
		Site:         "fake.com",
	},
	&datamodel.VantagePoint{
		Hostname:     "test1.fake1.com",
		Ip:           1239139959,
		Timestamp:    true,
		RecordRoute:  true,
		CanSpoof:     true,
		ReceiveSpoof: true,
		Site:         "fake1.com",
	},
	&datamodel.VantagePoint{
		Hostname:     "test2.fake1.com",
		Ip:           1239139956,
		Timestamp:    true,
		RecordRoute:  true,
		CanSpoof:     true,
		ReceiveSpoof: true,
		Site:         "fake1.com",
	},
	&datamodel.VantagePoint{
		Hostname:     "test1.fake2.com",
		Ip:           1239139960,
		Timestamp:    true,
		RecordRoute:  true,
		CanSpoof:     true,
		ReceiveSpoof: true,
		Site:         "fake2.com",
	},
	&datamodel.VantagePoint{
		Hostname:     "test2.fake2.com",
		Ip:           1239139957,
		Timestamp:    true,
		RecordRoute:  true,
		CanSpoof:     true,
		ReceiveSpoof: true,
		Site:         "fake2.com",
	},
}

var cs = &mocks.ClusterSource{}
var mvps = &vpm.VPSource{}

func initTests() {
	cs.On("GetClusterIDByIP", mock.AnythingOfType("uint32")).Return(0, fmt.Errorf("None found"))
	mvps.On("GetVPs").Return(&datamodel.VPReturn{Vps: vp}, nil)
}

func TestMain(m *testing.M) {
	flag.Parse()
	initTests()
	os.Exit(m.Run())
}

type vpSourceMock struct {
}

func (vps vpSourceMock) GetOneVPPerSite() (*datamodel.VPReturn, error) {
	vp := []*datamodel.VantagePoint{
		&datamodel.VantagePoint{
			Hostname:     "test1.fake.com",
			Ip:           1239139955,
			Timestamp:    true,
			RecordRoute:  true,
			CanSpoof:     true,
			ReceiveSpoof: true,
			Site:         "fake.com",
		},
		&datamodel.VantagePoint{
			Hostname:     "test2.fake1.com",
			Ip:           1239139956,
			Timestamp:    true,
			RecordRoute:  true,
			CanSpoof:     true,
			ReceiveSpoof: true,
			Site:         "fake1.com",
		},
		&datamodel.VantagePoint{
			Hostname:     "test2.fake2.com",
			Ip:           1239139957,
			Timestamp:    true,
			RecordRoute:  true,
			CanSpoof:     true,
			ReceiveSpoof: true,
			Site:         "fake2.com",
		},
	}
	return &datamodel.VPReturn{
		Vps: vp,
	}, nil
}

var myIP = "129.10.113.189"

func TestCreateReverseTraceroute(t *testing.T) {
	as := &mocks.AdjacencySource{}
	as.On("GetAdjacenciesByIP1", mock.AnythingOfType("uint32")).Return([]datamodel.Adjacency{
		datamodel.Adjacency{
			IP1: 111111,
			IP2: 222222,
			Cnt: 10,
		},
	})
	as.On("GetAdjacenciesByIP2", mock.AnythingOfType("uint32")).Return(nil, nil)
	as.On("GetAdjacencyToDestByAddrAndDest24", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil, nil)
	rm := datamodel.RevtrMeasurement{
		Src:       myIP,
		Dst:       "8.8.8.8",
		Id:        1,
		Staleness: 60,
	}
	revtr := CreateReverseTraceroute(rm, false, false, &cm.Client{}, &am.Atlas{}, mvps, as, cs)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	t.Log(revtr)
}

func TestSymmetricAssumptions(t *testing.T) {
	as := &mocks.AdjacencySource{}
	as.On("GetAdjacenciesByIP1", mock.AnythingOfType("uint32")).Return([]datamodel.Adjacency{
		datamodel.Adjacency{
			IP1: 111111,
			IP2: 222222,
			Cnt: 10,
		},
	})
	as.On("GetAdjacenciesByIP2", mock.AnythingOfType("uint32")).Return(nil, nil)
	as.On("GetAdjacencyToDestByAddrAndDest24", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil, nil)

	rm := datamodel.RevtrMeasurement{
		Src:       myIP,
		Dst:       "8.8.8.8",
		Id:        1,
		Staleness: 60,
	}
	revtr := CreateReverseTraceroute(rm, false, false, &cm.Client{}, &am.Atlas{}, mvps, as, cs)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	if revtr.SymmetricAssumptions() != 0 {
		t.Errorf("SymmetricAssumptions, Expected: 0, Got: %d", revtr.SymmetricAssumptions())
	}
}

func TestDeadends(t *testing.T) {
	as := &mocks.AdjacencySource{}
	as.On("GetAdjacenciesByIP1", mock.AnythingOfType("uint32")).Return([]datamodel.Adjacency{
		datamodel.Adjacency{
			IP1: 111111,
			IP2: 222222,
			Cnt: 10,
		},
	})
	as.On("GetAdjacenciesByIP2", mock.AnythingOfType("uint32")).Return(nil, nil)
	as.On("GetAdjacencyToDestByAddrAndDest24", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil, nil)

	rm := datamodel.RevtrMeasurement{
		Src:       myIP,
		Dst:       "8.8.8.8",
		Id:        1,
		Staleness: 60,
	}
	revtr := CreateReverseTraceroute(rm, false, false, &cm.Client{}, &am.Atlas{}, mvps, as, cs)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	deadends := revtr.Deadends()
	if len(deadends) != 0 {
		t.Fatalf("Deadends, Expected: 0, Got: %d", len(revtr.Deadends()))
	}
}

func TestRRVPSInitializedForHop(t *testing.T) {
	as := &mocks.AdjacencySource{}
	as.On("GetAdjacenciesByIP1", mock.AnythingOfType("uint32")).Return(nil)
	as.On("GetAdjacenciesByIP2", mock.AnythingOfType("uint32")).Return(nil, nil)
	as.On("GetAdjacencyToDestByAddrAndDest24", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil, nil)
	rm := datamodel.RevtrMeasurement{
		Src:       myIP,
		Dst:       "8.8.8.8",
		Id:        1,
		Staleness: 60,
	}
	revtr := CreateReverseTraceroute(rm, false, false, &cm.Client{}, &am.Atlas{}, mvps, as, cs)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	testIP := "192.168.1.1"
	out := revtr.rrVPSInitializedForHop(testIP)
	if out != false {
		t.Fatalf("Expected false, got true")
	}
	revtr.setRRVPSForHop(testIP, []string{"test", "test1"})
	out = revtr.rrVPSInitializedForHop(testIP)
	if out != true {
		t.Fatalf("Expected true, got false")
	}
}

func TestCurrPath(t *testing.T) {
	as := &mocks.AdjacencySource{}
	as.On("GetAdjacenciesByIP1", mock.AnythingOfType("uint32")).Return(nil)
	as.On("GetAdjacenciesByIP2", mock.AnythingOfType("uint32")).Return(nil, nil)
	as.On("GetAdjacencyToDestByAddrAndDest24", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil, nil)
	rm := datamodel.RevtrMeasurement{
		Src:       myIP,
		Dst:       "8.8.8.8",
		Id:        1,
		Staleness: 60,
	}
	revtr := CreateReverseTraceroute(rm, false, false, &cm.Client{}, &am.Atlas{}, mvps, as, cs)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	expected := NewReversePath(myIP, "8.8.8.8", nil)
	currPath := revtr.CurrPath()
	if currPath.Dst != expected.Dst || currPath.Src != expected.Src {
		t.Fatalf("Src or dst not equal")
	}
}

func TestAddSegments(t *testing.T) {
	as := &mocks.AdjacencySource{}
	as.On("GetAdjacenciesByIP1", mock.AnythingOfType("uint32")).Return(nil)
	as.On("GetAdjacenciesByIP2", mock.AnythingOfType("uint32")).Return(nil, nil)
	as.On("GetAdjacencyToDestByAddrAndDest24", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil, nil)
	rm := datamodel.RevtrMeasurement{
		Src:       myIP,
		Dst:       "8.8.8.8",
		Id:        1,
		Staleness: 60,
	}
	revtr := CreateReverseTraceroute(rm, false, false, &cm.Client{}, &am.Atlas{}, mvps, as, cs)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	segs := []Segment{NewTrtoSrcRevSegment([]string{"8.8.8.8", myIP}, "8.8.8.8", myIP)}
	added := revtr.AddSegments(segs)
	if !added {
		t.Fatal("Failed to add Segments")
	}
}

func TestReaches(t *testing.T) {
	as := &mocks.AdjacencySource{}
	as.On("GetAdjacenciesByIP1", mock.AnythingOfType("uint32")).Return(nil)
	as.On("GetAdjacenciesByIP2", mock.AnythingOfType("uint32")).Return(nil, nil)
	as.On("GetAdjacencyToDestByAddrAndDest24", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil, nil)
	rm := datamodel.RevtrMeasurement{
		Src:       myIP,
		Dst:       "8.8.8.8",
		Id:        1,
		Staleness: 60,
	}
	revtr := CreateReverseTraceroute(rm, false, false, &cm.Client{}, &am.Atlas{}, mvps, as, cs)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	if revtr.Reaches() {
		t.Fatal("Reaches, reaches on creation of ReverseTraceroute")
	}
	segs := []Segment{NewTrtoSrcRevSegment([]string{"8.8.8.8", "10.0.0.2", myIP}, myIP, "8.8.8.8")}
	added := revtr.AddSegments(segs)
	if !added {
		t.Fatal("Failed to add Segments")
	}
	reaches := revtr.Reaches()
	if !reaches {
		t.Fatal("Failed to reach after adding reaching segment LastHop: ", revtr.LastHop(), " Got ", revtr.CurrPath().LastSeg())
	}
	t.Log(revtr.CurrPath())
}

var adjs = []datamodel.Adjacency{
	datamodel.Adjacency{
		IP1: 167772161,
		IP2: 167772162,
		Cnt: 2,
	},
	datamodel.Adjacency{
		IP1: 167772161,
		IP2: 167772163,
		Cnt: 4,
	},
	datamodel.Adjacency{
		IP1: 167772161,
		IP2: 167772164,
		Cnt: 3,
	},
	datamodel.Adjacency{
		IP1: 167772165,
		IP2: 167772161,
		Cnt: 2,
	},
	datamodel.Adjacency{
		IP1: 167772166,
		IP2: 134744072,
		Cnt: 9,
	},
	datamodel.Adjacency{
		IP1: 167772167,
		IP2: 167772161,
		Cnt: 8,
	},
}

var adjstodst = []datamodel.AdjacencyToDest{
	datamodel.AdjacencyToDest{
		Dest24:   134744072 >> 8,
		Address:  167772166,
		Adjacent: 167772161,
		Cnt:      7,
	},
	datamodel.AdjacencyToDest{
		Dest24:   134744072 >> 8,
		Address:  167772167,
		Adjacent: 167772161,
		Cnt:      6,
	},
	datamodel.AdjacencyToDest{
		Dest24:   134744072 >> 8,
		Address:  134744072,
		Adjacent: 167772162,
		Cnt:      3,
	},
	datamodel.AdjacencyToDest{
		Dest24:   134744072 >> 8,
		Address:  167772161,
		Adjacent: 167772164,
		Cnt:      2,
	},
}

func TestInitializeRRVPs(t *testing.T) {
	as := &mocks.AdjacencySource{}
	as.On("GetAdjacenciesByIP1", mock.AnythingOfType("uint32")).Return(nil)
	as.On("GetAdjacenciesByIP2", mock.AnythingOfType("uint32")).Return(nil, nil)
	as.On("GetAdjacencyToDestByAddrAndDest24", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil, nil)
	rm := datamodel.RevtrMeasurement{
		Src:       myIP,
		Dst:       "8.8.8.8",
		Id:        1,
		Staleness: 60,
	}
	revtr := CreateReverseTraceroute(rm, false, false, &cm.Client{}, &am.Atlas{}, mvps, as, cs)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	testIP := "8.8.8.8"
	err := revtr.InitializeRRVPs(testIP)
	if err != nil {
		t.Fatalf("Failed to initialize RRVPs Expected[<nil>], Got[%v]", err)
	}
	if rl, ok := revtr.RRHop2RateLimit[testIP]; ok {
		if rl != RateLimit {
			t.Fatalf("Failed to init RR Rate limit, Epected %d, Got %d", RateLimit, rl)
		}
	} else {
		t.Fatalf("Failed to initalize for %s", testIP)
	}
	if ss, ok := revtr.RRHop2VPSLeft[testIP]; ok {
		if len(ss) < 1 {
			t.Fatalf("Failed to initialize RR vps, Got: %v", ss)
		}
		t.Log(ss)
	} else {
		t.Fatalf("Failed to initalize for %s, no vps in map", testIP)
	}
}

func TestGetRRVPs(t *testing.T) {
	as := &mocks.AdjacencySource{}
	as.On("GetAdjacenciesByIP1", mock.AnythingOfType("uint32")).Return(adjs, nil)
	as.On("GetAdjacenciesByIP2", mock.AnythingOfType("uint32")).Return(nil, nil)
	as.On("GetAdjacencyToDestByAddrAndDest24", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil, nil)
	rm := datamodel.RevtrMeasurement{
		Src:       myIP,
		Dst:       "8.8.8.8",
		Id:        1,
		Staleness: 60,
	}
	revtr := CreateReverseTraceroute(rm, false, false, &cm.Client{}, &am.Atlas{}, mvps, as, cs)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	vps, target := revtr.GetRRVPs("8.8.8.8")
	if vps, ok := revtr.RRHop2VPSLeft["8.8.8.8"]; ok {
		t.Log(vps)
	} else {
		t.Fatal("VP not initialized")
	}
	if vps[0] != "non_spoofed" || target != "8.8.8.8" {
		t.Fatal("Failed to get non_spoofed on first call")
	}
	vps, target = revtr.GetRRVPs("8.8.8.8")
	if vps[0] == "non_spoofed" || target != "8.8.8.8" {
		t.Fatal("Failed, got non_spoofed on second call")
	}
}

func TestInitializeTSAdjacents(t *testing.T) {
	as := &mocks.AdjacencySource{}
	as.On("GetAdjacenciesByIP1", mock.AnythingOfType("uint32")).Return(adjs, nil)
	as.On("GetAdjacenciesByIP2", mock.AnythingOfType("uint32")).Return(nil, nil)
	as.On("GetAdjacencyToDestByAddrAndDest24", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil, nil)
	rm := datamodel.RevtrMeasurement{
		Src:       myIP,
		Dst:       "8.8.8.8",
		Id:        1,
		Staleness: 60,
	}
	revtr := CreateReverseTraceroute(rm, false, false, &cm.Client{}, &am.Atlas{}, mvps, as, cs)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	err := revtr.InitializeTSAdjacents("8.8.8.8")
	if err != nil {
		log.Fatal(err)
	}
	if adjs, ok := revtr.TSHop2AdjsLeft["8.8.8.8"]; !ok {
		log.Fatal("Failed to init TS Adjs ", adjs)
	}
}

func TestGetTSAdjacents(t *testing.T) {
	as := &mocks.AdjacencySource{}
	as.On("GetAdjacenciesByIP1", mock.AnythingOfType("uint32")).Return(adjs, nil)
	as.On("GetAdjacenciesByIP2", mock.AnythingOfType("uint32")).Return(nil, nil)
	as.On("GetAdjacencyToDestByAddrAndDest24", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil, nil)
	rm := datamodel.RevtrMeasurement{
		Src:       myIP,
		Dst:       "8.8.8.8",
		Id:        1,
		Staleness: 60,
	}
	revtr := CreateReverseTraceroute(rm, false, false, &cm.Client{}, &am.Atlas{}, mvps, as, cs)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	adjs := revtr.GetTSAdjacents("8.8.8.8")
	if len(adjs) == 0 {
		t.Fatalf("Got no Adjs %d", len(adjs))
	}
	t.Log(adjs)
}

func TestLength(t *testing.T) {
	as := &mocks.AdjacencySource{}
	as.On("GetAdjacenciesByIP1", mock.AnythingOfType("uint32")).Return(nil)
	as.On("GetAdjacenciesByIP2", mock.AnythingOfType("uint32")).Return(nil, nil)
	as.On("GetAdjacencyToDestByAddrAndDest24", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil, nil)
	rm := datamodel.RevtrMeasurement{
		Src:       myIP,
		Dst:       "8.8.8.8",
		Id:        1,
		Staleness: 60,
	}
	revtr := CreateReverseTraceroute(rm, false, false, &cm.Client{}, &am.Atlas{}, mvps, as, cs)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	length := revtr.CurrPath().Length()
	if length == 0 {
		t.Fatalf("Got len: %d, Expected: %d", length, 1)
	}
	t.Log(length)
}

func TestPop(t *testing.T) {
	as := &mocks.AdjacencySource{}
	as.On("GetAdjacenciesByIP1", mock.AnythingOfType("uint32")).Return(nil)
	as.On("GetAdjacenciesByIP2", mock.AnythingOfType("uint32")).Return(nil, nil)
	as.On("GetAdjacencyToDestByAddrAndDest24", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil, nil)
	rm := datamodel.RevtrMeasurement{
		Src:       myIP,
		Dst:       "8.8.8.8",
		Id:        1,
		Staleness: 60,
	}
	revtr := CreateReverseTraceroute(rm, false, false, &cm.Client{}, &am.Atlas{}, mvps, as, cs)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	length := revtr.CurrPath().len()
	revtr.CurrPath().Pop()
	if length != revtr.CurrPath().len()+1 {
		t.Fatal("Failed to pop an item")
	}
	t.Log(length)
}

func TestHops(t *testing.T) {
	as := &mocks.AdjacencySource{}
	as.On("GetAdjacenciesByIP1", mock.AnythingOfType("uint32")).Return(nil)
	as.On("GetAdjacenciesByIP2", mock.AnythingOfType("uint32")).Return(nil, nil)
	as.On("GetAdjacencyToDestByAddrAndDest24", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil, nil)
	rm := datamodel.RevtrMeasurement{
		Src:       myIP,
		Dst:       "8.8.8.8",
		Id:        1,
		Staleness: 60,
	}
	revtr := CreateReverseTraceroute(rm, false, false, &cm.Client{}, &am.Atlas{}, mvps, as, cs)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	hops := revtr.Hops()
	if len(hops) == 0 {
		t.Fatal("Hops failed, expected 1, got 0")
	}
}

func TestFailed(t *testing.T) {
	as := &mocks.AdjacencySource{}
	as.On("GetAdjacenciesByIP1", mock.AnythingOfType("uint32")).Return(nil)
	as.On("GetAdjacenciesByIP2", mock.AnythingOfType("uint32")).Return(nil, nil)
	as.On("GetAdjacencyToDestByAddrAndDest24", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil, nil)
	rm := datamodel.RevtrMeasurement{
		Src:       myIP,
		Dst:       "8.8.8.8",
		Id:        1,
		Staleness: 60,
	}
	revtr := CreateReverseTraceroute(rm, false, false, &cm.Client{}, &am.Atlas{}, mvps, as, cs)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	if !revtr.Failed(true) {
		t.Fatal("Brand new revtr didn't fail Failed(true)")
	}
	if revtr.Failed(false) {
		t.Fatal("Brand new revtr failed Failed(false)")
	}
}

func TestFailCurrPath(t *testing.T) {
	as := &mocks.AdjacencySource{}
	as.On("GetAdjacenciesByIP1", mock.AnythingOfType("uint32")).Return(nil)
	as.On("GetAdjacenciesByIP2", mock.AnythingOfType("uint32")).Return(nil, nil)
	as.On("GetAdjacencyToDestByAddrAndDest24", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil, nil)
	rm := datamodel.RevtrMeasurement{
		Src:       myIP,
		Dst:       "8.8.8.8",
		Id:        1,
		Staleness: 60,
	}
	revtr := CreateReverseTraceroute(rm, false, false, &cm.Client{}, &am.Atlas{}, mvps, as, cs)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	hop := revtr.LastHop()
	revtr.FailCurrPath()
	if !revtr.DeadEnd[hop] {
		t.Fatal("LastHop not added to deadends")
	}
}

func TestAddAndReplaceSegment(t *testing.T) {
	as := &mocks.AdjacencySource{}
	as.On("GetAdjacenciesByIP1", mock.AnythingOfType("uint32")).Return(nil)
	as.On("GetAdjacenciesByIP2", mock.AnythingOfType("uint32")).Return(nil, nil)
	as.On("GetAdjacencyToDestByAddrAndDest24", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil, nil)
	rm := datamodel.RevtrMeasurement{
		Src:       myIP,
		Dst:       "8.8.8.8",
		Id:        1,
		Staleness: 60,
	}
	revtr := CreateReverseTraceroute(rm, false, false, &cm.Client{}, &am.Atlas{}, mvps, as, cs)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	startLen := revtr.len()
	seg := NewTrtoSrcRevSegment([]string{"8.8.8.8", "10.0.0.1", "10.0.0.2"}, "8.8.8.8", myIP)
	res := revtr.AddAndReplaceSegment(seg)
	if !res {
		t.Fatalf("Failed to add Seg")
	}
	endLen := revtr.len()
	if startLen >= endLen {
		t.Fatal("Failed to add path")
	}
	if revtr.CurrPath().len() != 1 {
		t.Fatal("Failed replacement ", revtr.CurrPath().len())
	}
}

func TestAddBackgroundTRSegment(t *testing.T) {
	as := &mocks.AdjacencySource{}
	as.On("GetAdjacenciesByIP1", mock.AnythingOfType("uint32")).Return(nil)
	as.On("GetAdjacenciesByIP2", mock.AnythingOfType("uint32")).Return(nil, nil)
	as.On("GetAdjacencyToDestByAddrAndDest24", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil, nil)
	rm := datamodel.RevtrMeasurement{
		Src:       myIP,
		Dst:       "8.8.8.8",
		Id:        1,
		Staleness: 60,
	}
	c := &cm.Client{}
	revtr := CreateReverseTraceroute(rm, false, false, c, &am.Atlas{}, mvps, as, cs)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	seg1 := NewRRRevSegment([]string{"8.8.8.8", "10.0.0.1", "192.168.1.1"}, myIP, "8.8.8.8")
	revtr.AddSegments([]Segment{seg1})
	seg := NewTrtoSrcRevSegment([]string{"10.0.0.1", "10.0.0.2", myIP}, myIP, "10.0.0.1")
	res := revtr.AddBackgroundTRSegment(seg)
	if !res {
		t.Fatal("Failed to add background tr seg")
	}
	if !revtr.Reaches() {
		t.Fatal("Adding reaching trsegment didn't reach")
	}
	t.Log(revtr.CurrPath())
}

type testConfig struct {
	as      AdjacencySource
	rm      datamodel.RevtrMeasurement
	exp     error
	backoff bool
	print   bool
	cl      client.Client
	a       at.Atlas
	vps     vpservice.VPSource
	cs      ClusterSource
	doInit  func(t *testConfig) error
}

func (t *testConfig) init() error {
	return t.doInit(t)
}

func TestReverseHopsAssumeSymmetric(t *testing.T) {
	as := &mocks.AdjacencySource{}
	for _, test := range []*testConfig{
		&testConfig{as: as, rm: datamodel.RevtrMeasurement{
			Src:       "8.8.8.8",
			Dst:       "10.0.0.1",
			Id:        1,
			Staleness: 60,
		}, exp: ErrNoHopFound, cl: &clientMock{}, a: &am.Atlas{}, vps: mvps, cs: cs,
			doInit: func(tc *testConfig) error {
				return nil
			}},
		&testConfig{
			as: as, rm: datamodel.RevtrMeasurement{
				Src:       "8.8.8.8",
				Dst:       myIP,
				Id:        2,
				Staleness: 60,
			}, exp: nil, cl: &clientMock{}, a: &am.Atlas{}, vps: mvps, cs: cs,
			doInit: func(tc *testConfig) error {
				return nil
			},
		},
	} {
		err := test.doInit(test)
		if err != nil {
			t.Fatalf("Failed to init test: Expected[<nil>], Got[%v]", err)
		}
		revtr := CreateReverseTraceroute(test.rm, test.backoff, test.print, test.cl, test.a, test.vps, test.as, test.cs)
		if revtr == nil {
			t.Fatalf("Failed to create ReverseTraceroute")
		}
		err = revtr.reverseHopsAssumeSymmetric()
		if err != test.exp {
			t.Fatalf("reverseHopsAssumeSymmetric failed: Expected[%v], Got[%v]", test.exp, err)
		}
		revtr = nil
	}
}

/*
func TestReverseHopsAssumeSymmetricWithPreviousSymmetric(t *testing.T) {
	as := &mocks.AdjacencySource{}
	as.On("GetAdjacenciesByIP1", mock.AnythingOfType("uint32")).Return(nil)
	as.On("GetAdjacenciesByIP2", mock.AnythingOfType("uint32")).Return(nil, nil)
	as.On("GetAdjacencyToDestByAddrAndDest24", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil, nil)
	rm := datamodel.RevtrMeasurement{
		Src:       myIP,
		Dst:       "8.8.8.8",
		Id:        1,
		Staleness: 60,
	}
	revtr := CreateReverseTraceroute(rm, false, false, &cm.Client{}, &am.Atlas{}, mvps, as, cs)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	err := revtr.reverseHopsAssumeSymmetric()
	if err != nil {
		t.Fatal(err)
	}
	err = revtr.reverseHopsAssumeSymmetric()
	if err != nil {
		t.Fatal(err)
	}
}
*/
type clientMock struct {
	traces []*datamodel.Traceroute
	pings  []*datamodel.Ping
}

type clientPingClientStreamFake struct {
	last  int
	pings []*datamodel.Ping
	clientStreamFake
}

type clientStreamFake struct {
	ctx context.Context
}

func (c clientStreamFake) CloseSend() error {
	return nil
}

func (c clientStreamFake) Header() (metadata.MD, error) {
	return nil, nil
}

func (c clientStreamFake) Trailer() metadata.MD {
	return nil
}

func (c clientPingClientStreamFake) Recv() (*datamodel.Ping, error) {
	if c.last >= len(c.pings) {
		return nil, io.EOF
	}
	idx := c.last
	c.last++
	return c.pings[idx], nil
}

func (c clientStreamFake) Context() context.Context {
	return c.ctx
}

func (c clientStreamFake) SendMsg(m interface{}) error {
	return nil
}

func (c clientStreamFake) RecvMsg(m interface{}) error {
	return nil
}

type clientTracerouteClientStreamFake struct {
	last   int
	traces []*datamodel.Traceroute
	clientMock
	clientStreamFake
}

func (c *clientTracerouteClientStreamFake) Recv() (*datamodel.Traceroute, error) {
	if c.last >= len(c.traces) {
		return nil, io.EOF
	}
	idx := c.last
	c.last++
	return c.traces[idx], nil
}

func (c clientMock) Traceroute(con context.Context, arg *datamodel.TracerouteArg) (controllerapi.Controller_TracerouteClient, error) {
	/*
		var traces []*datamodel.Traceroute
		meas := arg.GetTraceroutes()
		for _, trace := range meas {
			var t datamodel.Traceroute
			t.Dst = trace.Dst
			t.Src = trace.Src
			t.Hops = append(t.Hops, &datamodel.TracerouteHop{
				Addr: 167772162,
			})
			t.Hops = append(t.Hops, &datamodel.TracerouteHop{
				Addr: 167772163,
			})
			t.Hops = append(t.Hops, &datamodel.TracerouteHop{
				Addr: 167772164,
			})
			t.Hops = append(t.Hops, &datamodel.TracerouteHop{
				Addr: trace.Dst,
			})
			traces = append(traces, &t)
		}
	*/
	return &clientTracerouteClientStreamFake{clientStreamFake: clientStreamFake{ctx: context.Background()}, traces: c.traces}, nil
}

func (c clientMock) Ping(con context.Context, arg *datamodel.PingArg) (controllerapi.Controller_PingClient, error) {
	/*
		var pings []*datamodel.Ping
		meas := arg.GetPings()
		for _, ping := range meas {
			p := datamodel.Ping{}
			p.Dst = ping.Dst
			p.Src = ping.Src
			pr := datamodel.PingResponse{}
			pr.From = p.Dst
			if ping.RR {
				pr.RR = []uint32{
					3232235777,
					3232235777,
					3232235777,
					3232235777,
					3232235777,
					3232235777,
					134744072,
					167772162,
					167772163,
				}
			}
			pings = append(pings, &p)
		}
	*/
	return clientPingClientStreamFake{clientStreamFake: clientStreamFake{ctx: context.Background()}, pings: c.pings}, nil
}

func (c clientMock) GetVps(con context.Context, args *datamodel.VPRequest) (*datamodel.VPReturn, error) {
	return nil, nil
}

func (c clientMock) ReceiveSpoofedProbes(con context.Context) (controllerapi.Controller_ReceiveSpoofedProbesClient, error) {
	return nil, nil
}

func TestStringSetUnion(t *testing.T) {
	one := []string{"non_spoofed"}
	two := []string{"173.205.3.15", "213.244.128.172"}
	res := stringSet(one).union(stringSet(two))
	if len(res) != 0 {
		t.Fatal("Union should be empty ", res)
	}
}
