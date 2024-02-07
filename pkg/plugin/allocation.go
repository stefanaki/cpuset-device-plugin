package plugin

import (
	"github.com/stefanaki/cpuset-plugin/pkg/cpuset"
)

type AllocationType string

const (
	// AllocationTypeExclusive indicates that the resource allocation is exclusive.
	AllocationTypeExclusive AllocationType = "exclusive"
	// AllocationTypeShared indicates that the resource allocation is shared.
	AllocationTypeShared AllocationType = "shared"
)

type Allocation struct {
	Container cpuset.ContainerInfo `json:"container"`
	CPUs      string               `json:"cpus"`
	Type      AllocationType       `json:"type"`
}
