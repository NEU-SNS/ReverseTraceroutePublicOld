package reversetraceroute

import (
	"fmt"
	"net"
	"reflect"
	"strings"

	"github.com/NEU-SNS/ReverseTraceroute/log"
)

const (
	dstRevSegment                         = 1
	dstSymRevSegment                      = 2
	trToSrcRevSegment                     = 3
	rrRevSegment                          = 4
	spoofRRRevSegment                     = 5
	tsAdjRevSegment                       = 6
	spoofTSAdjRevSegment                  = 7
	spoofTSAdjRevSegmentTSZero            = 8
	spoofTSAdjRevSegmentTSZeroDoubleStamp = 9
)

type stringSet []string

func (ss stringSet) union(s stringSet) []string {
	var mm map[string]bool
	mm = make(map[string]bool)
	var ret []string
	for _, c := range ss {
		mm[ipToCluster.Get(c)] = false
	}
	for _, c := range s {
		if _, ok := mm[ipToCluster.Get(c)]; ok {
			mm[ipToCluster.Get(c)] = true
		}
	}
	var foundNonSpoofed *bool
	foundNonSpoofed = new(bool)
	for k, v := range mm {
		if k == "non_spoofed" && v {
			*foundNonSpoofed = true
			continue
		}
		if v {
			ret = append(ret, k)
		}
	}
	if *foundNonSpoofed {
		ret = append([]string{"non_spoofed"}, ret...)
	}
	return ret
}

// Segment is the interface for a segment
type Segment interface {
	Hops() []string
	LastHop() string
	Length(bool) int
	Reaches() bool
	SymmetricAssumptions() int
	Order(Segment) int
	RemoveHops([]string) error
	Clone() Segment
	RemoveAt(int)
	Type() int
}

// RevSegment is a...
type RevSegment struct {
	Segment  []string
	Src, Hop string
}

// Type ...
func (rv *RevSegment) Type() int {
	return 0
}

// RemoveAt removes a hop at a given index
func (rv *RevSegment) RemoveAt(idx int) {
	rv.Segment, rv.Segment[len(rv.Segment)-1] = append(rv.Segment[:idx], rv.Segment[idx+1:]...), ""
}

func (rv *RevSegment) clone() *RevSegment {
	ret := RevSegment{
		Src: rv.Src,
		Hop: rv.Hop,
	}
	ret.Segment = append(ret.Segment, rv.Segment...)
	return &ret
}

// Clone is for the interface
func (rv *RevSegment) Clone() Segment {
	return rv.clone()
}

func (rv *RevSegment) String() string {
	return fmt.Sprintf("RevSegment_%v_S%s_H%s", rv.Segment, rv.Src, rv.Hop)
}

// Hops ...
func (rv *RevSegment) Hops() []string {
	return rv.Segment
}

// SetHop ...
func (rv *RevSegment) SetHop(hop string) {
	rv.Hop = hop
}

func rIndex(ss []string, s string) int {
	index := -1
	for i, st := range ss {
		if ipToCluster.Get(s) == ipToCluster.Get(st) {
			index = i
		}
	}
	return index
}

// RemoveHops ...
func (rv *RevSegment) RemoveHops(toDel []string) error {
	var noZeros []string
	segAsSet := stringSet(rv.Segment)
	for _, ip := range toDel {
		if ip != "0.0.0.0" {
			noZeros = append(noZeros, ip)
		}
	}
	hop := new(string)
	common := segAsSet.union(stringSet(noZeros))
	if len(common) > 0 {
		mapIndex := -1
		for _, h := range common {
			tmp := rIndex(rv.Segment, h)
			if tmp > mapIndex {
				// This is in the original code but isn't used at any point
				*hop = h
				mapIndex = tmp
			}
		}
		if mapIndex == len(rv.Segment) {
			rv.Segment = nil
		} else {
			rv.Segment = rv.Segment[mapIndex+1 : len(rv.Segment)]
		}
	}
	common = stringSet(rv.Segment).union(stringSet(toDel))
	if len(common) > 0 {
		return fmt.Errorf("Still a loop, %v, %v, %v, %v", toDel, rv.Segment, common, *hop)
	}
	return nil
}

// RemoveLocalHops ...
func (rv *RevSegment) RemoveLocalHops() error {
	var ns []string
	for _, h := range rv.Segment {
		ip := net.ParseIP(h)
		if !isInPrivatePrefix(ip) {
			ns = append(ns, h)
		}
	}
	rv.Segment = ns
	if len(rv.Segment) == 0 {
		return nil
	}
	for rv.Segment[len(rv.Segment)-1] == "0.0.0.0" {
		rv.Segment = rv.Segment[:len(rv.Segment)-1]
	}
	return nil
}

// SymmetricAssumptions ...
func (rv *RevSegment) SymmetricAssumptions() int {
	return 0
}

// Length ...
func (rv *RevSegment) Length(excNullHops bool) int {
	if excNullHops {
		var length int
		for _, h := range rv.Segment {
			if h != "0.0.0.0" {
				length++
			}
		}
		return length
	}
	return len(rv.Segment)
}

// LastHop ...
func (rv *RevSegment) LastHop() string {
	if len(rv.Segment) == 0 {
		return ""
	}
	return rv.Segment[len(rv.Segment)-1]
}

// Reaches ...
func (rv *RevSegment) Reaches() bool {
	return rv.LastHop() == rv.Src || ipToCluster.Get(rv.LastHop()) == ipToCluster.Get(rv.Src)
}

func stringArrayEquals(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i, ll := range left {
		if ll != right[i] {
			return false
		}
	}
	return true
}

func orderStringArray(left, right []string) int {
	if len(left) > len(right) {
		return 1
	}
	if len(left) < len(right) {
		return -1
	}
	for i, ll := range left {
		cmp := strings.Compare(ll, right[i])
		if cmp != 0 {
			return cmp
		}
	}
	return 0
}

// Order ...
func (rv *RevSegment) Order(b Segment) int {
	if reflect.TypeOf(rv) == reflect.TypeOf(&DstSymRevSegment{}) &&
		reflect.TypeOf(b) != reflect.TypeOf(&DstSymRevSegment{}) {
		return -1
	} else if reflect.TypeOf(b) == reflect.TypeOf(&DstSymRevSegment{}) &&
		reflect.TypeOf(rv) != reflect.TypeOf(&DstSymRevSegment{}) {
		return 1
	} else if stringArrayEquals(rv.Segment, b.Hops()) {
		return 0
	} else if rv.LastHop() == b.LastHop() {
		if rv.Length(true) != b.Length(true) {
			if rv.Length(true) < b.Length(true) {
				return -1
			}
			if rv.Length(true) == b.Length(true) {
				return 0
			}
			return 1
		}
		return orderStringArray(rv.Segment, b.Hops())
	} else if rv.Reaches() {
		return 1
	} else if b.Reaches() {
		return -1
	} else if rv.Length(true) != b.Length(true) {
		ll := rv.Length(true)
		rl := b.Length(true)
		if ll < rl {
			return -1
		}
		if ll > rl {
			return 1
		}
		return 0
	}
	return strings.Compare(rv.LastHop(), b.LastHop())
}

// NewRevSegment creates a new RevSegment
func NewRevSegment(segment []string, src, hop string) *RevSegment {
	ret := RevSegment{
		Src:     src,
		Hop:     hop,
		Segment: segment,
	}
	for i, h := range ret.Segment {
		if strings.Index(h, "192.168.") == 0 {
			ret.Segment[i] = "0.0.0.0"
		}
	}
	return &ret
}

// DstRevSegment is when the reverse hop was verified by virtue of being the destination
type DstRevSegment struct {
	*RevSegment
}

// Type ...
func (d *DstRevSegment) Type() int {
	return dstRevSegment
}

// NewDstRevSegment creates a new DstRevSegment
func NewDstRevSegment(segment []string, src, hop string) *DstRevSegment {
	ret := DstRevSegment{
		RevSegment: NewRevSegment(segment, src, hop),
	}
	return &ret
}

func (d *DstRevSegment) String() string {
	return fmt.Sprintf("%s_Dst", d.RevSegment.String())
}

// Clone is for the Segment interface
func (d *DstRevSegment) Clone() Segment {
	return d.clone()
}

func (d *DstRevSegment) clone() *DstRevSegment {
	ret := &DstRevSegment{
		RevSegment: d.RevSegment.clone(),
	}
	return ret
}

// DstSymRevSegment when the reverse hop is
type DstSymRevSegment struct {
	*RevSegment
	tr      []string
	numHops int
}

// Type ...
func (d *DstSymRevSegment) Type() int {
	return dstSymRevSegment
}

func (d *DstSymRevSegment) clone() *DstSymRevSegment {
	ret := DstSymRevSegment{
		RevSegment: d.RevSegment.clone(),
		numHops:    d.numHops,
	}
	ret.tr = append(ret.tr, d.tr...)
	return &ret
}

// Clone is for the interface
func (d *DstSymRevSegment) Clone() Segment {
	return d.clone()
}

func (d *DstSymRevSegment) String() string {
	return fmt.Sprintf("%s_AssumeSym", d.RevSegment.String())
}

func inArray(arr []string, s string) bool {
	for _, ss := range arr {
		if ss == s {
			return true
		}
	}
	return false
}

// This mimics the functionality of the static method select_nonzero_hops
func ndsrsSelectNonzeroHops(tr []string, hops int, hopsToIgnore []string) []string {
	log.Debugf("Selecting %d non-zero hops from %v, ignoring %v", hops, tr, hopsToIgnore)
	var i, found int
	for found < hops && i < len(tr) {
		if tr[i] != "0.0.0.0" && (!inArray(hopsToIgnore, tr[i])) {
			found++
		} else if inArray(hopsToIgnore, tr[i]) {
			log.Debug("Skipping Deadend ", tr[i])
		}
		i++
	}
	if found == hops {
		log.Debug("Rev Seg is ", tr[:i])
		return tr[:i]
	}
	log.Debug("Only able to find ", found)
	lastValidHop := len(tr) - 1
	for tr[lastValidHop] == "0.0.0.0" {
		lastValidHop--
	}
	return tr[:lastValidHop+1]
}

func stringSliceReverse(ss []string) []string {
	ret := make([]string, len(ss))
	for i, s := range ss {
		ret[len(ss)-i-1] = s
	}
	return ret
}

//NewDstSymRevSegment creates a new NewDstSymRevSegment
// tr is an array of hops along the forward path, not including the source
// numhops is number of nonzero hops to assume
// even if we are passing in the array of hops to store as the segment
// still need to include numhops, since we don't know how many of those are being ignored
// hop to ignore does no persist
func NewDstSymRevSegment(src, hop string, tr []string, numhops int, hopsToIgnore []string) *DstSymRevSegment {
	log.Debugf("src: %s, hop: %s, tr: %v, numHops: %d, htoi: %v", src, hop, tr, numhops, hopsToIgnore)
	ntr := append([]string{src}, tr[:len(tr)-1]...)
	log.Debug("New TR: ", ntr)
	rev := stringSliceReverse(ntr)
	log.Debug("The reversed slice is: ", rev)
	segment := ndsrsSelectNonzeroHops(rev, numhops, hopsToIgnore)
	log.Debug("The segment is: ", segment)
	ret := DstSymRevSegment{
		tr:         tr,
		numHops:    numhops,
		RevSegment: NewRevSegment(segment, src, hop),
	}
	return &ret
}

// AddHop ...
// assume an addition hop is symmetric
// keeps going until finding one that isn't 0
// note: right now, this starts from scratch every time and counts hops,
// replacing the curring segments with a new one.
func (d *DstSymRevSegment) AddHop(hopsToIgnore []string) error {
	tr := append([]string{d.Src}, d.tr[:len(d.tr)-1]...)
	rev := stringSliceReverse(tr)
	d.numHops++
	d.Segment = ndsrsSelectNonzeroHops(rev, d.numHops, hopsToIgnore)
	return nil
}

// TRtoSrcRevSegment is a ....
type TRtoSrcRevSegment struct {
	*RevSegment
}

// Type ...
func (d *TRtoSrcRevSegment) Type() int {
	return trToSrcRevSegment
}

// NewTrtoSrcRevSegment creates a new TRtoSrcRevSegment
func NewTrtoSrcRevSegment(segment []string, src, hop string) *TRtoSrcRevSegment {
	ret := TRtoSrcRevSegment{
		RevSegment: NewRevSegment(segment, src, hop),
	}
	return &ret
}

func (d *TRtoSrcRevSegment) String() string {
	return fmt.Sprintf("%s_TRtoSrc", d.RevSegment.String())
}

// RRRevSegment when the reverse hop was found with a non-spoofed RR probe
type RRRevSegment struct {
	*RevSegment
}

// Type ...
func (d RRRevSegment) Type() int {
	return rrRevSegment
}

// NewRRRevSegment creates a new RRRevSegment
func NewRRRevSegment(segment []string, src, hop string) *RRRevSegment {
	ret := RRRevSegment{
		RevSegment: NewRevSegment(segment, src, hop),
	}
	return &ret
}

func (d *RRRevSegment) clone() *RRRevSegment {
	ret := RRRevSegment{
		RevSegment: d.RevSegment.clone(),
	}
	return &ret
}

// Clone is for the interface
func (d *RRRevSegment) Clone() Segment {
	return d.clone()
}

func (d *RRRevSegment) String() string {
	return fmt.Sprintf("%s_RR", d.RevSegment.String())
}

// SpoofRRRevSegment when the reverse hop was found with a spoofed RR probe
type SpoofRRRevSegment struct {
	*RRRevSegment
	SpoofSource string
}

// Type ...
func (d *SpoofRRRevSegment) Type() int {
	return spoofRRRevSegment
}

func (d *SpoofRRRevSegment) clone() *SpoofRRRevSegment {
	ret := SpoofRRRevSegment{
		RRRevSegment: d.RRRevSegment.clone(),
		SpoofSource:  d.SpoofSource,
	}
	return &ret
}

// Clone is for the interface
func (d *SpoofRRRevSegment) Clone() Segment {
	return d.clone()
}

func (d *SpoofRRRevSegment) String() string {
	return fmt.Sprintf("%s_SpfSrc", d.RRRevSegment.String())
}

// NewSpoofRRRevSegment creates a new SpoofRRRevSegment
func NewSpoofRRRevSegment(segment []string, src, hop, spfsrc string) *SpoofRRRevSegment {
	ret := SpoofRRRevSegment{
		RRRevSegment: NewRRRevSegment(segment, src, hop),
		SpoofSource:  spfsrc,
	}
	return &ret
}

// TSAdjRevSegment when the reverse hop identified as potentially adjacent to another hop,
// then verified using timestamp
type TSAdjRevSegment struct {
	*RevSegment
	LinuxBug bool
}

// Type ...
func (d *TSAdjRevSegment) Type() int {
	return tsAdjRevSegment
}

func (d *TSAdjRevSegment) clone() *TSAdjRevSegment {
	ret := TSAdjRevSegment{
		RevSegment: d.RevSegment.clone(),
		LinuxBug:   d.LinuxBug,
	}
	return &ret
}

// Clone is for the interface
func (d *TSAdjRevSegment) Clone() Segment {
	return d.clone()
}

func (d *TSAdjRevSegment) String() string {
	if d.LinuxBug {
		return fmt.Sprintf("%s_TSAdj_LinuxBug", d.RevSegment.String())
	}
	return fmt.Sprintf("%s_TSAdj", d.RevSegment.String())
}

// NewTSAdjRevSegment creates a new TSAdjRevSegment
func NewTSAdjRevSegment(segment []string, src, hop string, linuxbug bool) *TSAdjRevSegment {
	ret := TSAdjRevSegment{
		LinuxBug:   linuxbug,
		RevSegment: NewRevSegment(segment, src, hop),
	}
	return &ret
}

// SpoofTSAdjRevSegment is when the reverse hop was found with a spoofed TS probe
type SpoofTSAdjRevSegment struct {
	*TSAdjRevSegment
	SpoofSource string
}

// Type ...
func (d *SpoofTSAdjRevSegment) Type() int {
	return spoofTSAdjRevSegment
}

func (d *SpoofTSAdjRevSegment) clone() *SpoofTSAdjRevSegment {
	ret := SpoofTSAdjRevSegment{
		TSAdjRevSegment: d.TSAdjRevSegment.clone(),
		SpoofSource:     d.SpoofSource,
	}
	return &ret
}

// Clone is for the iterface
func (d *SpoofTSAdjRevSegment) Clone() Segment {
	return d.clone()
}

func (d *SpoofTSAdjRevSegment) String() string {
	return fmt.Sprintf("%s_SpfSrc_%s", d.TSAdjRevSegment.String(), d.SpoofSource)
}

// NewSpoofTSAdjRevSegment creates a new SpoofTSAdjRevSegment
func NewSpoofTSAdjRevSegment(segment []string, src, hop, spfsrc string, linuxbug bool) *SpoofTSAdjRevSegment {
	ret := SpoofTSAdjRevSegment{
		SpoofSource:     spfsrc,
		TSAdjRevSegment: NewTSAdjRevSegment(segment, src, hop, linuxbug),
	}
	return &ret
}

// SpoofTSAdjRevSegmentTSZero is when the reverse hop was found with a spoofed TS probe
// and the destination is not a timestamper
// in general this means we found it by first issuing a spoofed probe and
// finding it was on either fwd or rev
// then issuing a non-spoofed and findit it wasn't on that
type SpoofTSAdjRevSegmentTSZero struct {
	*SpoofTSAdjRevSegment
}

// Type ...
func (d *SpoofTSAdjRevSegmentTSZero) Type() int {
	return spoofTSAdjRevSegmentTSZero
}

func (d *SpoofTSAdjRevSegmentTSZero) clone() *SpoofTSAdjRevSegmentTSZero {
	ret := SpoofTSAdjRevSegmentTSZero{
		SpoofTSAdjRevSegment: d.SpoofTSAdjRevSegment.clone(),
	}
	return &ret
}

// Clone is for the interface
func (d *SpoofTSAdjRevSegmentTSZero) Clone() Segment {
	return d.clone()
}

func (d *SpoofTSAdjRevSegmentTSZero) String() string {
	return fmt.Sprintf("%s_Dst0", d.SpoofTSAdjRevSegment.String())
}

// NewSpoofTSAdjRevSegmentTSZero creates a new SpoofTSAdjRevSegmentTSZero
func NewSpoofTSAdjRevSegmentTSZero(segment []string, src, hop, spfsrc string, linuxbug bool) *SpoofTSAdjRevSegmentTSZero {
	ret := SpoofTSAdjRevSegmentTSZero{
		SpoofTSAdjRevSegment: NewSpoofTSAdjRevSegment(segment, src, hop, spfsrc, linuxbug),
	}
	return &ret
}

// SpoofTSAdjRevSegmentTSZeroDoubleStamp is when
// the reverse hop was found with a spoofed TS probe
// and the destination is not a timestamper
// and we assume one was fwd, one rev
type SpoofTSAdjRevSegmentTSZeroDoubleStamp struct {
	*SpoofTSAdjRevSegmentTSZero
}

// Type ...
func (d *SpoofTSAdjRevSegmentTSZeroDoubleStamp) Type() int {
	return spoofTSAdjRevSegmentTSZeroDoubleStamp
}

func (d *SpoofTSAdjRevSegmentTSZeroDoubleStamp) clone() *SpoofTSAdjRevSegmentTSZeroDoubleStamp {
	ret := SpoofTSAdjRevSegmentTSZeroDoubleStamp{
		SpoofTSAdjRevSegmentTSZero: d.SpoofTSAdjRevSegmentTSZero.clone(),
	}
	return &ret
}

// Clone is for the interface
func (d *SpoofTSAdjRevSegmentTSZeroDoubleStamp) Clone() Segment {
	return d.clone()
}

func (d *SpoofTSAdjRevSegmentTSZeroDoubleStamp) String() string {
	return fmt.Sprintf("%s_DblStamp", d.SpoofTSAdjRevSegmentTSZero.String())
}

// NewSpoofTSAdjRevSegmentTSZeroDoubleStamp creates a new SpoofTSAdjRevSegmentTSZeroDoubleStamp
func NewSpoofTSAdjRevSegmentTSZeroDoubleStamp(segment []string, src, hop, spfsrc string, linuxbug bool) *SpoofTSAdjRevSegmentTSZeroDoubleStamp {
	ret := SpoofTSAdjRevSegmentTSZeroDoubleStamp{
		SpoofTSAdjRevSegmentTSZero: NewSpoofTSAdjRevSegmentTSZero(segment, src, hop, spfsrc, linuxbug),
	}
	return &ret
}
