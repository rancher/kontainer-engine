/*
Package stub can only be imported if it is running as a library. The init function will start all the driver plugin servers
*/
package stub

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rancher/kontainer-engine/cluster"
	rpcDriver "github.com/rancher/kontainer-engine/driver"
	"github.com/rancher/kontainer-engine/plugin"
	"github.com/rancher/types/apis/management.cattle.io/v3"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	pluginAddress = map[string]string{}
)

func init() {
	go func() {
		for driver := range plugin.BuiltInDrivers {
			logrus.Infof("Activating driver %s", driver)
			addr := make(chan string)
			plugin.Run(driver, addr)
			listenAddr := <-addr
			pluginAddress[driver] = listenAddr
			logrus.Infof("Activating driver %s done", driver)
		}
	}()
}

type controllerConfigGetter struct {
	driverName  string
	clusterSpec v3.ClusterSpec
	clusterName string
}

func (c controllerConfigGetter) GetConfig() (rpcDriver.DriverOptions, error) {
	driverOptions := rpcDriver.DriverOptions{
		BoolOptions:        make(map[string]bool),
		StringOptions:      make(map[string]string),
		IntOptions:         make(map[string]int64),
		StringSliceOptions: make(map[string]*rpcDriver.StringSlice),
	}
	data := map[string]interface{}{}
	switch c.driverName {
	case "gke":
		config, err := toMap(c.clusterSpec.GoogleKubernetesEngineConfig, "json")
		if err != nil {
			return driverOptions, err
		}
		data = config
		flatten(data, &driverOptions)
	case "rke":
		config, err := yaml.Marshal(c.clusterSpec.RancherKubernetesEngineConfig)
		if err != nil {
			return driverOptions, err
		}
		driverOptions.StringOptions["rkeConfig"] = string(config)
	}
	driverOptions.StringOptions["name"] = c.clusterName

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

type controllerPersistStore struct {
	clusterClient v3.ClusterInterface
}

// no-op
func (c controllerPersistStore) Check(name string) (bool, error) {
	return false, nil
}

// no-op
func (c controllerPersistStore) Store(cluster cluster.Cluster) error {
	return nil
}

// no-op
func (c controllerPersistStore) Get(name string) (cluster.Cluster, error) {
	return cluster.Cluster{}, nil
}

// no-op
func (c controllerPersistStore) PersistStatus(cluster cluster.Cluster, status string) error {
	cls, err := c.clusterClient.Get(cluster.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if cls.Status.Conditions == nil {
		cls.Status.Conditions = []v3.ClusterCondition{}
	}
	now := time.Now().Format(time.RFC3339)
	cls.Status.Conditions = append(cls.Status.Conditions, v3.ClusterCondition{
		Type:               v3.ClusterConditionType(status),
		Status:             v1.ConditionTrue,
		LastTransitionTime: now,
		LastUpdateTime:     now,
	})
	_, err = c.clusterClient.Update(cls)
	return err
}

func toMap(obj interface{}, format string) (map[string]interface{}, error) {
	if format == "json" {
		data, err := json.Marshal(obj)
		if err != nil {
			return nil, err
		}
		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, err
		}
		return result, nil
	} else if format == "yaml" {
		data, err := yaml.Marshal(obj)
		if err != nil {
			return nil, err
		}
		var result map[string]interface{}
		if err := yaml.Unmarshal(data, &result); err != nil {
			return nil, err
		}
		return result, nil
	}
	return nil, nil
}

func convertCluster(name string, spec v3.ClusterSpec, clusterClient v3.ClusterInterface) (cluster.Cluster, error) {
	// todo: decide whether we need a driver field
	driverName := ""
	if spec.AzureKubernetesServiceConfig != nil {
		driverName = "aks"
	} else if spec.GoogleKubernetesEngineConfig != nil {
		driverName = "gke"
	} else if spec.RancherKubernetesEngineConfig != nil {
		driverName = "rke"
	}
	if driverName == "" {
		return cluster.Cluster{}, fmt.Errorf("no driver config found")
	}
	pluginAddr := pluginAddress[driverName]
	configGetter := controllerConfigGetter{
		driverName:  driverName,
		clusterSpec: spec,
		clusterName: name,
	}
	persistStore := controllerPersistStore{
		clusterClient: clusterClient,
	}
	clusterPlugin, err := cluster.NewCluster(driverName, pluginAddr, name, configGetter, persistStore)
	if err != nil {
		return cluster.Cluster{}, err
	}
	return *clusterPlugin, nil
}

// Create creates the stub for cluster manager to call
func Create(name string, clusterSpec v3.ClusterSpec, clusterClient v3.ClusterInterface) (string, string, string, error) {
	cls, err := convertCluster(name, clusterSpec, clusterClient)
	if err != nil {
		return "", "", "", err
	}
	if err := cls.Create(); err != nil {
		return "", "", "", err
	}
	endpoint := cls.Endpoint
	if !strings.HasPrefix(endpoint, "http://") {
		endpoint = fmt.Sprintf("http://%s", cls.Endpoint)
	}
	return endpoint, cls.ServiceAccountToken, cls.RootCACert, nil
}

// Update creates the stub for cluster manager to call
func Update(name string, clusterSpec v3.ClusterSpec, clusterClient v3.ClusterInterface) (string, string, string, error) {
	cls, err := convertCluster(name, clusterSpec, clusterClient)
	if err != nil {
		return "", "", "", err
	}
	if err := cls.Update(); err != nil {
		return "", "", "", err
	}
	endpoint := cls.Endpoint
	if !strings.HasPrefix(endpoint, "http://") {
		endpoint = fmt.Sprintf("http://%s", cls.Endpoint)
	}
	return endpoint, cls.ServiceAccountToken, cls.RootCACert, nil
}

// Remove removes stub for cluster manager to call
func Remove(name string, clusterSpec v3.ClusterSpec, clusterClient v3.ClusterInterface) error {
	cls, err := convertCluster(name, clusterSpec, clusterClient)
	if err != nil {
		return err
	}
	return cls.Remove()
}
