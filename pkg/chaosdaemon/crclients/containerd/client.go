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

package containerd

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	ctrl "sigs.k8s.io/controller-runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/containerd/containerd"

	"github.com/chaos-mesh/chaos-mesh/pkg/mock"
)

var log = ctrl.Log.WithName("containerd-client")

const (
	containerdProtocolPrefix = "containerd://"
)

// ContainerdClientInterface represents the ContainerClient, it's used to simply unit test
type ContainerdClientInterface interface {
	LoadContainer(ctx context.Context, id string) (containerd.Container, error)
}

// ContainerdClient can get information from containerd
type ContainerdClient struct {
	client ContainerdClientInterface
}

// FormatContainerID strips protocol prefix from the container ID
func (c ContainerdClient) FormatContainerID(ctx context.Context, containerID string) (string, error) {
	if len(containerID) < len(containerdProtocolPrefix) {
		return "", fmt.Errorf("container id %s is not a containerd container id", containerID)
	}
	if containerID[0:len(containerdProtocolPrefix)] != containerdProtocolPrefix {
		return "", fmt.Errorf("expected %s but got %s", containerdProtocolPrefix, containerID[0:len(containerdProtocolPrefix)])
	}
	return containerID[len(containerdProtocolPrefix):], nil
}

// GetPidFromContainerID fetches PID according to container id
func (c ContainerdClient) GetPidFromContainerID(ctx context.Context, containerID string) (uint32, error) {
	log.Info("GetPidFromContainerID", "ctx", ctx)
	id, err := c.FormatContainerID(ctx, containerID)
	if err != nil {
		return 0, err
	}
	container, err := c.client.LoadContainer(ctx, id)
	if err != nil {
		return 0, err
	}
	task, err := container.Task(ctx, nil)
	if err != nil {
		return 0, err
	}
	if ctx.Value("type") == "stress" {
		log.Info("type=stress", "value", task.Pid())
		return task.Pid(), nil
	}

	if pid,err := getpid(int(task.Pid())); err !=nil {
		return 0, fmt.Errorf("not funnd pid : %s", err.Error())
	} else {
		log.Info("GetPidFromContainerID", "return", uint32(pid))
		return uint32(pid),nil
	}

	//return task.Pid(), nil
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
func (c ContainerdClient) ContainerKillByContainerID(ctx context.Context, containerID string) error {
	containerID, err := c.FormatContainerID(ctx, containerID)
	if err != nil {
		return err
	}

	container, err := c.client.LoadContainer(ctx, containerID)
	if err != nil {
		return err
	}
	task, err := container.Task(ctx, nil)
	if err != nil {
		return err
	}

	err = task.Kill(ctx, syscall.SIGKILL)

	return err
}

func New(address string, opts ...containerd.ClientOpt) (*ContainerdClient, error) {
	// Mock point to return error in unit test
	if err := mock.On("NewContainerdClientError"); err != nil {
		return nil, err.(error)
	}
	if client := mock.On("MockContainerdClient"); client != nil {
		return &ContainerdClient{
			client.(ContainerdClientInterface),
		}, nil
	}

	c, err := containerd.New(address, opts...)
	if err != nil {
		return nil, err
	}
	// The real logic
	return &ContainerdClient{
		client: c,
	}, nil
}

// WithDefaultNamespace is an alias for the function in containerd with the same name
var WithDefaultNamespace = containerd.WithDefaultNamespace
