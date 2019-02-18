package envoy

import (
	"context"
	"fmt"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	hcm "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	"github.com/gogo/protobuf/proto"
	types "github.com/gogo/protobuf/types"
	"github.com/luguoxiang/envoy-demo/pkg/kubernetes"
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

type OutboundListenerInfo struct {
	Port uint32
}

func (info *OutboundListenerInfo) Name() string {
	return fmt.Sprintf("OutboundListener|%d", info.Port)
}

func (info *OutboundListenerInfo) String() string {
	return info.Name()
}

func (info *OutboundListenerInfo) Version() string {
	return "1"
}

type ListenersDiscoveryService struct {
	DiscoveryService
}

func NewListenersDiscoveryService() *ListenersDiscoveryService {
	return &ListenersDiscoveryService{
		DiscoveryService: NewDiscoveryService(),
	}
}

func (lds *ListenersDiscoveryService) updateResource(pod *kubernetes.PodInfo, remove bool) {
	app := pod.Labels["app"]

	port := DemoAppSet[app]
	if port == 0 || pod.PodIP == "" {
		return
	}

	outboundInfo := &OutboundListenerInfo{Port: port}
	inboundInfo := &InboundListenerInfo{PodIP: pod.PodIP, Port: port, PodName: pod.Name}
	if remove {
		lds.RemoveResource(inboundInfo.Name())
		//do not remove outbound listener
	} else {
		lds.UpdateResource(inboundInfo)
		lds.UpdateResource(outboundInfo)
	}
}

func (lds *ListenersDiscoveryService) PodAdded(pod *kubernetes.PodInfo) {
	lds.updateResource(pod, false)
}
func (lds *ListenersDiscoveryService) PodDeleted(pod *kubernetes.PodInfo) {
	lds.updateResource(pod, true)
}
func (lds *ListenersDiscoveryService) PodUpdated(oldPod, newPod *kubernetes.PodInfo) {
	lds.updateResource(newPod, false)
}
func (lds *ListenersDiscoveryService) StreamListeners(stream v2.ListenerDiscoveryService_StreamListenersServer) error {
	return lds.ProcessStream(stream, lds.BuildResource)
}

func (lds *ListenersDiscoveryService) FetchListeners(ctx context.Context, req *v2.DiscoveryRequest) (*v2.DiscoveryResponse, error) {
	return lds.FetchResource(req, lds.BuildResource)
}

func (lds *ListenersDiscoveryService) CreateVirtualListener() *v2.Listener {
	manager := &hcm.HttpConnectionManager{
		CodecType:  hcm.AUTO,
		StatPrefix: "http",
		RouteSpecifier: &hcm.HttpConnectionManager_RouteConfig{
			RouteConfig: &v2.RouteConfiguration{
				Name: "blackhole",
				VirtualHosts: []route.VirtualHost{{
					Name:    "blackhole_vh",
					Domains: []string{"*"},
					Routes: []route.Route{{
						Match: route.RouteMatch{
							PathSpecifier: &route.RouteMatch_Prefix{
								Prefix: "/",
							},
						},
						Action: &route.Route_DirectResponse{
							DirectResponse: &route.DirectResponseAction{
								Status: 404,
							},
						},
					},
					},
				},
				},
			},
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
		Name: "virtual",
		Address: core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Protocol: core.TCP,
					Address:  "0.0.0.0",
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: ENVOY_PROXY_PORT,
					},
				},
			},
		},

		UseOriginalDst: &types.BoolValue{Value: true},

		FilterChains: []listener.FilterChain{filterChain},
	}
}

func (info *OutboundListenerInfo) CreateListener() *v2.Listener {
	manager := &hcm.HttpConnectionManager{
		CodecType:  hcm.AUTO,
		StatPrefix: info.Name(),
		RouteSpecifier: &hcm.HttpConnectionManager_Rds{
			Rds: &hcm.Rds{
				ConfigSource: core.ConfigSource{
					ConfigSourceSpecifier: &core.ConfigSource_Ads{
						Ads: &core.AggregatedConfigSource{},
					},
				},
				RouteConfigName: fmt.Sprintf("%d", info.Port),
			},
		},

		Tracing: &hcm.HttpConnectionManager_Tracing{
			OperationName: hcm.EGRESS,
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
					Address:  "0.0.0.0",
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

func (lds *ListenersDiscoveryService) BuildResource(resourceMap map[string]EnvoyResource, version string, node *core.Node) (*v2.DiscoveryResponse, error) {
	var listeners []proto.Message
	for _, resource := range resourceMap {
		switch listenerInfo := resource.(type) {
		case *InboundListenerInfo:
			if listenerInfo.PodName == node.Id {
				listeners = append(listeners, listenerInfo.CreateListener())
			}
		case *OutboundListenerInfo:
			listeners = append(listeners, listenerInfo.CreateListener())
		default:
			panic("Unknown listener info")
		}
	}

	listeners = append(listeners, lds.CreateVirtualListener())
	return MakeResource(listeners, ListenerResource, version)
}
