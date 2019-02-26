package docker

import (
	"github.com/golang/glog"
	"github.com/luguoxiang/envoy-demo/pkg/kubernetes"
	"os"
	"sync"
)

type EnvoyManager struct {
	dockerClient *DockerClient
	k8sManager   *kubernetes.K8sResourceManager
	envoyMutex   *sync.RWMutex
	myHostIp     string
}

func NewEnvoyManager(k8sManager *kubernetes.K8sResourceManager) (*EnvoyManager, error) {
	dockerClient, err := NewDockerClient(true)
	if err != nil {
		return nil, err
	}
	return &EnvoyManager{
		dockerClient: dockerClient,
		k8sManager:   k8sManager,
		envoyMutex:   &sync.RWMutex{},

		myHostIp: os.Getenv("MY_HOST_IP"),
	}, nil
}
func (manager *EnvoyManager) CheckExistingEnvoy() {
	instances, err := manager.dockerClient.ListDockerInstances()
	if err != nil {
		glog.Warningf("ListDockerInstances failed: %s", err.Error())
		return
	}
	for _, instance := range instances {
		var exists bool
		exists, err = manager.k8sManager.PodExists(instance.Pod, instance.Namespace)
		if err != nil {
			glog.Warningf("Get pod %s@%s failed: %s", instance.Pod, instance.Namespace, err.Error())
			continue
		}
		if !exists {
			glog.Warningf("Envoy docker %s's target pod %s@%s dead", instance.ID, instance.Pod, instance.Namespace)
			manager.dockerClient.StopDockerInstance(instance.ID, instance.Pod)
			manager.dockerClient.RemoveDockerInstance(instance.ID, instance.Pod)
		}
	}
}
func (manager *EnvoyManager) checkEnvoy(podInfo *kubernetes.PodInfo) {
	if podInfo.HostIP != manager.myHostIp {
		return
	}

	envoyEnabled := podInfo.EnvoyAnnotated()
	go func() {
		//make all check run in serial
		manager.envoyMutex.Lock()
		defer manager.envoyMutex.Unlock()

		dockerId, err := manager.k8sManager.GetPodAnnotation(kubernetes.ENVOY_PROXY_ANNOTATION, podInfo)
		if err != nil {
			glog.Errorf("Get pod %s failed: %s", podInfo.Name, err.Error())
			return
		}
		if envoyEnabled {
			if dockerId != "" {
				if manager.dockerClient.IsDockerInstanceRunning(dockerId) {
					return
				}
				manager.dockerClient.RemoveDockerInstance(dockerId, podInfo.Name)
			}
			dockerId, err = manager.dockerClient.CreateDockerInstance(podInfo, podInfo.App())
			if err != nil {
				glog.Errorf("Create docker instances for %s failed: %s", podInfo.Name, err.Error())
				return
			}
		} else if dockerId != "" {
			manager.dockerClient.StopDockerInstance(dockerId, podInfo.Name)
			manager.dockerClient.RemoveDockerInstance(dockerId, podInfo.Name)
			dockerId = ""
		} else {
			return
		}

		err = manager.k8sManager.SetPodAnnotation(
			map[string]string{
				kubernetes.ENVOY_PROXY_ANNOTATION: dockerId,
			}, podInfo)
		if err != nil {
			glog.Errorf("Failed to annotate pod %s:%s", podInfo.Name, err.Error())
			if dockerId != "" {
				manager.dockerClient.StopDockerInstance(dockerId, podInfo.Name)
			}
		}
	}()
}

func (manager *EnvoyManager) PodValid(pod *kubernetes.PodInfo) bool {
	//Hostnetwork pod should not have envoy enabled
	return !pod.HostNetwork
}

func (manager *EnvoyManager) PodAdded(pod *kubernetes.PodInfo) {
	manager.checkEnvoy(pod)
}
func (manager *EnvoyManager) PodDeleted(pod *kubernetes.PodInfo) {
	dockerId := pod.EnvoyDockerId()
	if dockerId != "" {
		go manager.dockerClient.StopDockerInstance(dockerId, pod.Name)
	}

}
func (manager *EnvoyManager) PodUpdated(oldPod, newPod *kubernetes.PodInfo) {
	manager.checkEnvoy(newPod)
}
