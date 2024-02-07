package cpuset

import (
	"fmt"
)

// CgroupsDriver represents the cgroups driver used by the host.
type CgroupsDriver int

// Supported cgroups drivers.
const (
	DriverSystemd CgroupsDriver = iota
	DriverCgroupfs
)

// ParseCgroupsDriver parses the cgroups driver string and returns the corresponding CgroupsDriver.
func ParseCgroupsDriver(driver string) (CgroupsDriver, error) {
	val, ok := map[string]CgroupsDriver{
		"systemd":  DriverSystemd,
		"cgroupfs": DriverCgroupfs,
	}[driver]
	if !ok {
		return -1, fmt.Errorf("unknown cgroups driver: %s", driver)
	}
	return val, nil
}
