package envoy

import (
	"context"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
)

type RoutesDiscoveryService struct {
}

func NewRoutesDiscoveryService() *RoutesDiscoveryService {
	return &RoutesDiscoveryService{}
}

func (cps *RoutesDiscoveryService) StreamRoutes(stream v2.RouteDiscoveryService_StreamRoutesServer) error {
	return nil
}

func (cps *RoutesDiscoveryService) FetchRoutes(ctx context.Context, req *v2.DiscoveryRequest) (*v2.DiscoveryResponse, error) {
	return nil, nil
}

func (cps *RoutesDiscoveryService) IncrementalRoutes(v2.RouteDiscoveryService_IncrementalRoutesServer) error {
	return nil
}
