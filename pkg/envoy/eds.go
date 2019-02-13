package envoy

import (
	"context"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
)

type EndpointsDiscoveryService struct {
}

func NewEndpointsDiscoveryService() *EndpointsDiscoveryService {
	return &EndpointsDiscoveryService{}
}

func (cps *EndpointsDiscoveryService) StreamEndpoints(stream v2.EndpointDiscoveryService_StreamEndpointsServer) error {
	return nil
}

func (cps *EndpointsDiscoveryService) FetchEndpoints(ctx context.Context, req *v2.DiscoveryRequest) (*v2.DiscoveryResponse, error) {
	return nil, nil
}
