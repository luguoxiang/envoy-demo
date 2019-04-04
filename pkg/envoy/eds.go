package envoy

import (
	"context"
	"fmt"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/luguoxiang/envoy-demo/pkg/kubernetes"
	"sort"
	"strings"
)

type AssignmentInfo struct {
	PodIP   string
	Weight  uint32
	Version string
}

func (info *AssignmentInfo) String() string {
	return fmt.Sprintf("%s|%d", info.PodIP, info.Weight)
}

type EndpointInfo struct {
	App         string
	Port        uint32
	Assignments map[string]*AssignmentInfo
}

func (info *EndpointInfo) Name() string {
	cluster := OutboundClusterInfo{App: info.App, Port: info.Port}
	return cluster.Name()
}

func (info *EndpointInfo) String() string {
	var assignments []string
	for _, assignment := range info.Assignments {
		assignments = append(assignments, assignment.String())
	}
	return fmt.Sprintf("Endpoint|%s:%d|%s", info.App, info.Port, strings.Join(assignments, ","))
}

func (info *EndpointInfo) Version() string {
	var result []string
	for _, assignment := range info.Assignments {
		result = append(result, assignment.Version)
	}
	if len(result) == 0 {
		return "0"
	}
	sort.Strings(result)
	return strings.Join(result, "-")
}

type EndpointsDiscoveryService struct {
	DiscoveryService
}

func NewEndpointsDiscoveryService() *EndpointsDiscoveryService {
	return &EndpointsDiscoveryService{
		DiscoveryService: NewDiscoveryService(),
	}
}
func (eds *EndpointsDiscoveryService) updateResource(pod *kubernetes.PodInfo, remove bool) {
	app := pod.App()

	port := kubernetes.DemoAppSet[app]
	if port == 0 || pod.PodIP == "" {
		return
	}

	info := &EndpointInfo{
		App:         app,
		Port:        port,
		Assignments: map[string]*AssignmentInfo{},
	}
	resource := eds.GetResource(info.Name())
	if resource != nil {
		old := resource.(*EndpointInfo)
		for k, v := range old.Assignments {
			info.Assignments[k] = v
		}
	} else if remove {
		return
	}

	if remove {
		delete(info.Assignments, pod.PodIP)
	} else {
		info.Assignments[pod.PodIP] = &AssignmentInfo{
			PodIP:   pod.PodIP,
			Weight:  pod.Weight(),
			Version: pod.ResourceVersion,
		}
	}
	eds.UpdateResource(info)

}

func (eds *EndpointsDiscoveryService) PodValid(pod *kubernetes.PodInfo) bool {
	return pod.PodIP != ""
}

func (eds *EndpointsDiscoveryService) PodAdded(pod *kubernetes.PodInfo) {
	eds.updateResource(pod, false)
}

func (eds *EndpointsDiscoveryService) PodDeleted(pod *kubernetes.PodInfo) {
	eds.updateResource(pod, true)
}

func (eds *EndpointsDiscoveryService) PodUpdated(oldPod, newPod *kubernetes.PodInfo) {
	eds.updateResource(newPod, false)
}

func (ds *EndpointsDiscoveryService) StreamEndpoints(stream v2.EndpointDiscoveryService_StreamEndpointsServer) error {
	return ds.ProcessStream(stream, ds.BuildResource)
}

func (ds *EndpointsDiscoveryService) FetchEndpoints(ctx context.Context, req *v2.DiscoveryRequest) (*v2.DiscoveryResponse, error) {
	return ds.FetchResource(req, ds.BuildResource)
}

func (ds *EndpointsDiscoveryService) BuildResource(resourceMap map[string]EnvoyResource, version string, node *core.Node) (*v2.DiscoveryResponse, error) {
	var claList []proto.Message
	for _, resource := range resourceMap {
		endpointInfo := resource.(*EndpointInfo)
		cla := &v2.ClusterLoadAssignment{
			ClusterName: endpointInfo.Name(),
			Endpoints: []endpoint.LocalityLbEndpoints{{
				LbEndpoints: []endpoint.LbEndpoint{},
			}},
		}

		var lbEndpoints []endpoint.LbEndpoint
		for _, assignment := range endpointInfo.Assignments {
			if assignment.Weight == 0 {
				continue
			}
			lbEndpoint := endpoint.LbEndpoint{
				HostIdentifier: &endpoint.LbEndpoint_Endpoint{
					Endpoint: &endpoint.Endpoint{
						Address: &core.Address{
							Address: &core.Address_SocketAddress{
								SocketAddress: &core.SocketAddress{
									Protocol: core.TCP,
									Address:  assignment.PodIP,
									PortSpecifier: &core.SocketAddress_PortValue{
										PortValue: endpointInfo.Port,
									},
								},
							},
						},
					},
				},
				LoadBalancingWeight: &types.UInt32Value{
					Value: assignment.Weight,
				},
			}

			lbEndpoints = append(lbEndpoints, lbEndpoint)
		}

		cla.Endpoints[0].LbEndpoints = lbEndpoints
		claList = append(claList, cla)
	}

	return MakeResource(claList, EndpointResource, version)
}
