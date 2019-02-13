package envoy

import (
	"context"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
)

type ListenersDiscoveryService struct {
}

func NewListenersDiscoveryService() *ListenersDiscoveryService {
	return &ListenersDiscoveryService{}
}

func (cps *ListenersDiscoveryService) StreamListeners(stream v2.ListenerDiscoveryService_StreamListenersServer) error {
	return nil
}

func (cps *ListenersDiscoveryService) FetchListeners(ctx context.Context, req *v2.DiscoveryRequest) (*v2.DiscoveryResponse, error) {
	return nil, nil
}
