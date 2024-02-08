package config

import (
	"context"
	"errors"
	"github.com/stefanaki/cpuset-plugin/pkg/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
)

type NodeCPUConfig struct {
	NodeName string       `yaml:"nodeName"`
	Pools    []PoolConfig `yaml:"pools"`
}

type PoolConfig struct {
	Name      string `yaml:"name"`
	Cpus      string `yaml:"cpus"`
	Exclusive bool   `yaml:"exclusive"`
}

type Config struct {
	client *kubernetes.Clientset
	NodeCPUConfig
}

func NewConfig(nodeName string) (*Config, error) {
	clientset, err := client.NewClient()
	if err != nil {
		return nil, err
	}
	rawConfig, err := getCPUSetConfigMap(clientset)
	if err != nil {
		return nil, err
	}
	pools, err := getPoolsForNode(rawConfig, nodeName)
	if err != nil {
		return nil, err
	}
	return &Config{
		client: clientset,
		NodeCPUConfig: NodeCPUConfig{
			NodeName: nodeName,
			Pools:    pools,
		},
	}, nil
}

func getCPUSetConfigMap(client *kubernetes.Clientset) (map[string]string, error) {
	config, err := client.
		CoreV1().
		ConfigMaps("kube-system").
		Get(context.TODO(), "cpusets", metav1.GetOptions{})

	if err != nil {
		return nil, err
	}
	return config.Data, nil
}

func getPoolsForNode(rawConfig map[string]string, nodeName string) ([]PoolConfig, error) {
	for _, c := range rawConfig {
		var config NodeCPUConfig
		err := yaml.Unmarshal([]byte(c), &config)
		if err != nil {
			return nil, err
		}
		if config.NodeName == nodeName {
			return config.Pools, nil
		}
	}
	return nil, errors.New("no config found for node " + nodeName)
}
