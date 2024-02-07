package cpuset

import "fmt"

// ContainerRuntime represents different CRI used by k8s.
type ContainerRuntime int

// Supported runtimes.
const (
	Docker ContainerRuntime = iota
	ContainerdRunc
	Kind
)

// QoS pod and containers quality of service type.
type QoS int

// QoS classes as defined in K8s.
const (
	Guaranteed QoS = iota
	BestEffort
	Burstable
)

// ContainerInfo Represents a container in the Daemon.
type ContainerInfo struct {
	ContainerID string `json:"containerID"`
	PodID       string `json:"podID"`
	Name        string `json:"name"`
	CPUs        int32  `json:"cpus"`
	QoS         QoS    `json:"qos"`
}

func (cr ContainerRuntime) String() string {
	return []string{
		"Docker",
		"Containerd+Runc",
		"Kind",
	}[cr]
}

func ParseContainerRuntime(runtime string) (ContainerRuntime, error) {
	val, ok := map[string]ContainerRuntime{
		"containerd": ContainerdRunc,
		"kind":       Kind,
		"docker":     Docker,
	}[runtime]
	if !ok {
		return -1, fmt.Errorf("unknown container runtime: %s", runtime)
	}
	return val, nil
}
