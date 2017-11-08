/*
This package can only be imported if it is running as a library. The init function will start all the driver plugin servers
*/
package stub

import (
	"encoding/json"
	"fmt"

	"github.com/alena1108/cluster-controller/client/v1"
	"github.com/rancher/kontainer-engine/cluster"
	rpcDriver "github.com/rancher/kontainer-engine/driver"
	"github.com/rancher/kontainer-engine/plugin"
	"github.com/sirupsen/logrus"
)

var (
	PluginAddress = map[string]string{}
)

func init() {
	go func() {
		for driver := range plugin.BuiltInDrivers {
			logrus.Infof("Activating driver %s", driver)
			addr := make(chan string)
			plugin.Run(driver, addr)
			listenAddr := <-addr
			PluginAddress[driver] = listenAddr
			logrus.Infof("Activating driver %s done", driver)
		}
	}()
}

type controllerConfigGetter struct {
	driverName string
	cluster    v1.Cluster
}

func (c controllerConfigGetter) GetConfig() (rpcDriver.DriverOptions, error) {
	driverOptions := rpcDriver.DriverOptions{
		BoolOptions:        make(map[string]bool),
		StringOptions:      make(map[string]string),
		IntOptions:         make(map[string]int64),
		StringSliceOptions: make(map[string]*rpcDriver.StringSlice),
	}
	var config interface{}
	switch c.driverName {
	case "gke":
		config = c.cluster.Spec.GKEConfig
	case "aks":
		config = c.cluster.Spec.AKSConfig
	case "rke":
		config = c.cluster.Spec.RKEConfig
	}
	opts, err := toMap(config)
	if err != nil {
		return driverOptions, err
	}
	flatten(opts, &driverOptions)
	driverOptions.StringOptions["name"] = c.cluster.Name

	return driverOptions, nil
}

// flatten take a map and flatten it and convert it into driverOptions
func flatten(data map[string]interface{}, driverOptions *rpcDriver.DriverOptions) {
	for k, v := range data {
		switch v.(type) {
		case float64:
			driverOptions.IntOptions[k] = int64(v.(float64))
		case string:
			driverOptions.StringOptions[k] = v.(string)
		case bool:
			driverOptions.BoolOptions[k] = v.(bool)
		case []string:
			driverOptions.StringSliceOptions[k] = &rpcDriver.StringSlice{Value: v.([]string)}
		case map[string]interface{}:
			// hack for labels
			if k == "labels" {
				r := []string{}
				for key1, value1 := range v.(map[string]interface{}) {
					r = append(r, fmt.Sprintf("%v=%v", key1, value1))
				}
				driverOptions.StringSliceOptions[k] = &rpcDriver.StringSlice{Value: r}
			} else {
				flatten(v.(map[string]interface{}), driverOptions)
			}
		}
	}
}

type controllerPersistStore struct {}

// no-op
func (c controllerPersistStore) Check(name string) (bool, error) {
	return false, nil
}

// no-op
func (c controllerPersistStore) Store(cluster cluster.Cluster) error {
	return nil
}

func toMap(obj interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func convertCluster(cls v1.Cluster) (cluster.Cluster, error) {
	// todo: decide whether we need a driver field
	driverName := ""
	if cls.Spec.AKSConfig != nil {
		driverName = "aks"
	} else if cls.Spec.GKEConfig != nil {
		driverName = "gke"
	} else if cls.Spec.RKEConfig != nil {
		driverName = "rke"
	}
	if driverName == "" {
		return cluster.Cluster{}, fmt.Errorf("no driver config found")
	}
	pluginAddr := PluginAddress[driverName]
	configGetter := controllerConfigGetter{
		driverName: driverName,
		cluster:    cls,
	}
	persistStore := controllerPersistStore{}
	clusterPlugin, err := cluster.NewCluster(driverName, pluginAddr, cls.Name, configGetter, persistStore)
	if err != nil {
		return cluster.Cluster{}, err
	}
	return *clusterPlugin, nil
}

// stub for cluster manager to call
func Create(cluster v1.Cluster) (v1.Cluster, error) {
	cls, err := convertCluster(cluster)
	if err != nil {
		return v1.Cluster{}, err
	}
	if err := cls.Create(); err != nil {
		return v1.Cluster{}, err
	}
	if cluster.Status == nil {
		cluster.Status = &v1.ClusterStatus{}
	}
	cluster.Status.APIEndpoint = fmt.Sprintf("http://%s", cls.Endpoint)
	cluster.Status.ServiceAccountToken = cls.ServiceAccountToken
	// todo: cacerts
	return cluster, nil
}

// stub for cluster manager to call
func Update(cluster v1.Cluster) error {
	cls, err := convertCluster(cluster)
	if err != nil {
		return err
	}
	return cls.Update()
}

// stub for cluster manager to call
func Remove(cluster v1.Cluster) error {
	cls, err := convertCluster(cluster)
	if err != nil {
		return err
	}
	return cls.Remove()
}
