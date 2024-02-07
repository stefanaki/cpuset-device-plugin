package main

import (
	"flag"
	"github.com/fsnotify/fsnotify"
	"github.com/stefanaki/cpuset-plugin/pkg/plugin"
	"k8s.io/klog/v2"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	var (
		socketFile = flag.String("socket", "cpuset.sock", "CPUSet device plugin socket file")
	)
	flag.Parse()
	logger := klog.NewKlogr()
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Error(err, "Failed to create fsnotify watcher")
		os.Exit(1)
	}
	watcher.Add(pluginapi.KubeletSocket)
	defer watcher.Close()
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	pluginDriver, err := plugin.NewCPUSetDevicePluginDriver(*socketFile, logger)
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
				if err := pluginDriver.Stop(); err != nil {
					logger.Error(err, "Failed to stop device pluginDriver")
					os.Exit(1)
				}
				return
			}
			logger.Info("Received signal", sig)
		case event := <-watcher.Events:
			logger.Info("Kubelet change event in pluginpath %v", event)
			if err := pluginDriver.Stop(); err != nil {
				logger.Error(err, "Failed to stop device pluginDriver")
			}
			if pluginDriver, err = plugin.NewCPUSetDevicePluginDriver(*socketFile, logger); err != nil {
				log.Fatal(err)
			}
		}
	}
}
