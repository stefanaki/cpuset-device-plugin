package plugin

import (
	"encoding/json"
	"fmt"
	"github.com/stefanaki/cpuset-plugin/pkg/topology"
	"golang.org/x/exp/maps"
	"k8s.io/utils/cpuset"
	"os"
	"sync"
)

type State struct {
	Allocations        map[string]Allocation             `json:"allocations"`
	Topology           *topology.Topology                `json:"topology"`
	AvailableResources map[ResourceName]map[int]struct{} `json:"availableResources"`
	mutex              sync.Mutex
}

func (s *State) AddAllocation(containerID string, allocation Allocation) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.Allocations[containerID] = allocation
	cpus, _ := cpuset.Parse(allocation.CPUs)
	for _, cpu := range cpus.List() {
		cpuID, coreID, socketID, numaID := s.Topology.GetCPUParentInfo(cpu)
		delete(s.AvailableResources[ResourceNameCPU], cpuID)
		delete(s.AvailableResources[ResourceNameCore], coreID)
		delete(s.AvailableResources[ResourceNameSocket], socketID)
		delete(s.AvailableResources[ResourceNameNUMA], numaID)
	}

	s.PrintAvailableResources()
}

func (s *State) RemoveAllocation(containerID string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	allocation, ok := s.Allocations[containerID]
	if !ok {
		return
	}

	s.Allocations[containerID] = allocation

	cpus, _ := cpuset.Parse(allocation.CPUs)
	for _, cpu := range cpus.List() {
		cpuID, coreID, socketID, numaID := s.Topology.GetCPUParentInfo(cpu)
		s.AvailableResources[ResourceNameCPU][cpuID] = struct{}{}
		s.AvailableResources[ResourceNameCore][coreID] = struct{}{}
		s.AvailableResources[ResourceNameSocket][socketID] = struct{}{}
		s.AvailableResources[ResourceNameNUMA][numaID] = struct{}{}
	}

	delete(s.Allocations, containerID)

	for _, allocation := range s.Allocations {
		cpus, _ := cpuset.Parse(allocation.CPUs)
		for _, cpu := range cpus.List() {
			cpuID, coreID, socketID, numaID := s.Topology.GetCPUParentInfo(cpu)
			delete(s.AvailableResources[ResourceNameCPU], cpuID)
			delete(s.AvailableResources[ResourceNameCore], coreID)
			delete(s.AvailableResources[ResourceNameSocket], socketID)
			delete(s.AvailableResources[ResourceNameNUMA], numaID)
		}
	}

	s.PrintAvailableResources()
}

func (s *State) GetAllocations() map[string]Allocation {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.Allocations
}

func (s *State) GetAvailableResources() map[ResourceName]map[int]struct{} {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.AvailableResources
}

func NewState() (*State, error) {
	s := &State{}

	t, err := topology.NewTopology()
	if err != nil {
		return nil, err
	}

	availableResources := make(map[ResourceName]map[int]struct{})
	availableResources[ResourceNameNUMA] = make(map[int]struct{})
	availableResources[ResourceNameSocket] = make(map[int]struct{})
	availableResources[ResourceNameCore] = make(map[int]struct{})
	availableResources[ResourceNameCPU] = make(map[int]struct{})

	for nodeID := range t.NUMATopology.Nodes {
		availableResources[ResourceNameNUMA][nodeID] = struct{}{}
	}
	for socketID := range t.CPUTopology.Sockets {
		availableResources[ResourceNameSocket][socketID] = struct{}{}
		for coreID := range t.CPUTopology.Sockets[socketID].Cores {
			availableResources[ResourceNameCore][coreID] = struct{}{}
			for _, cpu := range t.CPUTopology.Sockets[socketID].Cores[coreID].CPUs.List() {
				availableResources[ResourceNameCPU][cpu] = struct{}{}
			}
		}
	}

	s.Allocations = make(map[string]Allocation)
	s.Topology = t
	s.AvailableResources = availableResources

	s.PrintAvailableResources()

	return s, nil
}

func LoadFromFile(filename string) (*State, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	state := &State{}
	err = json.Unmarshal(data, state)
	if err != nil {
		return nil, err
	}
	return state, nil
}

func (s *State) SaveToFile(filename string) error {
	stateJSON, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, stateJSON, 0644)
}

func (s *State) PrintAvailableResources() {
	fmt.Println("Available resources")
	fmt.Printf("numa: %v\n", maps.Keys(s.AvailableResources[ResourceNameNUMA]))
	fmt.Printf("socket: %v\n", maps.Keys(s.AvailableResources[ResourceNameSocket]))
	fmt.Printf("core: %v\n", maps.Keys(s.AvailableResources[ResourceNameCore]))
	fmt.Printf("cpu: %v\n", maps.Keys(s.AvailableResources[ResourceNameCPU]))
}
