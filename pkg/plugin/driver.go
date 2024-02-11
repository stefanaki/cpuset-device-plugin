package plugin

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/go-logr/logr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

const Vendor = "stefanaki.github.com"

type CPUSetDevicePluginDriver struct {
	name           string
	socketFile     string
	grpcServer     *grpc.Server
	allocationType AllocationType
	state          *State
	logger         logr.Logger
}

func NewCPUSetDevicePluginDriver(name string, socketFile string, allocationType AllocationType, state *State, logger logr.Logger) (*CPUSetDevicePluginDriver, error) {
	driver := &CPUSetDevicePluginDriver{
		name:           name,
		socketFile:     socketFile,
		allocationType: allocationType,
		state:          state,
		logger:         logger.WithName(fmt.Sprintf("device-%s", name)),
	}
	if err := driver.deleteExistingSocket(); err != nil {
		return nil, fmt.Errorf("failed to delete existing socket: %v", err)
	}
	return driver, nil
}

func (c CPUSetDevicePluginDriver) Register() error {
	conn, err := grpc.Dial(pluginapi.KubeletSocket, grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			d := &net.Dialer{}
			return d.DialContext(ctx, "unix", addr)
		}))
	if err != nil {
		c.logger.Error(err, "CPU Device Plugin cannot connect to Kubelet service")
		return err
	}
	defer conn.Close()
	client := pluginapi.NewRegistrationClient(conn)
	request := &pluginapi.RegisterRequest{
		Version:      pluginapi.Version,
		Endpoint:     c.socketFile,
		ResourceName: fmt.Sprintf("%s/%s", Vendor, c.name),
	}

	if _, err = client.Register(context.Background(), request); err != nil {
		c.logger.Error(err, "CPU Device Plugin cannot register to Kubelet service")
		return err
	}
	c.logger.Info("CPU Device Plugin registered to Kubelet")
	return nil
}

func (c CPUSetDevicePluginDriver) Start() error {
	pluginEndpoint := filepath.Join(pluginapi.DevicePluginPath, c.socketFile)
	c.logger.Info("Starting CPU Device Plugin server", "endpoint", pluginEndpoint)
	if err := os.Remove(c.socketFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing listening address: %v", err)
	}
	lis, err := net.Listen("unix", pluginEndpoint)
	if err != nil {
		c.logger.Error(err, "Starting CPU Device Plugin server failed")
		return err
	}
	c.grpcServer = grpc.NewServer()
	pluginapi.RegisterDevicePluginServer(c.grpcServer, c)
	go func() {
		err := c.grpcServer.Serve(lis)
		if err != nil {
			c.logger.Error(err, "CPU Device Plugin server failed")
		}
	}()

	conn, err := grpc.DialContext(context.Background(), pluginEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithIdleTimeout(5*time.Second),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			d := &net.Dialer{}
			return d.DialContext(ctx, "unix", addr)
		}),
	)

	if err != nil {
		c.logger.Error(err, "Could not establish connection with gRPC server")
		return err
	}

	c.logger.Info("CPU Device Plugin server started serving")
	conn.Close()

	return nil
}

func (c CPUSetDevicePluginDriver) Stop() error {
	c.logger.Info("Stopping CPU Device Plugin server")
	if c.grpcServer != nil {
		c.grpcServer.Stop()
		c.grpcServer = nil
	}
	return c.deleteExistingSocket()
}

func (c CPUSetDevicePluginDriver) deleteExistingSocket() error {
	pluginEndpoint := filepath.Join(pluginapi.DevicePluginPath, c.socketFile)
	if err := os.Remove(pluginEndpoint); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func CreatePluginsForResources(state *State, logger logr.Logger) ([]*CPUSetDevicePluginDriver, error) {
	plugins := make([]*CPUSetDevicePluginDriver, 0)

	// Create NUMA plugin
	numaPlugin, err := NewCPUSetDevicePluginDriver("numa", "numa.sock", AllocationTypeNUMA, state, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create plugins: %v", err)
	}
	plugins = append(plugins, numaPlugin)

	// Create Socket plugin
	socketPlugin, err := NewCPUSetDevicePluginDriver("socket", "socket.sock", AllocationTypeSocket, state, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create plugins: %v", err)
	}
	plugins = append(plugins, socketPlugin)

	// Create Core plugin
	corePlugin, err := NewCPUSetDevicePluginDriver("core", "core.sock", AllocationTypeCore, state, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create plugins: %v", err)
	}
	plugins = append(plugins, corePlugin)

	// Create CPU plugin
	cpuPlugin, err := NewCPUSetDevicePluginDriver("cpu", "cpu.sock", AllocationTypeCPU, state, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create plugins: %v", err)
	}
	plugins = append(plugins, cpuPlugin)

	for _, plugin := range plugins {
		if err := plugin.Start(); err != nil {
			return nil, fmt.Errorf("failed to start plugin: %v", err)
		}
		if err := plugin.Register(); err != nil {
			return nil, fmt.Errorf("failed to register plugin: %v", err)
		}
	}

	return plugins, nil
}

func StopPlugins(poolPlugins []*CPUSetDevicePluginDriver) error {
	for _, poolPlugin := range poolPlugins {
		if err := poolPlugin.Stop(); err != nil {
			return err
		}
	}
	return nil
}
