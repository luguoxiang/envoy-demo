package kubernetes

import (
	"encoding/json"
	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"reflect"
	"time"
)

type PodEventHandler interface {
	PodValid(pod *PodInfo) bool
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

func (manager *K8sResourceManager) GetPodAnnotation(key string, podInfo *PodInfo) (string, error) {
	rawPod, err := manager.clientSet.CoreV1().Pods(podInfo.Namespace).Get(podInfo.Name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	if rawPod.Annotations != nil {
		return rawPod.Annotations[key], nil
	}
	return "", nil
}

func (manager *K8sResourceManager) PodExists(name string, ns string) (bool, error) {
	_, err := manager.clientSet.CoreV1().Pods(ns).Get(name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (manager *K8sResourceManager) SetPodAnnotation(annotations map[string]string, podInfo *PodInfo) error {
	payload := map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": annotations,
		},
	}

	payloadBytes, _ := json.Marshal(payload)
	_, err := manager.clientSet.CoreV1().Pods(podInfo.Namespace).Patch(podInfo.Name, types.MergePatchType, payloadBytes)
	return err
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
					if h.PodValid(pod) {
						h.PodAdded(pod)
					}
				}
			},
			DeleteFunc: func(obj interface{}) {
				pod := NewPodInfo(obj.(*v1.Pod))
				for _, h := range handlers {
					if h.PodValid(pod) {
						h.PodDeleted(pod)
					}
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				oldPod := NewPodInfo(oldObj.(*v1.Pod))
				newPod := NewPodInfo(newObj.(*v1.Pod))

				if oldPod != nil && newPod != nil {
					newVersion := newPod.ResourceVersion
					//ignore ResourceVersion diff
					newPod.ResourceVersion = oldPod.ResourceVersion
					if reflect.DeepEqual(oldPod, newPod) {
						return
					}
					newPod.ResourceVersion = newVersion
				}
				for _, h := range handlers {
					oldValid := (h.PodValid(oldPod))
					newValid := (h.PodValid(newPod))
					if !oldValid && newValid {
						h.PodAdded(newPod)
					} else if oldValid && !newValid {
						h.PodDeleted(oldPod)
					} else if oldValid && newValid {
						h.PodUpdated(oldPod, newPod)
					}
				}
			},
		},
	)
	controller.Run(stopper)
}
