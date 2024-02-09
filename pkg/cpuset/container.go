package cpuset

import (
	"fmt"
	"k8s.io/api/core/v1"
)

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

func QoSFromLimit(limitCpu, requestCpu int64, limitMemory, requestMemory string) QoS {
	if (limitCpu > 0 || requestCpu > 0) || (limitMemory != "0" || requestMemory != "0") {
		if limitCpu > 0 && requestCpu > 0 && limitCpu == requestCpu &&
			limitMemory != "0" && requestMemory != "0" && limitMemory == requestMemory {
			return Guaranteed
		}
		return Burstable
	}
	return BestEffort
}

func GetContainerInfo(container v1.Container, pod v1.Pod) ContainerInfo {
	name := container.Name
	for _, c := range pod.Status.ContainerStatuses {
		if c.Name != name {
			continue
		}

		return ContainerInfo{
			ContainerID: c.ContainerID,
			PodID:       string(pod.ObjectMeta.UID),
			Name:        c.Name,
			QoS: QoSFromLimit(
				container.Resources.Limits.Cpu().MilliValue(),
				container.Resources.Requests.Cpu().MilliValue(),
				container.Resources.Limits.Memory().String(),
				container.Resources.Requests.Memory().String(),
			),
		}
	}
	return ContainerInfo{}
}
