package controller

import (
	"sync"

	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/prometheus/log"
)

type spoofMap struct {
	sync.Mutex
	sm map[uint32]chan *dm.Probe
}

func (sm *spoofMap) Add(notify chan *dm.Probe, ids ...uint32) {
	sm.Lock()
	defer sm.Unlock()
	for _, id := range ids {
		sm.sm[id] = notify
	}
}

func (sm *spoofMap) Notify(probe *dm.Probe) {
	sm.Lock()
	defer sm.Unlock()
	log.Debug("Notify: ", probe)
	if c, ok := sm.sm[probe.ProbeId]; ok {
		c <- probe
		delete(sm.sm, probe.ProbeId)
		return
	}
	log.Error("No channel found for probe: ", probe)
}
