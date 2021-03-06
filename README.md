# Install 
```
kubectl apply -f deploy.yaml

#wait all pods ready

kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.0/samples/bookinfo/platform/kube/bookinfo.yaml
```

# Quick start
## Query bookinfo service
```
kubectl run demo-client --image tutum/curl curl productpage:9080/productpage --restart=OnFailure
```

## access admin UI
```
kubectl port-forward (pod_name) 15000 &
access http://localhost:15000
```

## access zipkin UI
```
kubectl port-forward deployment/zipkin 9411 &
access http://localhost:9411
```

## Change endpoint weight
```
kubectl annotate pod (pod name) "demo.envoy.weight=weight" --overwrite
```
for example:
```
kubectl annotate pod reviews-v1-cb8655c75-fg8s4 "demo.envoy.weight=90" --overwrite
kubectl annotate pod reviews-v2-7fc9bb6dcf-8prsg "demo.envoy.weight=10" --overwrite
kubectl annotate pod reviews-v3-c995979bc-2sxqr "demo.envoy.weight=0" --overwrite
```

## Check envoy-demo configuration
```
cd $GOPATH/rc/github.com/luguoxiang/
git clone https://github.com/luguoxiang/envoy-demo.git
make build
kubectl port-forward deployment/envoy-demo 15010 &
./envoy-client -nodeId (pod_name) -typeUrl (typeUrl) -resource (resource)
```
typeUrl can be
* type.googleapis.com/envoy.api.v2.ClusterLoadAssignment
* type.googleapis.com/envoy.api.v2.Cluster
* type.googleapis.com/envoy.api.v2.RouteConfiguration
* type.googleapis.com/envoy.api.v2.Listener (default)

for example:
```
./envoy-client -nodeId productpage-v1-54d799c966-hhw5d
./envoy-client -nodeId productpage-v1-54d799c966-hhw5d -typeUrl type.googleapis.com/envoy.api.v2.Cluster
./envoy-client -nodeId productpage-v1-54d799c966-hhw5d -typeUrl type.googleapis.com/envoy.api.v2.ClusterLoadAssignment -resource "outbound|reviews:9080" 
./envoy-client -nodeId productpage-v1-54d799c966-hhw5d -typeUrl type.googleapis.com/envoy.api.v2.RouteConfiguration -resource "9080"
```

## Check istio pilot configuration
```
Install istio
kubectl port-forward deployment/istio-pilot -n istio-system 15010 &
./envoy-client -nodeId (node_id) -typeUrl (typeUrl) -resource (resource)
```
Node id can be found from following command result:
```
kubectl exec (pod name) -c istio-proxy cat /etc/istio/proxy/envoy-rev0.json
```
