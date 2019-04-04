package kubernetes

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"io/ioutil"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"net/http"
	"os"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()
)

type WebhookServer struct {
	server *http.Server
}

func NewWebhookServer() *WebhookServer {
	pair, err := tls.LoadX509KeyPair("/etc/webhook/certs/cert.crt", "/etc/webhook/certs/cert.key")
	if err != nil {
		panic(err.Error())
	}

	server := &WebhookServer{
		server: &http.Server{
			Addr:      ":443",
			TLSConfig: &tls.Config{Certificates: []tls.Certificate{pair}},
		},
	}

	return server
}

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func (server *WebhookServer) Mutate(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	req := ar.Request

	glog.Infof("AdmissionReview for Kind=%v, Namespace=%v Resource=%v patchOperation=%v",
		req.Kind, req.Namespace, req.Resource, req.Operation)

	var pod corev1.Pod

	err := json.Unmarshal(req.Object.Raw, &pod)
	if err != nil {
		glog.Errorf("Could not unmarshal raw object: %s", err.Error())
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}
	podInfo := NewPodInfo(&pod)
	port := DemoAppSet[podInfo.App()]
	if port == 0 {
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}
	var ports []string
	for _, container := range pod.Spec.Containers {
		for _, port := range container.Ports {
			ports = append(ports, fmt.Sprintf("%d", port.ContainerPort))
		}
	}
	envMap := map[string]string{
		"CONTROL_PLANE_PORT":    fmt.Sprintf("%d", CONTROL_PLANE_PORT),
		"CONTROL_PLANE_SERVICE": CONTROL_PLANE_SERVICE,
		"PROXY_MANAGE_PORT":     fmt.Sprintf("%d", MANAGE_PORT),
		"PROXY_PORT":            fmt.Sprintf("%d", ENVOY_LISTEN_PORT),
		"PROXY_UID":             fmt.Sprintf("%d", PROXY_UID),
		"ZIPKIN_SERVICE":        ZIPKIN_SERVICE,
		"ZIPKIN_PORT":           fmt.Sprintf("%d", ZIPKIN_PORT),
		"INBOUND_PORTS_INCLUDE": fmt.Sprintf("%d", APP_PORT),
		"SERVICE_CLUSTER":       podInfo.App(),
	}

	var container corev1.Container
	container.Name = "envoy-proxy"
	container.ImagePullPolicy = corev1.PullAlways
	container.Image = os.Getenv("ENVOY_IMAGE")
	container.Ports = append(container.Ports, corev1.ContainerPort{
		Name:          "grpc",
		ContainerPort: ENVOY_LISTEN_PORT,
	})
	container.Ports = append(container.Ports, corev1.ContainerPort{
		Name:          "manage",
		ContainerPort: MANAGE_PORT,
	})
	for k, v := range envMap {
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  k,
			Value: v,
		})
	}
	privileged := true
	container.SecurityContext = &corev1.SecurityContext{
		Privileged: &privileged,
	}
	container.Env = append(container.Env, corev1.EnvVar{
		Name: "NODE_ID",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.name",
			},
		},
	})

	containers := pod.Spec.Containers
	containers = append(containers, container)
	patch := []patchOperation{{
		Op:    "add",
		Path:  "/spec/containers",
		Value: containers,
	}}

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	glog.Infof("Patch Pod %v\n", string(patchBytes))
	return &v1beta1.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		PatchType: func() *v1beta1.PatchType {
			pt := v1beta1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}

func (server *WebhookServer) Process(resp http.ResponseWriter, req *http.Request) {
	var body []byte
	if req.Body != nil {
		if data, err := ioutil.ReadAll(req.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		http.Error(resp, "empty body", http.StatusBadRequest)
		return
	}

	// verify the content type is accurate
	contentType := req.Header.Get("Content-Type")
	if contentType != "application/json" {
		glog.Errorf("Content-Type=%s, expect application/json", contentType)
		http.Error(resp, "invalid Content-Type, expect `application/json`", http.StatusUnsupportedMediaType)
		return
	}

	var admissionResponse *v1beta1.AdmissionResponse
	ar := v1beta1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, &ar); err != nil {
		glog.Errorf("Can't decode body: %v", err)
		admissionResponse = &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	} else {
		admissionResponse = server.Mutate(&ar)
	}

	admissionReview := v1beta1.AdmissionReview{}
	if admissionResponse != nil {
		admissionReview.Response = admissionResponse
		if ar.Request != nil {
			admissionReview.Response.UID = ar.Request.UID
		}
	}

	result, err := json.Marshal(admissionReview)
	if err != nil {
		glog.Errorf("Can't encode response: %v", err)
		http.Error(resp, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
	}
	glog.Infof("Ready to write reponse ...")
	if _, err := resp.Write(result); err != nil {
		glog.Errorf("Can't write response: %v", err)
		http.Error(resp, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	}
}

func (server *WebhookServer) Run() {
	mux := http.NewServeMux()
	mux.HandleFunc("/mutate", server.Process)
	server.server.Handler = mux

	glog.Infof("Starting Webhook Server ...")
	if err := server.server.ListenAndServeTLS("", ""); err != nil {
		glog.Errorf("Failed to listen and serve webhook server: %s", err.Error())
	}
}
