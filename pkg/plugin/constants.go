package plugin

const Vendor = "stefanaki.github.com"

type ResourceName string

const (
	ResourceNameNUMA   ResourceName = "numa"
	ResourceNameSocket ResourceName = "socket"
	ResourceNameCore   ResourceName = "core"
	ResourceNameCPU    ResourceName = "cpu"
)

const (
	SocketFileNUMA   = "numa.sock"
	SocketFileSocket = "socket.sock"
	SocketFileCore   = "core.sock"
	SocketFileCPU    = "cpu.sock"
)
