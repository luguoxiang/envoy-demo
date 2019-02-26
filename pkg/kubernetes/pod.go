package kubernetes

import (
	"fmt"
	"k8s.io/api/core/v1"
	"strconv"
	"strings"
)

const (
	ENDPOINT_WEIGHT_ANNOTATION = "demo.envoy.weight"
	ENVOY_PROXY_ANNOTATION     = "demo.envoy.proxy"
	ENVOY_ENABLE_ANNOTATION    = "demo.envoy.enabled"
	DEFAULT_WEIGHT             = 100
)

type PodInfo struct {
	ResourceVersion string
	Name            string
	Namespace       string
	PodIP           string
	HostIP          string
	Annotations     map[string]string
	Labels          map[string]string
	HostNetwork     bool
	Containers      []string
	ContainerPorts  []uint32
}

func (pod *PodInfo) App() string {
	return pod.Labels["app"]
}

func GetLabelValueUInt32(value string) uint32 {
	if value == "" {
		return 0
	}
	i, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return uint32(i)
}

func (pod *PodInfo) Weight() uint32 {
	if pod.Annotations != nil {
		value := pod.Annotations[ENDPOINT_WEIGHT_ANNOTATION]
		if value != "" {
			result := GetLabelValueUInt32(value)
			if result > 128 {
				return 128
			} else {
				return result
			}
		}
	}
	return DEFAULT_WEIGHT
}

func (pod *PodInfo) EnvoyDockerId() string {
	if pod.Annotations != nil {
		return pod.Annotations[ENVOY_PROXY_ANNOTATION]
	}
	return ""
}

func (pod *PodInfo) EnvoyAnnotated() bool {
	if pod.Annotations != nil {
		return strings.EqualFold(pod.Annotations[ENVOY_ENABLE_ANNOTATION], "true")
	}
	return false
}

func (pod *PodInfo) String() string {
	return fmt.Sprintf("Pod %s@%s PodIP %s",
		pod.Name, pod.Namespace, pod.PodIP)
}

func NewPodInfo(pod *v1.Pod) *PodInfo {
	var containers []string
	for _, container := range pod.Status.ContainerStatuses {
		id := container.ContainerID
		if strings.HasPrefix(id, "docker://") {
			id = id[9:]
		}
		containers = append(containers, id)
	}

	result := &PodInfo{
		PodIP:           pod.Status.PodIP,
		HostIP:          pod.Status.HostIP,
		Namespace:       pod.Namespace,
		Name:            pod.Name,
		Annotations:     pod.Annotations,
		Labels:          pod.Labels,
		ResourceVersion: pod.ResourceVersion,
		HostNetwork:     pod.Spec.HostNetwork,
		Containers:      containers,
	}
	if result.Annotations == nil {
		result.Annotations = make(map[string]string)
	}
	if result.Labels == nil {
		result.Labels = make(map[string]string)
	}
	return result
}
