package watcher_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/NEU-SNS/ReverseTraceroute/util"
	"github.com/NEU-SNS/ReverseTraceroute/watcher"
)

func TestEvents(t *testing.T) {
	defer util.LeakCheck(t)()
	dir, err := ioutil.TempDir("", "watcher_test")
	if err != nil {
		t.Fatalf("Failed to create tmpdir")
	}
	defer os.RemoveAll(dir)
	w, err := watcher.New(dir)
	if err != nil {
		t.Fatalf("watcher.New(%s), Expected[<nil>], Got[%v]", err)
	}
	defer w.Close()
	var added []string
	for _, test := range []struct {
		action watcher.EventType
		err    error
	}{
		{action: watcher.Create},
		{action: watcher.Create},
		{action: watcher.Remove},
		{action: watcher.Create},
		{action: watcher.Remove},
		{action: watcher.Remove},
	} {
		switch test.action {
		case watcher.Create:
			f, err := ioutil.TempFile(dir, "")
			if err != nil {
				t.Fatalf("Failed to create file: %v", err)
			}
			e, err := w.GetEvent()
			if err != test.err {
				t.Fatalf("w.GetEvent(), Expected[%v], Got[%v]", test.err, err)
			}
			if e.Type() != test.action {
				t.Fatalf("w.GetEvent(), Type: Expected[%v], Got[%v]", test.action, e.Type())
			}
			if e.Name() != f.Name() {
				t.Fatalf("w.GetEvent(), Name: Expected[%v], Got[%v]", f.Name(), e.Name())
			}
			added = append(added, f.Name())
			f.Close()
		case watcher.Remove:
			f := added[0]
			added = added[1:]
			os.Remove(f)
			e, err := w.GetEvent()
			if err != test.err {
				t.Fatalf("w.GetEvent(), Expected[%v], Got[%v]", test.err, err)
			}
			if e.Type() != test.action {
				t.Fatalf("w.GetEvent(), Type: Expected[%v], Got[%v]", test.action, e.Type())
			}
			if e.Name() != f {
				t.Fatalf("w.GetEvent(), Name: Expected[%v], Got[%v]", f, e.Name())
			}
		}
	}
}
