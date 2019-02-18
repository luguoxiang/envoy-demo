package envoy

import (
	"fmt"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	hcm "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	types "github.com/gogo/protobuf/types"
)

type InboundListenerInfo struct {
	PodIP   string
	Port    uint32
	PodName string
}

func (info *InboundListenerInfo) Name() string {
	cluster := InboundClusterInfo{PodIP: info.PodIP, Port: info.Port}
	return cluster.Name()
}

func (info *InboundListenerInfo) String() string {
	return fmt.Sprintf("InboundListener|%s:%d", info.PodName, info.Port)
}

func (info *InboundListenerInfo) Version() string {
	return "1"
}

func (info *InboundListenerInfo) CreateListener() *v2.Listener {
	manager := &hcm.HttpConnectionManager{
		CodecType:  hcm.AUTO,
		StatPrefix: info.Name(),
		RouteSpecifier: &hcm.HttpConnectionManager_RouteConfig{
			RouteConfig: &v2.RouteConfiguration{
				Name: info.Name(),
				VirtualHosts: []route.VirtualHost{{
					Name:    fmt.Sprintf("%s_vh", info.Name()),
					Domains: []string{"*"},
					Routes: []route.Route{{
						Match: route.RouteMatch{
							PathSpecifier: &route.RouteMatch_Prefix{
								Prefix: "/",
							},
						},
						Action: &route.Route_Route{
							Route: &route.RouteAction{
								ClusterSpecifier: &route.RouteAction_Cluster{
									Cluster: info.Name(),
								},
							},
						},
					},
					},
				},
				},
			},
		},
		Tracing: &hcm.HttpConnectionManager_Tracing{
			OperationName: hcm.INGRESS,
		},
		HttpFilters: []*hcm.HttpFilter{{
			Name: RouterHttpFilter,
		}},
	}

	filterConfig, err := MessageToStruct(manager)
	if err != nil {
		panic(err.Error())
	}
	filterChain := listener.FilterChain{
		Filters: []listener.Filter{{
			Name:       HTTPConnectionManager,
			ConfigType: &listener.Filter_Config{Config: filterConfig},
		}},
	}

	return &v2.Listener{
		Name: info.Name(),
		Address: core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Protocol: core.TCP,
					Address:  info.PodIP,
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: info.Port,
					},
				},
			},
		},
		DeprecatedV1: &v2.Listener_DeprecatedV1{
			BindToPort: &types.BoolValue{Value: false},
		},

		FilterChains: []listener.FilterChain{filterChain},
	}
}
