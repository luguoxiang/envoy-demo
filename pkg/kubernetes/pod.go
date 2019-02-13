package kubernetes

import (
	"fmt"
	"k8s.io/api/core/v1"
	"strconv"
)

const (
	ENDPOINT_WEIGHT_ANNOTATION = "demo.envoy.weight"
	DEFAULT_WEIGHT             = 100
)

type PodInfo struct {
	ResourceVersion string
	Name            string
	Namespace       string
	PodIP           string
	Annotations     map[string]string
	Labels          map[string]string
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

func (pod *PodInfo) String() string {
	return fmt.Sprintf("Pod %s@%s",
		pod.Name, pod.Namespace)
}

func NewPodInfo(pod *v1.Pod) *PodInfo {
	result := &PodInfo{
		PodIP:           pod.Status.PodIP,
		Namespace:       pod.Namespace,
		Name:            pod.Name,
		Annotations:     pod.Annotations,
		Labels:          pod.Labels,
		ResourceVersion: pod.ResourceVersion,
	}
	if result.Annotations == nil {
		result.Annotations = make(map[string]string)
	}
	if result.Labels == nil {
		result.Labels = make(map[string]string)
	}
	return result
}
