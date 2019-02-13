package envoy

import (
	"context"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
)

type ClustersDiscoveryService struct {
}

func NewClustersDiscoveryService() *ClustersDiscoveryService {
	return &ClustersDiscoveryService{}
}

func (cps *ClustersDiscoveryService) StreamClusters(stream v2.ClusterDiscoveryService_StreamClustersServer) error {
	return nil
}

func (cps *ClustersDiscoveryService) FetchClusters(ctx context.Context, req *v2.DiscoveryRequest) (*v2.DiscoveryResponse, error) {
	return nil, nil
}

func (cps *ClustersDiscoveryService) IncrementalClusters(v2.ClusterDiscoveryService_IncrementalClustersServer) error {
	return nil
}
