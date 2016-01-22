package router

import (
	"testing"
	"time"
)

func TestGetMT(t *testing.T) {
	r := New()
	s := ServiceDef{
		Addr:    "plcontroller.revtr.ccs.neu.edu",
		Port:    "4380",
		Service: planetLab,
	}
	for i := 0; i < 5; i++ {
		mt, err := r.GetMT(s)
		if err != nil {
			t.Fatal(err)
		}
		<-time.After(time.Millisecond * 1)
		err = mt.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestGetMT2(t *testing.T) {
	r := New()
	s := ServiceDef{
		Addr:    "plcontroller.revtr.ccs.neu.edu",
		Port:    "4380",
		Service: planetLab,
	}
	var mts []MeasurementTool
	for i := 0; i < 5; i++ {
		mt, err := r.GetMT(s)
		if err != nil {
			t.Fatal(err)
		}
		mts = append(mts, mt)
	}
	<-time.After(time.Second * 5)
	for _, mt := range mts {
		err := mt.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
}
