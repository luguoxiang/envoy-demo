package envoy

const (
	typePrefix       = "type.googleapis.com/envoy.api.v2."
	EndpointResource = typePrefix + "ClusterLoadAssignment"
	ClusterResource  = typePrefix + "Cluster"
	RouteResource    = typePrefix + "RouteConfiguration"
	ListenerResource = typePrefix + "Listener"
	XdsCluster       = "xds_cluster"
	RouterHttpFilter = "envoy.router"
)
