package kubernetes

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"reflect"
	"time"
)

type PodEventHandler interface {
	PodAdded(pod *PodInfo)
	PodDeleted(pod *PodInfo)
	PodUpdated(oldPod, newPod *PodInfo)
}

type K8sResourceManager struct {
	clientSet kubernetes.Interface
}

func NewK8sResourceManager() (*K8sResourceManager, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	result := &K8sResourceManager{
		clientSet: clientSet,
	}

	return result, nil
}
func (manager *K8sResourceManager) GetServiceClusterIP(name string, namespace string) (string, error) {
	service, err := manager.clientSet.CoreV1().Services(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return service.Spec.ClusterIP, nil
}
func (manager *K8sResourceManager) WatchPods(stopper chan struct{}, handlers ...PodEventHandler) {
	watchlist := cache.NewListWatchFromClient(
		manager.clientSet.Core().RESTClient(), "pods", "",
		fields.Everything())
	_, controller := cache.NewInformer(
		watchlist,
		&v1.Pod{},
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				pod := NewPodInfo(obj.(*v1.Pod))
				for _, h := range handlers {
					h.PodAdded(pod)
				}
			},
			DeleteFunc: func(obj interface{}) {
				pod := NewPodInfo(obj.(*v1.Pod))
				for _, h := range handlers {
					h.PodDeleted(pod)
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				oldPod := NewPodInfo(oldObj.(*v1.Pod))
				newPod := NewPodInfo(newObj.(*v1.Pod))
				if reflect.DeepEqual(oldPod, newPod) {
					return
				}

				for _, h := range handlers {
					h.PodUpdated(oldPod, newPod)
				}
			},
		},
	)
	controller.Run(stopper)
}
