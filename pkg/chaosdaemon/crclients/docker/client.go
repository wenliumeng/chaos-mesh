// Copyright 2021 Chaos Mesh Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package docker

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	dockerclient "github.com/docker/docker/client"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/chaos-mesh/chaos-mesh/pkg/mock"
)

var log = ctrl.Log.WithName("docker-client")

const (
	dockerProtocolPrefix = "docker://"
)

// DockerClientInterface represents the DockerClient, it's used to simply unit test
type DockerClientInterface interface {
	ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error)
	ContainerKill(ctx context.Context, containerID, signal string) error
}

// DockerClient can get information from docker
type DockerClient struct {
	client DockerClientInterface
}

// FormatContainerID strips protocol prefix from the container ID
func (c DockerClient) FormatContainerID(ctx context.Context, containerID string) (string, error) {
	if len(containerID) < len(dockerProtocolPrefix) {
		return "", fmt.Errorf("container id %s is not a docker container id", containerID)
	}
	if containerID[0:len(dockerProtocolPrefix)] != dockerProtocolPrefix {
		return "", fmt.Errorf("expected %s but got %s", dockerProtocolPrefix, containerID[0:len(dockerProtocolPrefix)])
	}
	return containerID[len(dockerProtocolPrefix):], nil
}

// GetPidFromContainerID fetches PID according to container id
func (c DockerClient) GetPidFromContainerID(ctx context.Context, containerID string) (uint32, error) {
	log.Info("GetPidFromContainerID", "ctx", ctx)
	id, err := c.FormatContainerID(ctx, containerID)
	if err != nil {
		return 0, err
	}
	container, err := c.client.ContainerInspect(ctx, id)
	if err != nil {
		return 0, err
	}

	if container.State.Pid == 0 {
		return 0, fmt.Errorf("container is not running, status: %s", container.State.Status)
	}

	if ctx.Value("type") == "stress" {
		log.Info("type=stress", "value", container.State.Pid)
		return uint32(container.State.Pid), nil
	}

	if pid,err := getpid(container.State.Pid); err !=nil {
		return 0, fmt.Errorf("not funnd pid : %s", err.Error())
	} else {
		log.Info("GetPidFromContainerID", "return", uint32(pid))
		return uint32(pid),nil
	}

	//return uint32(container.State.Pid), nil
}

func getpid(pidOri int) (int, error) {
	var pid []byte
	var err error
	var cmd *exec.Cmd
	cmd = exec.Command("pgrep", "-l", "-P", strconv.Itoa(pidOri))
	if pid, err = cmd.Output(); err != nil {
		log.Info("pgrep error ", "pid", pidOri)
		return pidOri, nil
	}
	log.Info("begining step1 ", "cmd", string(pid))
	if strings.Contains(string(pid), "java") {
		atoi, _ := strconv.Atoi(strings.Split(string(pid), " ")[0])
		log.Info("begining step1-1  ", "java-pid", atoi)
		return atoi,nil
	} else if string(pid) == "" {
		log.Info("begining step1-2 Recursion nil")
		return 0, errors.New("Recursion nil")
	} else {
		p := strings.Split(string(pid), " ")[0]
		atoi, _ := strconv.Atoi(p)
		log.Info("begining step1-3  ", "other-pid", atoi)
		return getpid(atoi)
	}
}

// ContainerKillByContainerID kills container according to container id
func (c DockerClient) ContainerKillByContainerID(ctx context.Context, containerID string) error {
	id, err := c.FormatContainerID(ctx, containerID)
	if err != nil {
		return err
	}
	err = c.client.ContainerKill(ctx, id, "SIGKILL")

	return err
}

func New(host string, version string, client *http.Client, httpHeaders map[string]string) (*DockerClient, error) {
	// Mock point to return error or mock client in unit test
	if err := mock.On("NewDockerClientError"); err != nil {
		return nil, err.(error)
	}
	if client := mock.On("MockDockerClient"); client != nil {
		return &DockerClient{
			client: client.(DockerClientInterface),
		}, nil
	}

	c, err := dockerclient.NewClientWithOpts(
		dockerclient.WithHost(host),
		dockerclient.WithVersion(version),
		dockerclient.WithHTTPClient(client),
		dockerclient.WithHTTPHeaders(httpHeaders))
	if err != nil {
		return nil, err
	}
	// The real logic
	return &DockerClient{
		client: c,
	}, nil
}
