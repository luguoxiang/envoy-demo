package envoy

import (
	"context"
	"fmt"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	"github.com/gogo/protobuf/proto"
	"github.com/luguoxiang/envoy-demo/pkg/kubernetes"
)

type RouteInfo struct {
	port  uint32
	hosts []string
}

func (info *RouteInfo) Name() string {
	return fmt.Sprintf("%d", info.port)
}

func (info *RouteInfo) String() string {
	return fmt.Sprintf("Route|%d", info.port)
}

func (info *RouteInfo) Version() string {
	return fmt.Sprintf("%d", len(info.hosts))
}

type RoutesDiscoveryService struct {
	DiscoveryService
	k8sManager *kubernetes.K8sResourceManager
}

func NewRoutesDiscoveryService(k8sManager *kubernetes.K8sResourceManager) *RoutesDiscoveryService {
	result := &RoutesDiscoveryService{
		DiscoveryService: NewDiscoveryService(),
		k8sManager:       k8sManager,
	}
	portMap := make(map[uint32]*RouteInfo)
	for service, port := range kubernetes.DemoAppSet {
		routeInfo := portMap[port]
		if routeInfo == nil {
			routeInfo = &RouteInfo{
				port: port,
			}
			portMap[port] = routeInfo
		}
		routeInfo.hosts = append(routeInfo.hosts, service)
		result.UpdateResource(routeInfo)
	}
	return result
}

func (rds *RoutesDiscoveryService) StreamRoutes(stream v2.RouteDiscoveryService_StreamRoutesServer) error {
	return rds.ProcessStream(stream, rds.BuildResource)
}

func (rds *RoutesDiscoveryService) FetchRoutes(ctx context.Context, req *v2.DiscoveryRequest) (*v2.DiscoveryResponse, error) {
	return rds.FetchResource(req, rds.BuildResource)
}

func (rds *RoutesDiscoveryService) BuildResource(resourceMap map[string]EnvoyResource, version string, node *core.Node) (*v2.DiscoveryResponse, error) {

	var routes []proto.Message
	for port, resource := range resourceMap {
		routeInfo := resource.(*RouteInfo)

		var virtualHostList []route.VirtualHost
		for _, host := range routeInfo.hosts {
			var domains []string
			domains = append(domains, fmt.Sprintf("%s:%s", host, port))
			domains = append(domains, fmt.Sprintf("%s.%s:%s", host, kubernetes.APP_NAMESPACE, port))
			clusterIp, err := rds.k8sManager.GetServiceClusterIP(host, kubernetes.APP_NAMESPACE)
			if err == nil && clusterIp != "" {
				domains = append(domains, fmt.Sprintf("%s:%s", clusterIp, port))
			}
			clusterInfo := OutboundClusterInfo{App: host, Port: routeInfo.port}
			virtualHost := route.VirtualHost{
				Name:    fmt.Sprintf("%s_%s_vh", host, port),
				Domains: domains,
				Routes: []route.Route{{
					Match: route.RouteMatch{
						PathSpecifier: &route.RouteMatch_Prefix{
							Prefix: "/",
						},
					},
					Action: &route.Route_Route{
						Route: &route.RouteAction{
							ClusterSpecifier: &route.RouteAction_Cluster{
								Cluster: clusterInfo.Name(),
							},
						},
					},
				}},
			}
			virtualHostList = append(virtualHostList, virtualHost)
		}

		routes = append(routes, &v2.RouteConfiguration{
			Name:         port,
			VirtualHosts: virtualHostList,
		})
	}

	return MakeResource(routes, RouteResource, version)
}
