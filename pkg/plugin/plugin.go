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
	response := &pluginapi.ListAndWatchResponse{
		Devices: make([]*pluginapi.Device, 0),
	}
	for {
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
		deviceIDs := containerRequests.DevicesIDs
		cpus := cpuset.New()
		for _, deviceID := range deviceIDs {
			id, _ := strconv.Atoi(deviceID)
			switch c.allocationType {
			case AllocationTypeNUMA:
				cpus = cpus.Union(cpuset.New(c.state.Topology.GetAllCPUsInNUMA(id)...))
			case AllocationTypeSocket:
				cpus = cpus.Union(cpuset.New(c.state.Topology.GetAllCPUsInSocket(id)...))
			case AllocationTypeCore:
				cpus = cpus.Union(cpuset.New(c.state.Topology.GetAllCPUsInCore(id)...))
			case AllocationTypeCPU:
				cpus = cpus.Union(cpuset.New(id))
			}
		}
		containerEnv := make(map[string]string)
		containerEnv["CPUSET"] = cpus.String()
		response.ContainerResponses = append(response.ContainerResponses, &pluginapi.ContainerAllocateResponse{
			Envs: containerEnv,
		})
	}
	c.logger.Info("response", "res", response)
	return response, nil
}

func (c CPUSetDevicePluginDriver) PreStartContainer(ctx context.Context, request *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (c CPUSetDevicePluginDriver) getPluginResources() []int {
	res := make(map[int]struct{})
	switch c.allocationType {
	case AllocationTypeNUMA:
		for node := range c.state.Topology.NUMATopology.Nodes {
			res[node] = struct{}{}
		}
	case AllocationTypeSocket:
		for socket := range c.state.Topology.CPUTopology.Sockets {
			res[socket] = struct{}{}
		}
	case AllocationTypeCore:
		for _, socket := range c.state.Topology.CPUTopology.Sockets {
			for core := range socket.Cores {
				res[core] = struct{}{}
			}
		}
	case AllocationTypeCPU:
		for _, socket := range c.state.Topology.CPUTopology.Sockets {
			for _, core := range socket.Cores {
				for _, cpu := range core.CPUs.List() {
					res[cpu] = struct{}{}
				}
			}
		}
	}
	return maps.Keys(res)
}

//
//func (c CPUSetDevicePluginDriver) getAllocatableResources() []int {
//	allocations := c.state.GetAllocations()
//	allocatableResources := c.getPluginResources()
//	t := c.state.Topology
//	for _, alloc := range allocations {
//		cs, _ := cpuset.Parse(alloc.CPUs)
//		for _, cpuID := range cs.List() {
//			_, coreID, socketID, numaID := c.state.Topology.GetCPUParentInfo(cpuID)
//			switch alloc.Type {
//			case AllocationTypeNUMA:
//				c.excludeFromAllocatableResources(&allocatableResources, t.GetAllCPUsInNUMA(numaID))
//			case AllocationTypeSocket:
//				c.excludeFromAllocatableResources(&allocatableResources, t.GetAllCPUsInSocket(socketID))
//			case AllocationTypeCore:
//				c.excludeFromAllocatableResources(&allocatableResources, t.GetAllCPUsInCore(coreID))
//			case AllocationTypeCPU:
//				c.excludeFromAllocatableResources(&allocatableResources, []int{cpuID})
//			}
//		}
//	}
//	return maps.Keys(allocatableResources)
//}
//
//func (c CPUSetDevicePluginDriver) excludeFromAllocatableResources(allocatable *map[int]struct{}, cpus []int) {
//	for _, cpu := range cpus {
//		_, coreID, socketID, numaID := c.state.Topology.GetCPUParentInfo(cpu)
//		switch c.allocationType {
//		case AllocationTypeNUMA:
//			delete(*allocatable, numaID)
//		case AllocationTypeSocket:
//			delete(*allocatable, socketID)
//		case AllocationTypeCore:
//			delete(*allocatable, coreID)
//		case AllocationTypeCPU:
//			delete(*allocatable, cpu)
//		}
//	}
//}
