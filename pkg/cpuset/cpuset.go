package cpuset

import (
	"github.com/containerd/cgroups"
	cgroupsv2 "github.com/containerd/cgroups/v2"
	"github.com/go-logr/logr"
	"github.com/opencontainers/runtime-spec/specs-go"
	"os"
	"path"
)

type CPUSetController struct {
	cgroupsDriver    CgroupsDriver
	containerRuntime ContainerRuntime
	cgroupsPath      string
	logger           logr.Logger
}

func NewCPUSetController(cgroupsDriver CgroupsDriver, containerRuntime ContainerRuntime, cgroupsPath string, logger logr.Logger) *CPUSetController {
	logger.Info("Cpuset controller initialized", "cgroupsDriver", cgroupsDriver, "containerRuntime", containerRuntime, "cgroupsPath", cgroupsPath)
	return &CPUSetController{
		cgroupsDriver:    cgroupsDriver,
		cgroupsPath:      cgroupsPath,
		containerRuntime: containerRuntime,
		logger:           logger,
	}
}

func (c *CPUSetController) UpdateCPUSet(container ContainerInfo, cpus, mems string) error {
	sliceName := SliceName(container, c.containerRuntime, c.cgroupsDriver)
	c.logger.Info("Updating cpuset", "container", container, "cpus", cpus, "slice", sliceName)
	if cgroups.Mode() == cgroups.Unified {
		return c.updateCPUSetV2(sliceName, cpus, mems)
	}
	return c.updateCPUSetV1(sliceName, cpus, mems)
}

// updateCPUSetV1 updates cgroups for v1 mode.
func (c *CPUSetController) updateCPUSetV1(slice, cpus, mems string) error {
	ctrl := cgroups.NewCpuset(c.cgroupsPath)
	err := ctrl.Update(slice, &specs.LinuxResources{
		CPU: &specs.LinuxCPU{
			Cpus: cpus,
			Mems: mems,
		},
	})

	// Enable memory migration in cgroups v1 if memory set is specified.
	if err == nil && mems != "" {
		migratePath := path.Join(c.cgroupsPath, "cpuset", slice, "cpuset.memory_migrate")
		err = os.WriteFile(migratePath, []byte("1"), os.ModePerm)
	}

	return err
}

// updateCPUSetV2 updates cgroups for v2 (unified) mode.
func (c *CPUSetController) updateCPUSetV2(slice, cpus, mems string) error {
	res := cgroupsv2.Resources{CPU: &cgroupsv2.CPU{
		Cpus: cpus,
		Mems: mems,
	}}
	_, err := cgroupsv2.NewManager(c.cgroupsPath, slice, &res)
	// Memory migration in cgroups v2 is always enabled, no need to set it.
	return err
}
