package config

import (
	"context"
	"errors"
	"github.com/stefanaki/cpuset-plugin/pkg/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
)

type PoolConfig struct {
	NodeSelector map[string]string `yaml:"nodeSelector"`
	Pools        []Pool            `yaml:"pools"`
}

type Pool struct {
	Name      string `yaml:"name"`
	CPUs      string `yaml:"cpus"`
	Exclusive bool   `yaml:"exclusive"`
}

type Config struct {
	client *kubernetes.Clientset
	PoolConfig
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

	nodeLabels, err := getNodeLabels(clientset, nodeName)
	if err != nil {
		return nil, err
	}

	pools, nodeSelector, err := getPoolsForNode(rawConfig, nodeLabels)
	if err != nil {
		return nil, err
	}

	return &Config{
		client: clientset,
		PoolConfig: PoolConfig{
			NodeSelector: nodeSelector,
			Pools:        pools,
		},
	}, nil
}

func getCPUSetConfigMap(client *kubernetes.Clientset) (map[string]string, error) {
	config, err := client.
		CoreV1().
		ConfigMaps("kube-system").
		Get(context.TODO(), "cpu-pools", metav1.GetOptions{})

	if err != nil {
		return nil, err
	}
	return config.Data, nil
}

func getNodeLabels(client *kubernetes.Clientset, nodeName string) (map[string]string, error) {
	node, err := client.
		CoreV1().
		Nodes().
		Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return node.Labels, nil
}

func getPoolsForNode(rawConfig map[string]string, nodeLabels map[string]string) ([]Pool, map[string]string, error) {
	for _, c := range rawConfig {
		var config PoolConfig
		err := yaml.Unmarshal([]byte(c), &config)
		if err != nil {
			return nil, nil, err
		}
		if config.NodeSelector == nil {
			return config.Pools, nil, nil
		}
		for labelKey, labelVal := range nodeLabels {
			if val, ok := config.NodeSelector[labelKey]; ok && val == labelVal {
				return config.Pools, config.NodeSelector, nil
			}
		}
	}
	return nil, nil, errors.New("no config found for node")
}
