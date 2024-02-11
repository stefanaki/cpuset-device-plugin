package cpuset

import (
	"fmt"
	"strings"
)

// SliceName returns path to container cgroups leaf slice in cgroupfs.
func SliceName(c ContainerInfo, r ContainerRuntime, d CgroupsDriver) string {
	if r == Kind {
		return sliceNameKind(c)
	}
	if d == DriverSystemd {
		return sliceNameDockerContainerdWithSystemd(c, r)
	}
	return sliceNameDockerContainerdWithCgroupfs(c, r)
}

func sliceNameKind(c ContainerInfo) string {
	podType := [3]string{"", "besteffort/", "burstable/"}
	return fmt.Sprintf(
		"kubelet/kubepods/%spod%s/%s",
		podType[c.QoS],
		c.PodID,
		strings.ReplaceAll(c.ContainerID, "containerd://", ""),
	)
}

func sliceNameDockerContainerdWithSystemd(c ContainerInfo, r ContainerRuntime) string {
	sliceType := [3]string{"", "kubepods-besteffort.slice/", "kubepods-burstable.slice/"}
	podType := [3]string{"", "-besteffort", "-burstable"}
	runtimeTypePrefix := [2]string{"docker", "cri-containerd"}
	runtimeURLPrefix := [2]string{"docker://", "containerd://"}
	return fmt.Sprintf(
		"/kubepods.slice/%skubepods%s-pod%s.slice/%s-%s.scope",
		sliceType[c.QoS],
		podType[c.QoS],
		strings.ReplaceAll(c.PodID, "-", "_"),
		runtimeTypePrefix[r],
		strings.ReplaceAll(c.ContainerID, runtimeURLPrefix[r], ""),
	)
}

func sliceNameDockerContainerdWithCgroupfs(c ContainerInfo, r ContainerRuntime) string {
	sliceType := [3]string{"", "besteffort/", "burstable/"}
	runtimeURLPrefix := [2]string{"docker://", "containerd://"}
	return fmt.Sprintf(
		"/kubepods/%spod%s/%s",
		sliceType[c.QoS],
		c.PodID,
		strings.ReplaceAll(c.ContainerID, runtimeURLPrefix[r], ""),
	)
}
