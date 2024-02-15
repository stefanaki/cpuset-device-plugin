package plugin

import (
	"context"
	"golang.org/x/exp/maps"
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
	for {
		response := &pluginapi.ListAndWatchResponse{
			Devices: make([]*pluginapi.Device, 0),
		}
		allocatableResources := c.getPluginResources()
		for _, res := range allocatableResources {
			response.Devices = append(response.Devices, &pluginapi.Device{
				ID:     strconv.Itoa(res),
				Health: pluginapi.Healthy,
			})
		}
		if err := server.Send(response); err != nil {
			return err
		}
		time.Sleep(2 * time.Second)
	}
}

func (c CPUSetDevicePluginDriver) Allocate(ctx context.Context, request *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	response := &pluginapi.AllocateResponse{}
	for _, containerRequests := range request.ContainerRequests {
		deviceIDs := containerRequests.DevicesIDs
		cpus := cpuset.New()
		for _, deviceID := range deviceIDs {
			id, _ := strconv.Atoi(deviceID)
			cpus = cpus.Union(c.getCPUSetForDevice(id))
		}
		containerEnv := make(map[string]string)
		containerEnv["CPUSET"] = cpus.String()
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

func (c CPUSetDevicePluginDriver) GetPreferredAllocation(ctx context.Context, request *pluginapi.PreferredAllocationRequest) (*pluginapi.PreferredAllocationResponse, error) {
	// TODO infer the preferred allocation from the state of the plugin
	return &pluginapi.PreferredAllocationResponse{}, nil
}

func (c CPUSetDevicePluginDriver) getPluginResources() []int {
	switch c.allocationType {
	case AllocationTypeNUMA:
		return maps.Keys(c.state.GetAvailableResources()[ResourceNameNUMA])
	case AllocationTypeSocket:
		return maps.Keys(c.state.GetAvailableResources()[ResourceNameSocket])
	case AllocationTypeCore:
		return maps.Keys(c.state.GetAvailableResources()[ResourceNameCore])
	case AllocationTypeCPU:
		return maps.Keys(c.state.GetAvailableResources()[ResourceNameCPU])
	}
	return []int{}
}

func (c CPUSetDevicePluginDriver) getCPUSetForDevice(deviceID int) cpuset.CPUSet {
	switch c.allocationType {
	case AllocationTypeNUMA:
		return cpuset.New(c.state.Topology.GetAllCPUsInNUMA(deviceID)...)
	case AllocationTypeSocket:
		return cpuset.New(c.state.Topology.GetAllCPUsInSocket(deviceID)...)
	case AllocationTypeCore:
		return cpuset.New(c.state.Topology.GetAllCPUsInCore(deviceID)...)
	case AllocationTypeCPU:
		return cpuset.New(deviceID)
	}
	return cpuset.New()
}
