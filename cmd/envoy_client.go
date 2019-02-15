package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"

	"github.com/gogo/protobuf/proto"
	"github.com/luguoxiang/envoy-demo/pkg/client"
	"github.com/luguoxiang/envoy-demo/pkg/envoy"
	"google.golang.org/grpc"
)

type StreamClient interface {
	Send(*v2.DiscoveryRequest) error
	Recv() (*v2.DiscoveryResponse, error)
}

type StreamFunction func(cc *grpc.ClientConn) (StreamClient, error)

func main() {
	var serverAddr string
	var typeUrl string
	var nodeId string
	flag.StringVar(&serverAddr, "serverAddr", "localhost:15010", "grpc server address")
	flag.StringVar(&nodeId, "nodeId", "", "nodeId")
	urls := []string{
		envoy.ListenerResource,
		envoy.ClusterResource,
		envoy.RouteResource,
		envoy.EndpointResource,
	}
	flag.StringVar(&typeUrl, "typeUrl", envoy.ListenerResource, fmt.Sprintf("one of %v", urls))
	flag.Parse()
	fmt.Printf("connecting %s\n", serverAddr)
	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	fmt.Println("connected")
	ctx := context.Background()

	ads, err := discovery.NewAggregatedDiscoveryServiceClient(conn).StreamAggregatedResources(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Println("client")

	var request v2.DiscoveryRequest
	request.TypeUrl = typeUrl

	request.Node = &core.Node{Id: nodeId, Cluster: "httpbin"}
	err = ads.Send(&request)
	if err != nil {
		panic(err)
	}
	fmt.Println("receiving")
	response, err := ads.Recv()
	if err != nil {
		panic(err)
	}
	fmt.Println("received")
	for _, resource := range response.Resources {

		if typeUrl == envoy.EndpointResource {
			data := &v2.ClusterLoadAssignment{}
			proto.Unmarshal(resource.Value, data)
			fmt.Printf("------%s--------\n", data.ClusterName)
			client.DoPrint(data)
		} else if typeUrl == envoy.ClusterResource {
			data := &v2.Cluster{}
			proto.Unmarshal(resource.Value, data)
			fmt.Printf("------%s--------\n", data.Name)
			client.DoPrint(data)
		} else if typeUrl == envoy.RouteResource {
			data := &v2.RouteConfiguration{}
			proto.Unmarshal(resource.Value, data)
			fmt.Printf("------%s--------\n", data.Name)
			client.DoPrint(data)
		} else if typeUrl == envoy.ListenerResource {
			data := &v2.Listener{}
			proto.Unmarshal(resource.Value, data)
			fmt.Printf("------%s--------\n", data.Name)
			client.DoPrint(data)
		} else {
			panic("unknown type typeURL:" + typeUrl)
		}

	}

}
