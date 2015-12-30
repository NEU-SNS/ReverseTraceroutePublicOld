package revtr

import (
	"log"
	"testing"

	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
)

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

func (vps vpSourceMock) GetVPs() (*datamodel.VPReturn, error) {
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
	return &datamodel.VPReturn{
		Vps: vp,
	}, nil
}

var myIP = "129.10.113.189"

func TestInitialize(t *testing.T) {
	initialize(vpSourceMock{}, "./alias_lists.txt")
	if len(ipToCluster) == 0 || len(clusterToIps) == 0 {
		t.Errorf("Failed to initialize")
	}
}

func TestNewReverseTraceroute(t *testing.T) {
	revtr := NewReverseTraceroute(myIP, "8.8.8.8", nil)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	t.Log(revtr)
}

func TestSymmetricAssumptions(t *testing.T) {
	revtr := NewReverseTraceroute(myIP, "8.8.8.8", nil)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	if revtr.SymmetricAssumptions() != 0 {
		t.Errorf("SymmetricAssumptions, Expected: 0, Got: %d", revtr.SymmetricAssumptions())
	}
}

func TestDeadends(t *testing.T) {
	revtr := NewReverseTraceroute(myIP, "8.8.8.8", nil)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	deadends := revtr.Deadends()
	if len(deadends) != 0 {
		t.Fatalf("Deadends, Expected: 0, Got: %d", len(revtr.Deadends()))
	}
}

func TestRRVPSInitializedForHop(t *testing.T) {
	revtr := NewReverseTraceroute(myIP, "8.8.8.8", nil)
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
	revtr := NewReverseTraceroute(myIP, "8.8.8.8", nil)
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
	revtr := NewReverseTraceroute(myIP, "8.8.8.8", nil)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	segs := []Segment{NewTrtoSrcRevSegment([]string{myIP, "8.8.8.8"}, myIP, "8.8.8.8")}
	added := revtr.AddSegments(segs)
	if !added {
		t.Fatal("Failed to add Segments")
	}
}

func TestReaches(t *testing.T) {
	revtr := NewReverseTraceroute(myIP, "8.8.8.8", nil)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	segs := []Segment{NewTrtoSrcRevSegment([]string{myIP, "8.8.8.8"}, myIP, "8.8.8.8")}
	added := revtr.AddSegments(segs)
	if !added {
		t.Fatal("Failed to add Segments")
	}
	reaches := revtr.Reaches()
	if !reaches {
		t.Fatal("Failed to reach after adding reaching segment")
	}
	t.Log(revtr)
}

type adjacencySourceMock struct {
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

func (as adjacencySourceMock) GetAdjacenciesByIP1(ip uint32) ([]datamodel.Adjacency, error) {
	var ret []datamodel.Adjacency
	for _, a := range adjs {
		if a.IP1 == ip {
			ret = append(ret, a)
		}
	}
	return ret, nil
}

func (as adjacencySourceMock) GetAdjacenciesByIP2(ip uint32) ([]datamodel.Adjacency, error) {
	var ret []datamodel.Adjacency
	for _, a := range adjs {
		if a.IP2 == ip {
			ret = append(ret, a)
		}
	}
	return ret, nil
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

func (as adjacencySourceMock) GetAdjacencyToDestByAddrAndDest24(dest24, addr uint32) ([]datamodel.AdjacencyToDest, error) {
	var ret []datamodel.AdjacencyToDest
	for _, i := range adjstodst {
		if i.Dest24 == dest24 && i.Address == addr {
			ret = append(ret, i)
		}
	}
	return ret, nil
}

func TestInitializeRRVPs(t *testing.T) {
	initialize(vpSourceMock{}, "./alias_lists.txt")
	revtr := NewReverseTraceroute(myIP, "8.8.8.8", nil)
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	testIP := "8.8.8.8"
	revtr.InitializeRRVPs(testIP)
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
	initialize(vpSourceMock{}, "./alias_lists.txt")
	revtr := NewReverseTraceroute(myIP, "8.8.8.8", nil)
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

func TestChooseOneSpooferPerSite(t *testing.T) {
	initialize(vpSourceMock{}, "./alias_lists.txt")
	ps := chooseOneSpooferPerSite()
	if len(ps) == 0 {
		t.Fatalf("Failed to get one spoofer per site")
	}
	t.Log(ps)
}

func TestInitializeTSAdjacents(t *testing.T) {
	initialize(vpSourceMock{}, "./alias_lists.txt")
	revtr := NewReverseTraceroute(myIP, "8.8.8.8", adjacencySourceMock{})
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
	initialize(vpSourceMock{}, "./alias_lists.txt")
	revtr := NewReverseTraceroute(myIP, "8.8.8.8", adjacencySourceMock{})
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
	initialize(vpSourceMock{}, "./alias_lists.txt")
	revtr := NewReverseTraceroute(myIP, "8.8.8.8", adjacencySourceMock{})
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
	initialize(vpSourceMock{}, "./alias_lists.txt")
	revtr := NewReverseTraceroute(myIP, "8.8.8.8", adjacencySourceMock{})
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	length := len(revtr.CurrPath().Path)
	revtr.CurrPath().Pop()
	if length != len(revtr.CurrPath().Path)+1 {
		t.Fatal("Failed to pop an item")
	}
	t.Log(length)
}

func TestHops(t *testing.T) {
	initialize(vpSourceMock{}, "./alias_lists.txt")
	revtr := NewReverseTraceroute(myIP, "8.8.8.8", adjacencySourceMock{})
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	hops := revtr.Hops()
	if len(hops) == 0 {
		t.Fatal("Hops failed, expected 1, got 0")
	}
}

func TestFailed(t *testing.T) {
	initialize(vpSourceMock{}, "./alias_lists.txt")
	revtr := NewReverseTraceroute(myIP, "8.8.8.8", adjacencySourceMock{})
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
	initialize(vpSourceMock{}, "./alias_lists.txt")
	revtr := NewReverseTraceroute(myIP, "8.8.8.8", adjacencySourceMock{})
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
	initialize(vpSourceMock{}, "./alias_lists.txt")
	revtr := NewReverseTraceroute(myIP, "8.8.8.8", adjacencySourceMock{})
	if revtr == nil {
		t.Fatalf("Failed to create ReverseTraceroute")
	}
	startLen := len(revtr.Paths)
	seg := NewTrtoSrcRevSegment([]string{"8.8.8.8", "10.0.0.1", "10.0.0.2"}, "8.8.8.8", myIP)
	res := revtr.AddAndReplaceSegment(seg)
	if !res {
		t.Fatalf("Failed to add Seg")
	}
	endLen := len(revtr.Paths)
	if startLen >= endLen {
		t.Fatal("Failed to add path")
	}
	if len(revtr.CurrPath().Path) != 1 {
		t.Fatal("Failed replacement ", len(revtr.CurrPath().Path))
	}
}

func TestAddBackgroundTRSegment(t *testing.T) {
	initialize(vpSourceMock{}, "./alias_lists.txt")
	revtr := NewReverseTraceroute(myIP, "8.8.8.8", adjacencySourceMock{})
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
