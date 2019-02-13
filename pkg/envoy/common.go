package envoy

import (
	"bytes"
	"fmt"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/golang/glog"
	"reflect"
	"sort"
	"strings"
	"sync"
)

const (
	typePrefix            = "type.googleapis.com/envoy.api.v2."
	EndpointResource      = typePrefix + "ClusterLoadAssignment"
	ClusterResource       = typePrefix + "Cluster"
	RouteResource         = typePrefix + "RouteConfiguration"
	ListenerResource      = typePrefix + "Listener"
	XdsCluster            = "xds_cluster"
	RouterHttpFilter      = "envoy.router"
	HTTPConnectionManager = "envoy.http_connection_manager"
	ENVOY_PROXY_PORT      = 10000
)

var (
	DemoAppSet = map[string]uint32{
		"productpage": 9080,
		"reviews":     9080,
		"ratings":     9080,
		"details":     9080,
	}
)

type EnvoyResource interface {
	Name() string
	Version() string
	String() string
}

type DiscoveryService struct {
	resourceMap map[string]EnvoyResource
	mutex       *sync.RWMutex
	cond        *sync.Cond
}

func NewDiscoveryService() DiscoveryService {
	mutex := &sync.RWMutex{}

	return DiscoveryService{
		resourceMap: map[string]EnvoyResource{},
		mutex:       mutex,
		cond:        sync.NewCond(mutex),
	}
}

func (ds *DiscoveryService) GetResource(name string) EnvoyResource {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	return ds.resourceMap[name]
}

func (ds *DiscoveryService) RemoveResource(name string) {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	resource := ds.resourceMap[name]
	if resource != nil {
		delete(ds.resourceMap, name)
		glog.Infof("RemoveResource %s, version=%s", resource.String(), resource.Version())
	}
	ds.cond.Broadcast()
}

func (ds *DiscoveryService) UpdateResource(resource EnvoyResource) {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	name := resource.Name()

	current := ds.resourceMap[name]
	if current != nil {
		if reflect.DeepEqual(current, resource) {
			return
		}
	}
	glog.Infof("UpdateResource %s, version=%s", resource.String(), resource.Version())
	ds.resourceMap[name] = resource
	ds.cond.Broadcast()
}

type ResponseBuilder func(resourceMap map[string]EnvoyResource, version string, node *core.Node) (*v2.DiscoveryResponse, error)

type stream interface {
	Send(*v2.DiscoveryResponse) error
	Recv() (*v2.DiscoveryRequest, error)
}

func (ds *DiscoveryService) GetResources(resourceNames []string) (map[string]EnvoyResource, string) {
	requested := make(map[string]EnvoyResource)
	var versions []string
	if len(resourceNames) > 0 {
		sort.Strings(resourceNames)
		for _, name := range resourceNames {
			resource := ds.resourceMap[name]
			if resource == nil {
				glog.Warningf("Could not find requested %s", name)
				continue
			}
			requested[name] = resource
			versions = append(versions, resource.Version())
		}
	} else {
		for name, resource := range ds.resourceMap {
			requested[name] = resource
			versions = append(versions, resource.Version())
		}
	}
	sort.Strings(versions)
	return requested, strings.Join(versions, ",")
}

func (ds *DiscoveryService) FetchResource(req *v2.DiscoveryRequest, builder ResponseBuilder) (*v2.DiscoveryResponse, error) {
	glog.Infof("Fetch %s for %v", req.TypeUrl, req.ResourceNames)

	ds.mutex.Lock()
	resourceMap, currentVersion := ds.GetResources(req.ResourceNames)
	ds.mutex.Unlock()

	return builder(resourceMap, currentVersion, req.Node)
}

func (ds *DiscoveryService) ProcessStream(stream stream, builder ResponseBuilder) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			glog.Error(err.Error())
			return err
		}
		if req.Node == nil || req.Node.Id == "" {
			err := fmt.Errorf("Missing node id info, type=%s, resource=%s", req.TypeUrl, strings.Join(req.ResourceNames, ","))
			glog.Error(err.Error())
			return err
		}

		glog.Infof("Request recevied: type=%s, nonce=%s, version=%s, resource=%s, node=%s",
			req.TypeUrl, req.GetResponseNonce(), req.VersionInfo, strings.Join(req.ResourceNames, ","), req.Node.Id)

		ds.mutex.Lock()

		var currentVersion string
		var resourceMap map[string]EnvoyResource
		for {
			resourceMap, currentVersion = ds.GetResources(req.ResourceNames)

			if currentVersion == req.VersionInfo {
				glog.Infof("Waiting update on %s for %v, current version=%s", req.TypeUrl, req.ResourceNames, currentVersion)
				ds.cond.Wait()
			} else {
				break
			}
		}

		ds.mutex.Unlock()

		resp, err := builder(resourceMap, currentVersion, req.Node)
		if err != nil {
			glog.Errorf(err.Error())
			return err
		}
		glog.Infof("Send %s, version=%s", req.TypeUrl, resp.VersionInfo)
		stream.Send(resp)
	}
}

func MakeXdsCluster() *core.ConfigSource {
	grpcService := &core.GrpcService{
		TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
			EnvoyGrpc: &core.GrpcService_EnvoyGrpc{
				ClusterName: XdsCluster,
			},
		},
	}
	return &core.ConfigSource{
		ConfigSourceSpecifier: &core.ConfigSource_ApiConfigSource{
			ApiConfigSource: &core.ApiConfigSource{
				ApiType: core.ApiConfigSource_GRPC,
				GrpcServices: []*core.GrpcService{
					grpcService,
				},
			},
		},
	}
}

func MakeResource(resources []proto.Message, typeURL string, version string) (*v2.DiscoveryResponse, error) {
	var resoureList []types.Any
	for _, resource := range resources {
		data, err := proto.Marshal(resource)
		if err != nil {
			glog.Error(err.Error())
			return nil, err
		}

		resourceAny := types.Any{
			TypeUrl: typeURL,
			Value:   data,
		}
		resoureList = append(resoureList, resourceAny)
	}

	out := &v2.DiscoveryResponse{
		Nonce:       "0",
		VersionInfo: version,
		Resources:   resoureList,
		TypeUrl:     typeURL,
	}
	return out, nil
}

func MessageToStruct(msg proto.Message) (*types.Struct, error) {
	buf := &bytes.Buffer{}
	if err := (&jsonpb.Marshaler{OrigName: true}).Marshal(buf, msg); err != nil {
		return nil, err
	}

	pbs := &types.Struct{}
	if err := jsonpb.Unmarshal(buf, pbs); err != nil {
		return nil, err
	}

	return pbs, nil
}
