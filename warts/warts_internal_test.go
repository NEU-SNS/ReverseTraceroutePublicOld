package warts

import "testing"

func TestTimeSinceMidnight(t *testing.T) {
	var testtime uint32 = 74413867
	t.Log(timeSinceMidnight(testtime))
}
