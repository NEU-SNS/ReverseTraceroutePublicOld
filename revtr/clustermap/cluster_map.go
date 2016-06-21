package clustermap

import (
	"fmt"

	"github.com/NEU-SNS/ReverseTraceroute/cache"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/types"
	"github.com/NEU-SNS/ReverseTraceroute/util"
)

// ClusterMap maps IP addresses to cluster ids
type ClusterMap struct {
	cs types.ClusterSource
	ca cache.Cache
}

func (cm ClusterMap) fetchCluster(s string) string {

	ipint, _ := util.IPStringToInt32(s)
	cluster, err := cm.cs.GetClusterIDByIP(ipint)
	if err != nil {
		log.Error(err)
		err := cm.ca.SetWithExpire("CM_"+s, []byte(s), 120)
		if err != nil {
			log.Error(err)
		}
		return s
	}
	clusters := fmt.Sprintf("%d", cluster)
	err = cm.ca.SetWithExpire("CM_"+clusters, []byte(clusters), 120)
	if err != nil {
		log.Error(err)
	}
	return clusters
}

// Get gets a cluster id for ip s or returns s if there is none
func (cm ClusterMap) Get(s string) string {
	item, err := cm.ca.Get("CM_" + s)
	if err == cache.ErrorCacheMiss {
		return cm.fetchCluster(s)
	}
	if err != nil {
		log.Error(err)
		return s
	}
	return string(item.Value())
}

// New creates a new cluster map using ClusterSource cs to retreive
// cluster ids and caching them in ca
func New(cs types.ClusterSource, ca cache.Cache) ClusterMap {
	return ClusterMap{
		cs: cs,
		ca: ca,
	}
}
