package atlas

import (
	"sort"
	"testing"
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
		for _, t := range test.setup {
			rt.TryAdd(test.dst, t)
		}
		res := rt.TryAdd(test.dst, test.test)
		if !uint32SliceEqual(res, test.res) {
			t.Errorf("%s: got: %v, expected: %v", test.desc, res, test.res)
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
	}{}
}
