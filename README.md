# Kubernetes CPU Device Plugin

Expose CPU resources as consumable devices in Kubernetes.

## Prerequisites

- Enable `PodResources` feature gates in the kubelet configuration and restart the kubelet.
  ```yaml
  # /var/lib/kubelet/config.yaml
  # ...
  featureGates:
      KubeletPodResources: true
      KubeletPodResourcesGet: true
      KubeletPodResourcesGetAllocatable: true
  ```
  
## Installation

1. Clone the repository and navigate to the root directory.
   ```bash
    git clone https://github.com/stefanaki/cpuset-device-plugin.git
    cd cpuset-device-plugin
    ```
2. Edit the environment variables of the DaemonSet to match your system.
   ```yaml
    # manifests/cpuset-device-plugin-daemonset.yaml
    # ...
     containers:
      - image: stefanaki/cpuset-device-plugin:0.0.1
        imagePullPolicy: Always
        args:
          - "--node-name=$(NODE_NAME)"
          - "--container-runtime=docker"
          - "--cgroups-path=/sys/fs/cgroup"
          - "--cgroups-driver=systemd"
    # ...
    ```

3. Apply the device plugin manifest.
   ```bash
   kubectl apply -f manifests/cpuset-device-plugin-daemonset.yaml
   ```
   
## Usage

You can view the available CPU resources by describing the node.
```bash
kubectl describe node <node-name>
```

```bash
# ...
Capacity:
  cpu:                          16
  ephemeral-storage:            967735612Ki
  hugepages-1Gi:                0
  hugepages-2Mi:                0
  memory:                       14161548Ki
  pods:                         110
  stefanaki.github.com/core:    8
  stefanaki.github.com/cpu:     16
  stefanaki.github.com/numa:    1
  stefanaki.github.com/socket:  1
```

In your pod specification, you can request an amount of `numa`/`socket`/`core`/`cpu` resources.


```yaml
containers:
- name: benchmark
  image: spirals/parsec-3.0
  resources:
    requests:
      "cpu": "4000m"
      "stefanaki.github.com/core": "2"
    limits:
      "cpu": "4000m"
      "stefanaki.github.com/core": "2"
```

The daemon will set the `cpuset.cpus` and `cpuset.mems` of the container to the requested resources.
