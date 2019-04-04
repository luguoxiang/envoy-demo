package envoy

import (
	"fmt"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/golang/glog"
	"strings"
)

type AggregatedDiscoveryService struct {
	cds *ClustersDiscoveryService
	eds *EndpointsDiscoveryService
	lds *ListenersDiscoveryService
	rds *RoutesDiscoveryService
}

func NewAggregatedDiscoveryService(cds *ClustersDiscoveryService,
	eds *EndpointsDiscoveryService,
	lds *ListenersDiscoveryService,
	rds *RoutesDiscoveryService) *AggregatedDiscoveryService {
	return &AggregatedDiscoveryService{
		cds: cds, eds: eds, lds: lds, rds: rds,
	}
}

func (ads *AggregatedDiscoveryService) StreamAggregatedResources(stream discovery.AggregatedDiscoveryService_StreamAggregatedResourcesServer) error {
	requestCh := make(chan *v2.DiscoveryRequest)
	go func() {
		for {
			req, err := stream.Recv()
			if err != nil {
				glog.Error(err.Error())
				requestCh <- nil
				return
			}
			if req.Node == nil || req.Node.Id == "" {
				err := fmt.Errorf("Missing node id info, type=%s, resource=%s", req.TypeUrl, strings.Join(req.ResourceNames, ","))
				glog.Error(err.Error())
				continue
			}

			glog.Infof("Request recevied: type=%s, nonce=%s, version=%s, resource=%s, node=%s",
				req.TypeUrl, req.GetResponseNonce(), req.VersionInfo, strings.Join(req.ResourceNames, ","), req.Node.Id)
			requestCh <- req
		}
	}()
	for {
		req := <-requestCh
		if req == nil {
			break
		}
		go func() {
			var resp *v2.DiscoveryResponse
			var err error
			switch req.TypeUrl {
			case EndpointResource:
				resp, err = ads.eds.ProcessRequest(req, ads.eds.BuildResource)
			case ClusterResource:
				resp, err = ads.cds.ProcessRequest(req, ads.cds.BuildResource)
			case RouteResource:
				resp, err = ads.rds.ProcessRequest(req, ads.rds.BuildResource)
			case ListenerResource:
				resp, err = ads.lds.ProcessRequest(req, ads.lds.BuildResource)
			default:
				panic("Unsupported TypeUrl" + req.TypeUrl)
			}

			if err != nil {
				glog.Errorf(err.Error())
				return
			}
			glog.Infof("Send %s, version=%s", req.TypeUrl, resp.VersionInfo)
			stream.Send(resp)
		}()
	}
	return nil
}

func (ads *AggregatedDiscoveryService) DeltaAggregatedResources(discovery.AggregatedDiscoveryService_DeltaAggregatedResourcesServer) error {
	return nil
}
