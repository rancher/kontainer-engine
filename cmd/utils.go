package cmd

import (
	"io/ioutil"

	"github.com/rancher/kontainer-engine/cluster"
	generic "github.com/rancher/kontainer-engine/driver"
	rpcDriver "github.com/rancher/kontainer-engine/driver"
	"github.com/rancher/kontainer-engine/utils"
	"github.com/urfave/cli"
	yaml "gopkg.in/yaml.v2"
	"github.com/sirupsen/logrus"
	"os"
	"fmt"
	"github.com/rancher/kontainer-engine/plugin"
)

// runRPCDriver runs the rpc server and returns
func runRPCDriver(driverName string) (*generic.GrpcClient, string, error) {
	// addrChan is the channel to receive the server listen address
	addrChan := make(chan string)
	plugin.Run(driverName, addrChan)

	addr := <-addrChan
	rpcClient, err := generic.NewClient(driverName, addr)
	if err != nil {
		return nil, "", err
	}
	return rpcClient, addr, nil
}

func GetConfigFromFile() (KubeConfig, error) {
	configFile := utils.KubeConfigFilePath()
	config := KubeConfig{}
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return KubeConfig{}, err
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return KubeConfig{}, err
	}
	return config, nil
}

func SetConfigToFile(config KubeConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	return utils.WriteToFile(data, utils.KubeConfigFilePath())
}

func deleteConfigByName(config *KubeConfig, name string) {
	contexts := []ConfigContext{}
	for _, context := range config.Contexts {
		if context.Name != name {
			contexts = append(contexts, context)
		}
	}
	clusters := []ConfigCluster{}
	for _, cls := range config.Clusters {
		if cls.Name != name {
			clusters = append(clusters, cls)
		}
	}
	users := []ConfigUser{}
	for _, user := range config.Users {
		if user.Name != name {
			users = append(users, user)
		}
	}
	config.Contexts = contexts
	config.Clusters = clusters
	config.Users = users
}

// getDriverOpts get the flags and value and generate DriverOptions
func getDriverOpts(ctx *cli.Context) rpcDriver.DriverOptions {
	driverOptions := rpcDriver.DriverOptions{
		BoolOptions:        make(map[string]bool),
		StringOptions:      make(map[string]string),
		IntOptions:         make(map[string]int64),
		StringSliceOptions: make(map[string]*rpcDriver.StringSlice),
	}
	for _, flag := range ctx.Command.Flags {
		switch flag.(type) {
		case cli.StringFlag:
			driverOptions.StringOptions[flag.GetName()] = ctx.String(flag.GetName())
		case cli.BoolFlag:
			driverOptions.BoolOptions[flag.GetName()] = ctx.Bool(flag.GetName())
		case cli.Int64Flag:
			driverOptions.IntOptions[flag.GetName()] = ctx.Int64(flag.GetName())
		case cli.StringSliceFlag:
			driverOptions.StringSliceOptions[flag.GetName()] = &rpcDriver.StringSlice{
				Value: ctx.StringSlice(flag.GetName()),
			}
		}
	}
	return driverOptions
}

func storeConfig(c cluster.Cluster) error {
	isBasicOn := false
	if c.Username != "" && c.Password != "" {
		isBasicOn = true
	}
	username, password, token := "", "", ""
	if isBasicOn {
		username = c.Username
		password = c.Password
	} else {
		token = c.ServiceAccountToken
	}

	configFile := utils.KubeConfigFilePath()
	config := KubeConfig{}
	if _, err := os.Stat(configFile); err == nil {
		data, err := ioutil.ReadFile(configFile)
		if err != nil {
			return err
		}
		if err := yaml.Unmarshal(data, &config); err != nil {
			return err
		}
	}
	config.APIVersion = "v1"
	config.Kind = "Config"

	// setup clusters
	cluster := ConfigCluster{
		Cluster: DataCluster{
			CertificateAuthorityData: string(c.RootCACert),
			Server: fmt.Sprintf("https://%s", c.Endpoint),
		},
		Name: c.Name,
	}
	if config.Clusters == nil || len(config.Clusters) == 0 {
		config.Clusters = []ConfigCluster{cluster}
	} else {
		exist := false
		for _, cluster := range config.Clusters {
			if cluster.Name == c.Name {
				exist = true
				break
			}
		}
		if !exist {
			config.Clusters = append(config.Clusters, cluster)
		}
	}

	// setup users
	user := ConfigUser{
		User: UserData{
			Username: username,
			Password: password,
			Token:    token,
		},
		Name: c.Name,
	}
	if config.Users == nil || len(config.Users) == 0 {
		config.Users = []ConfigUser{user}
	} else {
		exist := false
		for _, user := range config.Users {
			if user.Name == c.Name {
				exist = true
				break
			}
		}
		if !exist {
			config.Users = append(config.Users, user)
		}
	}

	// setup context
	context := ConfigContext{
		Context: ContextData{
			Cluster: c.Name,
			User:    c.Name,
		},
		Name: c.Name,
	}
	if config.Contexts == nil || len(config.Contexts) == 0 {
		config.Contexts = []ConfigContext{context}
	} else {
		exist := false
		for _, context := range config.Contexts {
			if context.Name == c.Name {
				exist = true
				break
			}
		}
		if !exist {
			config.Contexts = append(config.Contexts, context)
		}
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	fileToWrite := utils.KubeConfigFilePath()
	if err := utils.WriteToFile(data, fileToWrite); err != nil {
		return err
	}
	logrus.Debugf("KubeConfig files is saved to %s", fileToWrite)
	logrus.Debug("Kubeconfig file\n" + string(data))

	return nil
}
