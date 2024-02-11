package plugin

type AllocationType string

const (
	AllocationTypeSocket AllocationType = "AllocationTypeSocket"
	AllocationTypeNUMA   AllocationType = "AllocationTypeNUMA"
	AllocationTypeCore   AllocationType = "AllocationTypeCore"
	AllocationTypeCPU    AllocationType = "AllocationTypeCPU"
)

type Allocation struct {
	CPUs string         `json:"cpus"`
	Type AllocationType `json:"type"`
}
