package topology

import "golang.org/x/exp/maps"

func (t *Topology) GetCPUParentInfo(targetCpuId int) (int, int, int, int) {
	numaID := -1
	for nodeID, node := range t.NUMATopology.Nodes {
		for _, cpu := range node.CPUs.List() {
			if cpu == targetCpuId {
				numaID = nodeID
			}
		}
	}

	for socketID, socket := range t.CPUTopology.Sockets {
		for coreID, core := range socket.Cores {
			for _, cpu := range core.CPUs.List() {
				if cpu == targetCpuId {
					return cpu, coreID, socketID, numaID
				}
			}
		}
	}

	return -1, -1, -1, -1
}

func (t *Topology) GetAllCPUsInCore(targetCoreID int) []int {
	var cpus []int
	for _, socket := range t.CPUTopology.Sockets {
		for coreID, core := range socket.Cores {
			if coreID == targetCoreID {
				for _, cpu := range core.CPUs.List() {
					cpus = append(cpus, cpu)
				}
				return cpus
			}
		}
	}
	return cpus
}

func (t *Topology) GetAllCPUsInSocket(targetSocketID int) []int {
	var cpus []int
	for socketID, socket := range t.CPUTopology.Sockets {
		if socketID == targetSocketID {
			for _, core := range socket.Cores {
				for _, cpu := range core.CPUs.List() {
					cpus = append(cpus, cpu)
				}
			}
		}
	}
	return cpus
}

func (t *Topology) GetAllCPUsInNUMA(targetNUMAID int) []int {
	var cpus []int
	for numaNodeId, numaNode := range t.NUMATopology.Nodes {
		if numaNodeId == targetNUMAID {
			for _, cpu := range numaNode.CPUs.List() {
				cpus = append(cpus, cpu)
			}
			return cpus
		}
	}
	return cpus
}

func (t *Topology) GetNUMANodeForCPU(cpus int) int {
	for nodeID, node := range t.NUMATopology.Nodes {
		if node.CPUs.Contains(cpus) {
			return nodeID
		}
	}
	return -1
}

func (t *Topology) GetNUMANodesForCPUs(cpus []int) []int {
	nodes := make(map[int]struct{})
	for _, cpu := range cpus {
		nodeID := t.GetNUMANodeForCPU(cpu)
		if nodeID != -1 {
			nodes[nodeID] = struct{}{}
		}
	}
	return maps.Keys(nodes)
}
