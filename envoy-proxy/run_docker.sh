docker run --name $3 --privileged  --network container:$1 -e SERVICE_CLUSTER=$2 -e NODE_ID=$3 -l envoy.demo=true -d luguoxiang/envoy_proxy
