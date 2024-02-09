package plugin

import (
	"github.com/stefanaki/cpuset-plugin/pkg/cpuset"
)

type AllocationType string

const (
	AllocationTypeSocket AllocationType = "AllocationTypeSocket"
	AllocationTypeNUMA   AllocationType = "AllocationTypeNUMA"
	AllocationTypeCore   AllocationType = "AllocationTypeCore"
	AllocationTypeCPU    AllocationType = "AllocationTypeCPU"
)

type Allocation struct {
	Container cpuset.ContainerInfo `json:"container"`
	CPUs      string               `json:"cpus"`
	Type      AllocationType       `json:"type"`
}
