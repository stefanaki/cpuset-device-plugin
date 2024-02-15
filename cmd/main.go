package main

import (
	"flag"
	"github.com/fsnotify/fsnotify"
	"github.com/stefanaki/cpuset-plugin/pkg/controller"
	"github.com/stefanaki/cpuset-plugin/pkg/cpuset"
	"github.com/stefanaki/cpuset-plugin/pkg/plugin"
	"k8s.io/klog/v2"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	var nodeName *string = flag.String("node-name", "minikube", "Name of the node")
	var containerRuntime = flag.String("container-runtime", "docker", "Container Runtime (Default: containerd, Values: containerd, docker, kind)")
	var cgroupsPath = flag.String("cgroups-path", "/sys/fs/cgroup", "Path to cgroups")
	var cgroupsDriver = flag.String("cgroups-driver", "systemd", "Set cgroups driver used by kubelet. Values: systemd, cgroupfs")
	flag.Parse()

	logger := klog.NewKlogr()
	logger.Info("Starting cpuset plugin", "node-name", *nodeName, "container-runtime", *containerRuntime, "cgroups-path", *cgroupsPath, "cgroups-driver", *cgroupsDriver)

	state, err := plugin.NewState()
	if err != nil {
		logger.Error(err, "Failed to create daemon state")
		os.Exit(1)
	}
	cpusetController, err := cpuset.NewCPUSetController(*cgroupsDriver, *containerRuntime, *cgroupsPath, logger)
	if err != nil {
		logger.Error(err, "Failed to create cpuset controller")
		os.Exit(1)
	}

	// Controller
	podController, err := controller.NewController(state, cpusetController, logger)
	if err != nil {
		logger.Error(err, "Failed to create controller")
		os.Exit(1)
	}
	controllerStopCh := make(chan struct{})
	err = podController.Run(1, &controllerStopCh)
	if err != nil {
		logger.Error(err, "Failed to run controller")
		os.Exit(1)
	}

	// Device plugins
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Error(err, "Failed to create fsnotify watcher")
		os.Exit(1)
	}
	watcher.Add(pluginapi.KubeletSocket)
	defer watcher.Close()
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	plugins, err := plugin.CreatePluginsForResources(state, logger)
	if err != nil {
		logger.Error(err, "Failed to create device pluginDriver")
		os.Exit(1)
	}

	for {
		select {
		case sig := <-signalCh:
			switch sig {
			case syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT:
				logger.Info("Received signal, shutting down.", sig)
				err := plugin.StopPlugins(plugins)
				if err != nil {
					logger.Error(err, "Failed to stop pool plugin")
				}
				podController.Stop()
				return
			}
			logger.Info("Received signal", sig)
		case event := <-watcher.Events:
			logger.Info("Kubelet change event in pluginpath %v", event)
			err := plugin.StopPlugins(plugins)
			if err != nil {
				logger.Error(err, "Failed to stop pool plugins")
			}
			plugins, err = plugin.CreatePluginsForResources(state, logger)
			if err != nil {
				logger.Error(err, "Failed to create device pluginDriver")
				os.Exit(1)
			}
		}
	}
}
