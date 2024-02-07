package cpuset

import (
	"github.com/containerd/cgroups"
	cgroupsv2 "github.com/containerd/cgroups/v2"
	"github.com/go-logr/logr"
	"github.com/opencontainers/runtime-spec/specs-go"
	"os"
	"path"
)

type CPUSetsController struct {
	cgroupsDriver CgroupsDriver
	cgroupsPath   string
	logger        logr.Logger
}

func NewCPUSetsController(cgroupsDriver CgroupsDriver, cgroupsPath string, logger logr.Logger) (*CPUSetsController, error) {
	return &CPUSetsController{
		cgroupsDriver: cgroupsDriver,
		cgroupsPath:   cgroupsPath,
		logger:        logger,
	}, nil
}

func (c *CPUSetsController) UpdateCPUSet(slice, cpus, mems string) error {
	if cgroups.Mode() == cgroups.Unified {
		return c.updateCPUSetV2(slice, cpus, mems)
	}
	return c.updateCPUSetV1(slice, cpus, mems)
}

// updateCPUSetV1 updates cgroups for v1 mode.
func (c *CPUSetsController) updateCPUSetV1(slice, cpus, mems string) error {
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
func (c *CPUSetsController) updateCPUSetV2(slice, cpus, mems string) error {
	res := cgroupsv2.Resources{CPU: &cgroupsv2.CPU{
		Cpus: cpus,
		Mems: mems,
	}}
	_, err := cgroupsv2.NewManager(c.cgroupsPath, slice, &res)
	// Memory migration in cgroups v2 is always enabled, no need to set it.
	return err
}
