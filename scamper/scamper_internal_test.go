package scamper

import (
	"testing"

	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/util"
)

func TestCmdSpoof(t *testing.T) {
	saddr, _ := util.Int32ToIPString(2164947137)
	var test = &datamodel.PingMeasurement{
		Src:   2170636814,
		SAddr: saddr,
		Dst:   2162100337,
		Spoof: true,
		Count: "1",
	}
	cmd, err := newCmd(test, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(cmd.marshal()))
}
