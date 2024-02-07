package topology

import (
	"fmt"
	"k8s.io/utils/cpuset"
	"os/exec"
	"strconv"
	"strings"
)

func LSCPU() ([]byte, error) {
	return exec.Command("lscpu", "-p=socket,node,core,cpu", "--online").Output()
}

func ParseTopologyFromLSCPUOutput(output []byte) (*Topology, error) {
	topology := &Topology{
		CPUTopology: CPUTopology{
			Sockets: make(map[int]Socket),
		},
		NUMATopology: NUMATopology{
			Nodes: make(map[int]NUMANode),
		},
	}
	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, ",")
		if len(fields) != 4 {
			continue
		}
		var socketID, nodeID, coreID, cpuID int
		var err error
		socketID, err = strconv.Atoi(fields[0])
		if socketID, err = strconv.Atoi(fields[0]); err != nil {
			return nil, fmt.Errorf("failed to parse socket ID: %v", err)
		}
		if nodeID, err = strconv.Atoi(fields[1]); err != nil {
			return nil, fmt.Errorf("failed to parse node ID: %v", err)
		}
		if coreID, err = strconv.Atoi(fields[2]); err != nil {
			return nil, fmt.Errorf("failed to parse core ID: %v", err)
		}
		if cpuID, err = strconv.Atoi(fields[3]); err != nil {
			return nil, fmt.Errorf("failed to parse cpu ID: %v", err)
		}

		if _, ok := topology.CPUTopology.Sockets[socketID]; !ok {
			topology.CPUTopology.Sockets[socketID] = Socket{
				Cores: make(map[int]Core),
			}
		}
		if _, ok := topology.CPUTopology.Sockets[socketID].Cores[coreID]; !ok {
			topology.CPUTopology.Sockets[socketID].Cores[coreID] = Core{
				CPUs: cpuset.New(),
			}
		}
		if _, ok := topology.NUMATopology.Nodes[nodeID]; !ok {
			topology.NUMATopology.Nodes[nodeID] = NUMANode{
				CPUs: cpuset.New(),
			}
		}
		c := topology.CPUTopology.Sockets[socketID].Cores[coreID].CPUs.Union(cpuset.New(cpuID))
		n := topology.NUMATopology.Nodes[nodeID].CPUs.Union(cpuset.New(cpuID))
		topology.CPUTopology.Sockets[socketID].Cores[coreID] = Core{
			CPUs:   c,
			CPUStr: c.String(),
		}
		topology.NUMATopology.Nodes[nodeID] = NUMANode{
			CPUs:   n,
			CPUStr: n.String(),
		}
	}

	return topology, nil
}
