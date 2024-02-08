package main

import (
	"flag"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/stefanaki/cpuset-plugin/pkg/config"
	"github.com/stefanaki/cpuset-plugin/pkg/plugin"
	"k8s.io/klog/v2"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var (
	nodeName *string
)

func init() {
	nodeName = flag.String("node-name", "minikube-m02", "Name of the node")
	flag.Parse()
}

func main() {
	conf, err := config.NewConfig(*nodeName)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(conf)

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

	poolPlugins, err := plugin.CreatePluginsForPools(conf.Pools)
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
				err := plugin.StopPlugins(poolPlugins)
				if err != nil {
					logger.Error(err, "Failed to stop pool plugin")
				}
				return
			}

			logger.Info("Received signal", sig)
		case event := <-watcher.Events:
			logger.Info("Kubelet change event in pluginpath %v", event)

			err := plugin.StopPlugins(poolPlugins)
			if err != nil {
				logger.Error(err, "Failed to stop pool plugins")
			}

			poolPlugins, err = plugin.CreatePluginsForPools(conf.Pools)
			if err != nil {
				logger.Error(err, "Failed to create device pluginDriver")
				os.Exit(1)
			}
		}
	}
}
