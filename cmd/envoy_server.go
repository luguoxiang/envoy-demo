package main

import (
	"context"
	"flag"
	"fmt"
	//"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/golang/glog"
	"github.com/luguoxiang/envoy-demo/pkg/docker"
	"github.com/luguoxiang/envoy-demo/pkg/envoy"
	"github.com/luguoxiang/envoy-demo/pkg/kubernetes"
	"google.golang.org/grpc"
	"net"
)

const grpcMaxConcurrentStreams = 1000000
const grpcPort = "15010"

func main() {
	flag.Parse()

	ctx := context.Background()

	grpcServer := grpc.NewServer(
		grpc.MaxConcurrentStreams(grpcMaxConcurrentStreams))

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		glog.Fatalf("failed to listen %s:%s", grpcPort, err.Error())
		panic(err.Error())
	}

	k8sManager, err := kubernetes.NewK8sResourceManager()
	if err != nil {
		glog.Fatalf("failed to create  K8sResourceManager:%s", err.Error())
		panic(err.Error())
	}

	envoyManager, err := docker.NewEnvoyManager(k8sManager)
	if err != nil {
		glog.Fatalf("failed to create EnvoyManager:%s", err.Error())
		panic(err.Error())
	}

	cds := envoy.NewClustersDiscoveryService()
	eds := envoy.NewEndpointsDiscoveryService()
	lds := envoy.NewListenersDiscoveryService()
	rds := envoy.NewRoutesDiscoveryService(k8sManager)

	ads := envoy.NewAggregatedDiscoveryService(cds, eds, lds, rds)
	stopper := make(chan struct{})
	go k8sManager.WatchPods(stopper, cds, eds, lds, envoyManager)

	//v2.RegisterEndpointDiscoveryServiceServer(grpcServer, eds)
	//v2.RegisterClusterDiscoveryServiceServer(grpcServer, cds)
	//v2.RegisterListenerDiscoveryServiceServer(grpcServer, lds)
	//v2.RegisterRouteDiscoveryServiceServer(grpcServer, rds)
	discovery.RegisterAggregatedDiscoveryServiceServer(grpcServer, ads)
	glog.Infof("grpc server listening %s", grpcPort)

	go func() {
		if err = grpcServer.Serve(lis); err != nil {
			glog.Error(err)
		}
	}()
	<-ctx.Done()

	grpcServer.GracefulStop()
}
