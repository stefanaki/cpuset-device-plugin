package topology

import (
	"k8s.io/utils/cpuset"
)

// Core represents a CPU core.
type Core struct {
	CPUs   cpuset.CPUSet // CPUs is the set of CPUs in the core.
	CPUStr string        `json:"cpus"` // CPUStr is the string representation of the set of CPUs in the core.
}

// Socket represents a CPU socket.
type Socket struct {
	Cores map[int]Core `json:"cores"` // Cores is a map of core ID to Core.
}

// CPUTopology represents the CPU topology.
type CPUTopology struct {
	Sockets map[int]Socket `json:"sockets"` // Sockets is a map of socket ID to Socket.
}

// NUMANode represents a NUMA node.
type NUMANode struct {
	CPUs   cpuset.CPUSet // CPUs is the set of CPUs in the NUMA node.
	CPUStr string        `json:"cpus"` // CPUStr is the string representation of the set of CPUs in the NUMA node.
}

// NUMATopology represents the NUMA topology.
type NUMATopology struct {
	Nodes map[int]NUMANode `json:"nodes"` // Nodes is a map of NUMA node ID to NUMANode.
}

// Topology represents the overall system topology.
type Topology struct {
	CPUTopology  CPUTopology  `json:"cpuTopology"`  // CPUTopology is the CPU topology.
	NUMATopology NUMATopology `json:"numaTopology"` // NUMATopology is the NUMA topology.
}

// NewTopology creates a new Topology instance by parsing system topology.
func NewTopology() (*Topology, error) {
	lscpu, err := LSCPU()
	if err != nil {
		return nil, err
	}
	return ParseTopologyFromLSCPUOutput(lscpu)
}
