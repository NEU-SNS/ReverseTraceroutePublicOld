package server

import (
	"sort"
	"testing"

	"github.com/NEU-SNS/ReverseTraceroute/atlas/pb"
)

func uint32SliceEqual(l, r []uint32) bool {
	if len(l) != len(r) {
		return false
	}
	sort.Sort(UInt32Slice(l))
	sort.Sort(UInt32Slice(r))
	for i, li := range l {
		if li != r[i] {
			return false
		}
	}
	return true
}

func TestRunningTrace_TryAdd(t *testing.T) {
	var tests = []struct {
		desc  string
		dst   uint32
		setup [][]uint32
		test  []uint32
		res   []uint32
	}{
		{
			desc:  "Add Overlapping Addresses",
			dst:   0,
			setup: [][]uint32{[]uint32{1, 2, 3}},
			test:  []uint32{2, 4, 5},
			res:   []uint32{4, 5},
		},
		{
			desc:  "Add Overlapping Addresses out of order",
			dst:   0,
			setup: [][]uint32{[]uint32{1, 5, 7, 99, 3, 18}}, test: []uint32{2, 5, 7, 18, 4, 15, 105},
			res: []uint32{2, 4, 15, 105},
		},
	}
	for _, test := range tests {
		rt := newRunningTraces()
		for _, ts := range test.setup {
			rt.TryAdd(test.dst, ts)
		}
		res := rt.TryAdd(test.dst, test.test)
		if !uint32SliceEqual(res, test.res) {
			t.Fatalf("%s: got: %v, expected: %v", test.desc, res, test.res)
		}
	}
}

func TestRunningTrace_Remove(t *testing.T) {
	var tests = []struct {
		desc  string
		dst   uint32
		setup [][]uint32
		test  []uint32
		res   []uint32
	}{
		{
			desc:  "Remove addresses",
			dst:   0,
			setup: [][]uint32{[]uint32{1, 2, 3}},
			test:  []uint32{1, 2, 3},
			res:   nil,
		},
		{
			desc:  "Remove partial addresses",
			dst:   600,
			setup: [][]uint32{[]uint32{1, 2, 3}},
			test:  []uint32{1, 3},
			res:   []uint32{2},
		},
		{
			desc:  "Remove not present addresses",
			dst:   999,
			setup: [][]uint32{[]uint32{1, 2, 3}},
			test:  []uint32{4, 5},
			res:   []uint32{1, 2, 3},
		},
	}
	for _, test := range tests {
		rt := newRunningTraces()
		for _, ts := range test.setup {
			added := rt.TryAdd(test.dst, ts)
			if !uint32SliceEqual(added, ts) {
				t.Fatalf("%s: failed to add sources got: %v, expected: %v", test.desc, added, test.res)
			}
		}
		rt.Remove(test.dst, test.test)
		check, _ := rt.Check(test.dst)
		if !uint32SliceEqual(check, test.res) {
			t.Fatalf("%s: got: %v, expected: %v", test.desc, check, test.res)
		}
	}
}

func TestTokenCache_Add(t *testing.T) {
	var tests = []struct {
		desc string
		add  *pb.IntersectionRequest
	}{
		{
			desc: "Add IR",
			add: &pb.IntersectionRequest{
				Address:      1,
				Dest:         2,
				Staleness:    30,
				UseAliases:   true,
				IgnoreSource: true,
				Src:          566,
			},
		},
	}
	for _, test := range tests {
		tc := newTokenCache()
		id := tc.Add(test.add)
		ir := tc.Get(id)
		if *ir != *test.add {
			t.Fatalf("%s: got: %v, expected: %v", test.desc, ir, test.add)
		}
	}
}

func TestTokenCache_Get(t *testing.T) {
	var tests = []struct {
		desc   string
		add    *pb.IntersectionRequest
		expect *pb.IntersectionRequest
	}{
		{
			desc: "Get IR",
			add: &pb.IntersectionRequest{
				Address:      1,
				Dest:         2,
				Staleness:    30,
				UseAliases:   true,
				IgnoreSource: true,
				Src:          566,
			},
			expect: &pb.IntersectionRequest{
				Address:      1,
				Dest:         2,
				Staleness:    30,
				UseAliases:   true,
				IgnoreSource: true,
				Src:          566,
			},
		},
		{
			desc:   "Get IR Nil",
			add:    nil,
			expect: nil,
		},
	}
	for _, test := range tests {
		tc := newTokenCache()
		id := tc.Add(test.add)
		ir := tc.Get(id)
		if ir == nil {
			if ir != test.expect {
				t.Fatalf("%s: got: %v, expected: %v", test.desc, ir, test.expect)
			}
			continue
		}
		if *ir != *test.expect {
			t.Fatalf("%s: got: %v, expected: %v", test.desc, ir, test.expect)
		}
	}
}

func TestTokenCache_Remove(t *testing.T) {
	var tests = []struct {
		desc   string
		add    []*pb.IntersectionRequest
		remove []uint32
		expect []error
	}{
		{
			desc: "Remove IR",
			add: []*pb.IntersectionRequest{&pb.IntersectionRequest{
				Address:      1,
				Dest:         2,
				Staleness:    30,
				UseAliases:   true,
				IgnoreSource: true,
				Src:          566,
			}},
			remove: []uint32{1},
			expect: []error{nil},
		},
		{
			desc:   "Remove Unadded",
			add:    nil,
			remove: []uint32{1},
			expect: []error{cacheError{id: 1}},
		},
	}
	for _, test := range tests {
		tc := newTokenCache()
		var ids []uint32
		for _, a := range test.add {
			id := tc.Add(a)
			ids = append(ids, id)
		}
		for i, rem := range test.remove {
			res := tc.Remove(rem)
			if res != test.expect[i] {
				t.Fatalf("%s: got: %v, expected: %v", test.desc, res, test.expect[i])
			}
		}
	}
}
