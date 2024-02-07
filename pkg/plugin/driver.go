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

const Name = "stefanaki.github.com/cpuset"

type CPUSetDevicePluginDriver struct {
	socketFile string
	grpcServer *grpc.Server
	state      *State
	logger     logr.Logger
}

func NewCPUSetDevicePluginDriver(socketFile string, logger logr.Logger) (*CPUSetDevicePluginDriver, error) {
	driver := &CPUSetDevicePluginDriver{
		socketFile: socketFile,
		logger:     logger.WithName("CPUSetDevicePlugin"),
	}
	if err := driver.deleteExistingSocket(); err != nil {
		return nil, fmt.Errorf("failed to delete existing socket: %v", err)
	}
	if err := driver.Start(); err != nil {
		return nil, fmt.Errorf("failed to start CPU Device Plugin: %v", err)
	}
	if err := driver.Register(Name); err != nil {
		return nil, fmt.Errorf("failed to register CPU Device Plugin: %v", err)
	}
	return driver, nil
}

func (c CPUSetDevicePluginDriver) Register(resourceName string) error {
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
		ResourceName: resourceName,
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
	go c.grpcServer.Serve(lis)

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
