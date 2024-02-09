package plugin

import (
	"context"
	"fmt"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
	"k8s.io/utils/cpuset"
	"strconv"
	"time"
)

func (c CPUSetDevicePluginDriver) GetDevicePluginOptions(ctx context.Context, empty *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return &pluginapi.DevicePluginOptions{
		PreStartRequired:                false,
		GetPreferredAllocationAvailable: false,
	}, nil
}

func (c CPUSetDevicePluginDriver) ListAndWatch(empty *pluginapi.Empty, server pluginapi.DevicePlugin_ListAndWatchServer) error {
	response := &pluginapi.ListAndWatchResponse{
		Devices: make([]*pluginapi.Device, 0),
	}
	for {
		cpus, err := cpuset.Parse(c.poolConfig.CPUs)
		if err != nil {
			return err
		}
		for _, cpu := range cpus.List() {
			if c.poolConfig.Exclusive {
				response.Devices = append(response.Devices, &pluginapi.Device{
					ID:     strconv.Itoa(cpu),
					Health: pluginapi.Healthy,
				})
			} else {
				for i := 0; i < cpus.Size()*1000; i++ {
					response.Devices = append(response.Devices, &pluginapi.Device{
						ID:     strconv.Itoa(i),
						Health: pluginapi.Healthy,
					})
				}
			}
		}
		if err := server.Send(response); err != nil {
			return err
		}
		time.Sleep(5 * time.Second)
	}
}

func (c CPUSetDevicePluginDriver) GetPreferredAllocation(ctx context.Context, request *pluginapi.PreferredAllocationRequest) (*pluginapi.PreferredAllocationResponse, error) {
	// TODO infer the preferred allocation from the state of the plugin
	return &pluginapi.PreferredAllocationResponse{}, nil
}

func (c CPUSetDevicePluginDriver) Allocate(ctx context.Context, request *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	response := &pluginapi.AllocateResponse{}
	for _, containerRequests := range request.ContainerRequests {
		fmt.Printf("Container requests: %v\n", containerRequests)
		deviceIDs := containerRequests.DevicesIDs
		cpus := cpuset.New()
		for _, deviceID := range deviceIDs {
			c, _ := cpuset.Parse(deviceID)
			cpus = cpus.Union(c)
		}
		println("CPUS ARE", cpus.String())
		containerEnv := make(map[string]string)
		if c.poolConfig.Exclusive {
			containerEnv["CPUSET"] = cpus.String()
		} else {
			containerEnv["CPUSET"] = c.poolConfig.CPUs
		}
		fmt.Printf("Container env: %v\n", containerEnv)
		response.ContainerResponses = append(response.ContainerResponses, &pluginapi.ContainerAllocateResponse{
			Envs: containerEnv,
		})
	}

	return response, nil
}

func (c CPUSetDevicePluginDriver) PreStartContainer(ctx context.Context, request *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	//TODO implement me
	panic("implement me")
}
