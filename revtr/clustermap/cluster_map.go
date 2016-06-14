package clustermap

import (
	"fmt"
	"sync"
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/revtr/types"
	"github.com/NEU-SNS/ReverseTraceroute/util"
)

// ClusterMap maps IP addresses to cluster ids
type ClusterMap struct {
	ipc map[string]cmItem
	mu  *sync.Mutex
	cs  types.ClusterSource
}

type cmItem struct {
	fetched time.Time
	val     string
}

func (cm ClusterMap) fetchCluster(s string) string {

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

// Get gets a cluster id for ip s or returns s if there is none
func (cm ClusterMap) Get(s string) string {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if cl, ok := cm.ipc[s]; ok {
		if time.Since(cl.fetched) < time.Hour*2 {
			return cl.val
		}
	}
	return cm.fetchCluster(s)
}

// New creates a new cluster map using ClusterSource cs to retreive
// cluster ids
func New(cs types.ClusterSource) ClusterMap {
	i := make(map[string]cmItem)
	return ClusterMap{
		ipc: i,
		mu:  &sync.Mutex{},
		cs:  cs,
	}
}
