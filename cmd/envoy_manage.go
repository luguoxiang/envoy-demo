package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/luguoxiang/envoy-demo/pkg/docker"

	"os"
	"strings"
	"text/tabwriter"
)

type TabWriterOutput struct {
	writer *tabwriter.Writer
}

func NewTabWriterOutput() *TabWriterOutput {
	const padding = 3
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, padding, ' ', tabwriter.Debug)
	var buffer bytes.Buffer
	headers := []string{"ID", "Pod", "Namespace", "Status", "State"}
	for _, header := range headers {
		buffer.WriteString(header)
		buffer.WriteString("\t")
	}
	buffer.WriteString("\n")
	for _, header := range headers {
		buffer.WriteString(strings.Repeat("-", len(header)))
		buffer.WriteString("\t")
	}
	fmt.Fprintln(writer, buffer.String())
	return &TabWriterOutput{writer: writer}
}

func (out *TabWriterOutput) WriteRecord(instance *docker.DockerInstanceInfo) {
	var buffer bytes.Buffer

	buffer.WriteString(instance.ID)
	buffer.WriteString("\t")
	buffer.WriteString(instance.Pod)
	buffer.WriteString("\t")
	buffer.WriteString(instance.Namespace)
	buffer.WriteString("\t")
	buffer.WriteString(instance.Status)
	buffer.WriteString("\t")
	buffer.WriteString(instance.State)
	buffer.WriteString("\t")
	fmt.Fprintln(out.writer, buffer.String())
}

func main() {
	list := flag.Bool("list", true, "list envoy docker instances")
	remove := flag.Bool("remove", false, "remove envoy docker instances")
	id := flag.String("id", "", "docker id")
	log := flag.Bool("log", false, "show envoy logs")
	flag.Parse()

	dockerClient, err := docker.NewDockerClient(false)
	if err != nil {
		panic(err.Error())
		return
	}
	instances, err := dockerClient.ListDockerInstances()
	if err != nil {
		panic(err.Error())
		return
	}
	if *id != "" {
		var found *docker.DockerInstanceInfo
		for _, instance := range instances {
			if instance.ID == *id {
				found = instance
			}
		}
		if found == nil {
			panic("Could not found docker with id " + *id)
			return
		}
		if *log {
			reader, err := dockerClient.GetDockerInstanceLog(found.ID)
			if err != nil {
				panic(err.Error())
				return
			}
			buf := new(bytes.Buffer)
			buf.ReadFrom(reader)
			fmt.Print(buf.String())
			return
		}
		instances = []*docker.DockerInstanceInfo{found}
	} else if *log {
		panic("-log must be used with -id")
	}
	if *list {
		out := NewTabWriterOutput()
		for _, instance := range instances {
			out.WriteRecord(instance)
		}
		out.writer.Flush()
		return
	}

	if *remove {
		for _, instance := range instances {
			dockerClient.StopDockerInstance(instance.ID, instance.Pod)
			dockerClient.RemoveDockerInstance(instance.ID, instance.Pod)
		}
	}
}
