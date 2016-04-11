package revtr

import (
	"fmt"
	"sync"
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/util"
)

type clusterMap struct {
	ipc map[string]cmItem
	mu  *sync.Mutex
	cs  ClusterSource
}

type cmItem struct {
	fetched time.Time
	val     string
}

func (cm clusterMap) fetchCluster(s string) string {

	ipint, _ := util.IPStringToInt32(s)
	cluster, err := cm.cs.GetClusterIDByIP(ipint)
	if err != nil {
		cm.ipc[s] = cmItem{
			fetched: time.Now(),
			val:     s,
		}
		return s
	}
	clusters := fmt.Sprintf("%d", cluster)
	cm.ipc[s] = cmItem{
		fetched: time.Now(),
		val:     clusters,
	}
	return clusters
}

func (cm clusterMap) Get(s string) string {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if cl, ok := cm.ipc[s]; ok {
		if time.Since(cl.fetched) < time.Hour*2 {
			return cl.val
		}
	}
	return cm.fetchCluster(s)
}

func newClusterMap(cs ClusterSource) clusterMap {
	i := make(map[string]cmItem)
	return clusterMap{
		ipc: i,
		mu:  &sync.Mutex{},
		cs:  cs,
	}
}
