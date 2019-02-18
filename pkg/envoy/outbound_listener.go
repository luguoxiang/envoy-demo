package envoy

import (
	"fmt"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	hcm "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"

	types "github.com/gogo/protobuf/types"
)

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
