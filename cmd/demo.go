package main

import (
	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/glog"
	"github.com/luguoxiang/envoy-demo/pkg/envoy"

	"context"
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"net"
)

const grpcMaxConcurrentStreams = 1000000
const grpcPort = "18000"

func main() {
	flag.Parse()

	ctx := context.Background()

	grpcServer := grpc.NewServer(
		grpc.MaxConcurrentStreams(grpcMaxConcurrentStreams))

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		errInfo := fmt.Sprintf("failed to listen %s", grpcPort)
		glog.Fatal(errInfo)
	}

	ccps := envoy.NewClustersDiscoveryService()
	ecps := envoy.NewEndpointsDiscoveryService()
	lcps := envoy.NewListenersDiscoveryService()
	rcps := envoy.NewRoutesDiscoveryService()

	v2.RegisterEndpointDiscoveryServiceServer(grpcServer, ecps)
	v2.RegisterClusterDiscoveryServiceServer(grpcServer, ccps)
	v2.RegisterListenerDiscoveryServiceServer(grpcServer, lcps)
	v2.RegisterRouteDiscoveryServiceServer(grpcServer, rcps)
	glog.Infof("grpc server listening %s", grpcPort)
	go func() {
		if err = grpcServer.Serve(lis); err != nil {
			glog.Error(err)
		}
	}()
	<-ctx.Done()

	grpcServer.GracefulStop()
}
