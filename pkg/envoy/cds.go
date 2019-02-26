package envoy

import (
	"context"
	"fmt"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/gogo/protobuf/proto"
	"github.com/luguoxiang/envoy-demo/pkg/kubernetes"
	"time"
)

type InboundClusterInfo struct {
	PodIP string
	Port  uint32
}

func (info *InboundClusterInfo) Name() string {
	return fmt.Sprintf("inbound|%s:%d", info.PodIP, info.Port)
}

func (info *InboundClusterInfo) String() string {
	return fmt.Sprintf("InboundCluster|%s:%d", info.PodIP, info.Port)
}

func (info *InboundClusterInfo) Version() string {
	return "1"
}

type OutboundClusterInfo struct {
	App  string
	Port uint32
}

func (info *OutboundClusterInfo) Name() string {
	return fmt.Sprintf("outbound|%s:%d", info.App, info.Port)
}

func (info *OutboundClusterInfo) String() string {
	return fmt.Sprintf("OutboundCluster|%s:%d", info.App, info.Port)
}

func (info *OutboundClusterInfo) Version() string {
	return "1"
}

type ClustersDiscoveryService struct {
	DiscoveryService
}

func NewClustersDiscoveryService() *ClustersDiscoveryService {
	return &ClustersDiscoveryService{
		DiscoveryService: NewDiscoveryService(),
	}
}

func (cds *ClustersDiscoveryService) updateResource(pod *kubernetes.PodInfo, remove bool) {
	app := pod.App()

	port := DemoAppSet[app]
	if port == 0 || pod.PodIP == "" {
		return
	}
	outboundInfo := &OutboundClusterInfo{App: app, Port: port}
	inboundInfo := &InboundClusterInfo{PodIP: pod.PodIP, Port: port}
	if remove {
		cds.RemoveResource(inboundInfo.Name())
		//do not remove outbound cluster
	} else {
		cds.UpdateResource(inboundInfo)
		cds.UpdateResource(outboundInfo)
	}
}

func (cds *ClustersDiscoveryService) PodValid(pod *kubernetes.PodInfo) bool {
	return pod.PodIP != ""
}

func (cds *ClustersDiscoveryService) PodAdded(pod *kubernetes.PodInfo) {
	cds.updateResource(pod, false)
}
func (cds *ClustersDiscoveryService) PodDeleted(pod *kubernetes.PodInfo) {
	cds.updateResource(pod, true)
}
func (cds *ClustersDiscoveryService) PodUpdated(oldPod, newPod *kubernetes.PodInfo) {
	cds.updateResource(newPod, false)
}

func (cds *ClustersDiscoveryService) StreamClusters(stream v2.ClusterDiscoveryService_StreamClustersServer) error {
	return cds.ProcessStream(stream, cds.BuildResource)
}

func (cds *ClustersDiscoveryService) FetchClusters(ctx context.Context, req *v2.DiscoveryRequest) (*v2.DiscoveryResponse, error) {
	return cds.FetchResource(req, cds.BuildResource)
}

func (cds *ClustersDiscoveryService) IncrementalClusters(v2.ClusterDiscoveryService_IncrementalClustersServer) error {
	return fmt.Errorf("Not supported")
}

func (cds *ClustersDiscoveryService) BuildResource(resourceMap map[string]EnvoyResource, version string, node *core.Node) (*v2.DiscoveryResponse, error) {
	var clusters []proto.Message

	connectionTimeout := time.Duration(60*1000) * time.Millisecond

	for _, resource := range resourceMap {
		var serviceCluster *v2.Cluster
		switch clusterInfo := resource.(type) {
		case *InboundClusterInfo:
			serviceCluster = &v2.Cluster{
				Name:           clusterInfo.Name(),
				ConnectTimeout: connectionTimeout,
				Type:           v2.Cluster_STATIC,
				Hosts: []*core.Address{
					&core.Address{
						Address: &core.Address_SocketAddress{
							SocketAddress: &core.SocketAddress{
								Protocol: core.TCP,
								Address:  "127.0.0.1",
								PortSpecifier: &core.SocketAddress_PortValue{
									PortValue: uint32(clusterInfo.Port),
								},
							},
						},
					},
				},
			}
		case *OutboundClusterInfo:
			serviceCluster = &v2.Cluster{
				Name:           clusterInfo.Name(),
				ConnectTimeout: connectionTimeout,
				Type:           v2.Cluster_EDS,
				EdsClusterConfig: &v2.Cluster_EdsClusterConfig{
					EdsConfig: &core.ConfigSource{
						ConfigSourceSpecifier: &core.ConfigSource_Ads{
							Ads: &core.AggregatedConfigSource{},
						},
					},
				},
			}
		default:
			panic("wrong cluster info type")
		}
		clusters = append(clusters, serviceCluster)
	}

	return MakeResource(clusters, ClusterResource, version)
}
