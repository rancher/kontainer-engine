package cmd

import (
	"io/ioutil"

	"github.com/rancher/kontainer-engine/cluster"
	generic "github.com/rancher/kontainer-engine/driver"
	"github.com/rancher/kontainer-engine/utils"
	yaml "gopkg.in/yaml.v2"
)

// runRPCDriver runs the rpc server and returns
func runRPCDriver(driverName string) (*generic.GrpcClient, string, error) {
	// addrChan is the channel to receive the server listen address
	addrChan := make(chan string)
	runDriver(driverName, addrChan)

	addr := <-addrChan
	rpcClient, err := generic.NewClient(driverName, addr)
	if err != nil {
		return nil, "", err
	}
	return rpcClient, addr, nil
}

func GetConfigFromFile() (cluster.KubeConfig, error) {
	configFile := utils.KubeConfigFilePath()
	config := cluster.KubeConfig{}
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return cluster.KubeConfig{}, err
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return cluster.KubeConfig{}, err
	}
	return config, nil
}

func SetConfigToFile(config cluster.KubeConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	return utils.WriteToFile(data, utils.KubeConfigFilePath())
}

func deleteConfigByName(config *cluster.KubeConfig, name string) {
	contexts := []cluster.ConfigContext{}
	for _, context := range config.Contexts {
		if context.Name != name {
			contexts = append(contexts, context)
		}
	}
	clusters := []cluster.ConfigCluster{}
	for _, cls := range config.Clusters {
		if cls.Name != name {
			clusters = append(clusters, cls)
		}
	}
	users := []cluster.ConfigUser{}
	for _, user := range config.Users {
		if user.Name != name {
			users = append(users, user)
		}
	}
	config.Contexts = contexts
	config.Clusters = clusters
	config.Users = users
}
