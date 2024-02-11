package plugin

import (
	"encoding/json"
	"github.com/stefanaki/cpuset-plugin/pkg/topology"
	"os"
	"sync"
)

type State struct {
	Allocations map[string]Allocation `json:"allocations"`
	Topology    *topology.Topology    `json:"topology"`
	mutex       sync.Mutex
}

func (s *State) AddAllocation(containerID string, allocation Allocation) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.Allocations[containerID] = allocation
}

func (s *State) RemoveAllocation(containerID string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.Allocations, containerID)
}

func (s *State) GetAllocations() map[string]Allocation {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.Allocations
}

func NewState() (*State, error) {
	t, err := topology.NewTopology()
	if err != nil {
		return nil, err
	}
	return &State{
		Allocations: make(map[string]Allocation),
		Topology:    t,
	}, nil
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
