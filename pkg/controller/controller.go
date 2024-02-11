package controller

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/stefanaki/cpuset-plugin/pkg/client"
	"github.com/stefanaki/cpuset-plugin/pkg/cpuset"
	"github.com/stefanaki/cpuset-plugin/pkg/plugin"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	podresources "k8s.io/kubelet/pkg/apis/podresources/v1"
	cpusetutils "k8s.io/utils/cpuset"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// Controller is responsible for managing the reconciliation and event handling of Pods in Kubernetes.
type Controller struct {
	state                  *plugin.State
	informerFactory        informers.SharedInformerFactory
	queue                  workqueue.RateLimitingInterface
	client                 kubernetes.Interface
	informer               cache.SharedInformer
	cpusetController       *cpuset.CPUSetController
	podResourcesClient     podresources.PodResourcesListerClient
	podresourcesConnection *grpc.ClientConn
	stopCh                 *chan struct{}
	logger                 logr.Logger
}

// NewController creates a new instance of the Controller.
func NewController(state *plugin.State, cpusetController *cpuset.CPUSetController, logger logr.Logger) (*Controller, error) {
	controller := &Controller{}

	// Create the Kubernetes clientset
	clientset, err := client.NewClient()
	if err != nil {
		return nil, err
	}

	// Create the work queue and informer factory
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	informerFactory := informers.NewSharedInformerFactory(clientset, 30*time.Second)

	// Create the Pod informer and add event handlers
	podInformer := informerFactory.Core().V1().Pods().Informer()
	_, err = podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			controller.handleAddPod(obj.(*corev1.Pod))
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			controller.handleUpdatePod(newObj.(*corev1.Pod))
		},
		DeleteFunc: nil,
	})
	if err != nil {
		return nil, err
	}

	// Set the watch error handler for the Pod informer
	err = podInformer.SetWatchErrorHandler(controller.WatchErrorHandler)
	if err != nil {
		return nil, err
	}

	controller.state = state
	controller.informerFactory = informerFactory
	controller.queue = queue
	controller.informer = podInformer
	controller.client = clientset
	controller.cpusetController = cpusetController
	controller.logger = logger.WithName("controller")

	conn, err := grpc.Dial("/var/lib/kubelet/pod-resources/kubelet.sock", grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			d := &net.Dialer{}
			return d.DialContext(ctx, "unix", addr)
		}))
	if err != nil {
		controller.logger.Error(err, "CPU Device Plugin cannot connect to Kubelet service")
		return nil, err
	}
	controller.podResourcesClient = podresources.NewPodResourcesListerClient(conn)
	controller.podresourcesConnection = conn

	return controller, nil
}

// Run starts the Controller and its worker threads.
func (c *Controller) Run(threadiness int, stopCh *chan struct{}) error {
	c.stopCh = stopCh
	c.informerFactory.Start(*stopCh)

	c.logger.Info("Starting controller...")
	c.logger.Info("Waiting for Pod Controller cache to sync...")

	if ok := cache.WaitForCacheSync(*stopCh, c.informer.HasSynced); !ok {
		return errors.New("failed to sync Pod Controller from cache, Are you sure everything is properly connected")
	}

	c.logger.Info("Starting controller worker threads...", "threadiness", threadiness)
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, *stopCh)
	}

	// c.StartReconciliation()
	c.logger.Info("Controller successfully initialized, worker threads are now serving requests")
	return nil
}

// runWorker runs the worker loop for processing items in the work queue.
func (c *Controller) runWorker() {
	for c.processNextItem() {
	}
}

// processNextItem retrieves the next item from the work queue and processes it.
func (c *Controller) processNextItem() bool {
	obj, shutdown := c.queue.Get()
	if shutdown {
		c.logger.Info("WARNING: Received shutdown command from queue in thread:" + strconv.Itoa(unix.Gettid()))
		return false
	}
	c.processNextItemInQueue(obj)
	return true
}

// processNextItemInQueue processes the item in the work queue.
func (c *Controller) processNextItemInQueue(obj interface{}) {
	defer c.queue.Done(obj)

	pod, ok := obj.(*corev1.Pod)
	if !ok {
		c.queue.Forget(obj)
		c.logger.Error(fmt.Errorf("expected type *v1.Pod, but got %T", obj), "failed to process item in queue")
		return
	}

	c.handlePod(pod)
}

// StartReconciliation starts the periodic reconciliation loop.
func (c *Controller) StartReconciliation() {
	go c.startReconciliationLoop()
	c.logger.Info("Successfully started the periodic reconciliation thread")
}

// startReconciliationLoop is the main loop for the periodic reconciliation.
func (c *Controller) startReconciliationLoop() {
	timeToReconcile := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-timeToReconcile.C:
			// TODO: Implement reconciliation logic here...
		case <-*c.stopCh:
			c.logger.Info("Shutting down the periodic reconciliation thread")
			timeToReconcile.Stop()
			return
		}
	}
}

// WatchErrorHandler handles errors from the informer's watch operation.
func (c *Controller) WatchErrorHandler(r *cache.Reflector, err error) {
	if apierrors.IsResourceExpired(err) || apierrors.IsGone(err) || err == io.EOF {
		c.logger.Info("One of the API watchers closed gracefully, re-establishing connection")
		return
	}

	// The default K8s client retry mechanism expires after a certain amount of time, and just gives up
	// It is better to shut down the whole process now and freshly re-build the watchers, rather than risking becoming a permanent zombie
	c.logger.Error(err, "One of the API watchers closed unexpectedly")

	c.Stop()

	// Give some time for gracefully terminating the connections
	time.Sleep(5 * time.Second)
	os.Exit(0)
}

// Stop initiates a graceful shutdown procedure for the Controller.
func (c *Controller) Stop() {
	*c.stopCh <- struct{}{}
	c.podresourcesConnection.Close()
	c.queue.ShutDown()
}

func (c *Controller) handleAddPod(pod *corev1.Pod) {
	if c.validatePod(pod) {
		fmt.Printf("Pod %s added and added to queue\n", pod.Name)
		c.queue.Add(pod)
	}
}

func (c *Controller) handleUpdatePod(pod *corev1.Pod) {
	if pod.GetDeletionTimestamp() != nil {
		c.logger.Info("deletion timestamp found", "name", pod.Name, "deletionTimestamp", pod.GetDeletionTimestamp())
		c.handleDeletePod(pod)
		return
	}

	if c.validatePod(pod) {
		fmt.Printf("Pod %s updated and added to queue\n", pod.Name)
		c.queue.Add(pod)
	}
}

func (c *Controller) handleDeletePod(pod *corev1.Pod) {
	if pod.Spec.NodeName != os.Getenv("NODE_NAME") {
		return
	}

	for _, container := range pod.Spec.Containers {
		for resourceName := range container.Resources.Requests {
			c.logger.Info("resourceName name", "name", resourceName.String())
			if !strings.Contains(resourceName.String(), plugin.Vendor) {
				continue
			}
			c.state.RemoveAllocation(cpuset.GetContainerInfo(container, *pod).Name)
			c.logger.Info("Pod deleted", "state", c.state)
		}
	}
}

func (c *Controller) handlePod(pod *corev1.Pod) {
	c.logger.Info("Processing Pod...", "name", pod.Name)

	podResources, err := c.podResourcesClient.Get(context.TODO(), &podresources.GetPodResourcesRequest{
		PodName:      pod.Name,
		PodNamespace: pod.Namespace,
	})
	if err != nil {
		c.logger.Error(err, "Failed to get pod resources", "name", pod.Name)
		return
	}

	for _, container := range pod.Spec.Containers {
		containerInfo := cpuset.GetContainerInfo(container, *pod)
		cpus := cpusetutils.New()
		allocationType := plugin.AllocationTypeCPU
		for resourceName := range container.Resources.Requests {
			if !strings.Contains(resourceName.String(), plugin.Vendor) {
				continue
			}
			for _, containerResources := range podResources.GetPodResources().GetContainers() {
				if containerResources.Name != container.Name {
					continue
				}
				for _, device := range containerResources.GetDevices() {
					for _, deviceId := range device.GetDeviceIds() {
						id, _ := strconv.Atoi(deviceId)
						if strings.Contains(resourceName.String(), "numa") {
							cpus = cpus.Union(cpusetutils.New(c.state.Topology.GetAllCPUsInNUMA(id)...))
							allocationType = plugin.AllocationTypeNUMA
						} else if strings.Contains(resourceName.String(), "socket") {
							cpus = cpus.Union(cpusetutils.New(c.state.Topology.GetAllCPUsInSocket(id)...))
							allocationType = plugin.AllocationTypeSocket
						} else if strings.Contains(resourceName.String(), "core") {
							cpus = cpus.Union(cpusetutils.New(c.state.Topology.GetAllCPUsInCore(id)...))
							allocationType = plugin.AllocationTypeCore
						} else {
							cpus = cpus.Union(cpusetutils.New(id))
						}
					}
				}
			}
			mems := c.state.Topology.GetNUMANodesForCPUs(cpus.List())
			memStr := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(mems)), ","), "[]")
			err := c.cpusetController.UpdateCPUSet(cpuset.GetContainerInfo(container, *pod), cpus.String(), memStr)
			if err != nil {
				c.logger.Error(err, "Failed to update cpuset for container", "name", container.Name)
				return
			}
			c.state.AddAllocation(containerInfo.Name, plugin.Allocation{
				CPUs: cpus.String(),
				Type: allocationType,
			})
		}
	}
}

func (c *Controller) validatePod(pod *corev1.Pod) bool {
	if pod.Spec.NodeName != os.Getenv("NODE_NAME") {
		return false
	}
	if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
		return false
	}
	if len(pod.Status.ContainerStatuses) != len(pod.Spec.Containers) {
		return false
	}
	for _, container := range pod.Status.ContainerStatuses {
		if container.ContainerID == "" {
			return false
		}
	}
	for _, container := range pod.Spec.Containers {
		for resourceName := range container.Resources.Requests {
			if strings.Contains(resourceName.String(), plugin.Vendor) {
				return true
			}
		}
	}
	return false
}
