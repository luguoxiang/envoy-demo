package docker

import (
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	dockerclient "github.com/docker/docker/client"
	"github.com/golang/glog"
	"github.com/luguoxiang/envoy-demo/pkg/kubernetes"
	"golang.org/x/net/context"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

const (
	DOCKER_LABEL_NAMESPACE = "demo.envoy.namespace"
	DOCKER_LABEL_POD       = "demo.envoy.pod"
	DOCKER_LABEL_PROXY     = "demo.envoy.proxy"
)

type DockerInstanceInfo struct {
	ID        string
	Namespace string
	Pod       string
	Status    string
	State     string
}

type DockerClient struct {
	client              *dockerclient.Client
	UserName            string
	Password            string
	ProxyPort           string
	ControlPlanePort    string
	ControlPlaneService string
	ProxyManagePort     string
	ProxyUID            string
	EnvoyImage          string
}

func NewDockerClient(pullImage bool) (*DockerClient, error) {
	dockerClient := &DockerClient{
		EnvoyImage: os.Getenv("ENVOY_IMAGE"),
	}
	if dockerClient.EnvoyImage == "" {
		panic("Missing env ENVOY_IMAGE")
	}
	var err error
	dockerClient.client, err = dockerclient.NewEnvClient()
	if err != nil {
		return nil, err
	}
	if pullImage {
		err = dockerClient.PullImage(context.Background(), dockerClient.EnvoyImage)
		if err != nil {
			return nil, err
		}
	}
	return dockerClient, nil
}

func (client *DockerClient) ListDockerInstances() ([]*DockerInstanceInfo, error) {
	var result []*DockerInstanceInfo
	args := filters.NewArgs()
	args, err := filters.ParseFlag("label="+DOCKER_LABEL_PROXY, args)
	if err != nil {
		panic(err.Error())
	}
	containers, err := client.client.ContainerList(context.Background(), types.ContainerListOptions{Filters: args})
	if err != nil {
		return nil, err
	}

	for _, container := range containers {
		labels := container.Labels
		ns := labels[DOCKER_LABEL_NAMESPACE]
		pod := labels[DOCKER_LABEL_POD]

		if ns != "" && pod != "" {
			dockerInfo := DockerInstanceInfo{
				ID: container.ID, Namespace: ns, Pod: pod,
				Status: container.Status,
				State:  container.State,
			}
			result = append(result, &dockerInfo)
		}
	}

	return result, nil
}

func (client *DockerClient) CreateDockerInstance(podInfo *kubernetes.PodInfo, serviceCluster string) (string, error) {
	ctx := context.Background()
	var pauseDocker string

	for _, container := range podInfo.Containers {
		containerJson, err := client.client.ContainerInspect(ctx, container)
		if err != nil {
			glog.Errorf("Failed to inspect docker %s for pod %s", container, podInfo.Name)
			continue
		}
		pauseDocker = string(containerJson.HostConfig.NetworkMode)
		if strings.HasPrefix(pauseDocker, "container:") {
			pauseDocker = pauseDocker[10:]
		} else {
			continue
		}
	}

	if pauseDocker == "" {
		return "", fmt.Errorf("could not find pause docker for %s", podInfo.Name)
	}
	if glog.V(2) {
		glog.Infof("target network container %s for pod %s", pauseDocker, podInfo.Name)
	}
	network := container.NetworkMode(fmt.Sprintf("container:%s", pauseDocker))
	env := []string{
		fmt.Sprintf("MY_POD_IP=%s", podInfo.PodIP),

		////used for envoy's --service-cluster option
		fmt.Sprintf("SERVICE_CLUSTER=%s", serviceCluster),
		fmt.Sprintf("NODE_ID=%s.%s", podInfo.Name, podInfo.Namespace),
	}

	proxy_config := &container.Config{
		Env: env,
		Labels: map[string]string{
			DOCKER_LABEL_PROXY:     "true",
			DOCKER_LABEL_NAMESPACE: podInfo.Namespace,
			DOCKER_LABEL_POD:       podInfo.Name,
		},
		Image:     client.EnvoyImage,
		Tty:       false,
		OpenStdin: false,
	}
	host_config := &container.HostConfig{
		NetworkMode: network,
		Privileged:  true,
	}

	resp, err := client.client.ContainerCreate(ctx, proxy_config, host_config, nil, fmt.Sprintf("envoy_%s_%s", podInfo.Name, podInfo.Namespace))
	if err != nil {
		return "", err
	}
	glog.Infof("Create proxy docker %s for pod %s, env=%v, network:%s", resp.ID, podInfo.Name, env, pauseDocker)
	err = client.client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})
	if err != nil {
		glog.Warningf("Removing proxy docker %s for start failure", resp.ID)
		removeErr := client.client.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{})
		if removeErr != nil {
			glog.Errorf("Remove container failed: %s", removeErr.Error())
		}
		return "", err
	}

	return resp.ID, nil
}
func (client *DockerClient) PullImage(ctx context.Context, imageName string) error {
	var option types.ImagePullOptions
	glog.Infof("Pulling Image %s", imageName)

	out, err := client.client.ImagePull(ctx, imageName, option)
	if err != nil {
		glog.Errorf("Pull image failed: %s", err.Error())
		return err
	}

	defer out.Close()
	body, err := ioutil.ReadAll(out)
	if err != nil {
		glog.Errorf("Read image pulling output failed: %s", err.Error())
	} else {
		lines := strings.Split(string(body), "\n")
		linesNum := len(lines)
		if linesNum > 3 {
			lines = lines[linesNum-3 : linesNum]
		}
		glog.Infof("Pulled Image %s", imageName)
		for _, line := range lines {
			glog.Infof("Status: %s", line)
		}
	}
	return nil
}

func (client *DockerClient) GetDockerInstanceLog(dockerId string) (io.ReadCloser, error) {
	ctx := context.Background()
	return client.client.ContainerLogs(ctx, dockerId, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
}
func (client *DockerClient) IsDockerInstanceRunning(dockerId string) bool {
	ctx := context.Background()
	containerJson, err := client.client.ContainerInspect(ctx, dockerId)
	if err != nil {
		glog.Errorf("Inspect container %s failed: %s", dockerId, err.Error())
		return false
	}
	if containerJson.State != nil {
		return containerJson.State.Running
	} else {
		return false
	}
}

func (client *DockerClient) StopDockerInstance(dockerId string, podName string) {
	ctx := context.Background()
	err := client.client.ContainerStop(ctx, dockerId, nil)
	if err != nil {
		glog.Errorf("Stop container %s failed: %s", dockerId, err.Error())
	} else {
		glog.Infof("Stopped container %s for %s", dockerId, podName)
	}
}

func (client *DockerClient) RemoveDockerInstance(dockerId string, podName string) {
	ctx := context.Background()
	err := client.client.ContainerRemove(ctx, dockerId, types.ContainerRemoveOptions{})
	if err != nil {
		glog.Errorf("Remove container %s failed: %s", dockerId, err.Error())
	} else {
		glog.Infof("Removed container %s for %s", dockerId, podName)
	}
}
