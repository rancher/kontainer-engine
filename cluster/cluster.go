package cluster

import (
	rpcDriver "github.com/rancher/kontainer-engine/driver"
	"github.com/sirupsen/logrus"
)

// Cluster represents a kubernetes cluster
type Cluster struct {
	// The cluster driver to provision cluster
	Driver Driver `json:"-"`
	// The name of the cluster driver
	DriverName string `json:"driverName,omitempty" yaml:"driver_name,omitempty"`
	// The name of the cluster
	Name string `json:"name,omitempty" yaml:"name,omitempty"`

	// specific info about kubernetes cluster
	// Kubernetes cluster version
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
	// Service account token to access kubernetes API
	ServiceAccountToken string `json:"serviceAccountToken,omitempty" yaml:"service_account_token,omitempty"`
	// Kubernetes API master endpoint
	Endpoint string `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	// Username for http basic authentication
	Username string `json:"username,omitempty" yaml:"username,omitempty"`
	// Password for http basic authentication
	Password string `json:"password,omitempty" yaml:"password,omitempty"`
	// Root CaCertificate for API server(base64 encoded)
	RootCACert string `json:"rootCACert,omitempty" yaml:"root_ca_cert,omitempty"`
	// Client Certificate(base64 encoded)
	ClientCertificate string `json:"clientCertificate,omitempty" yaml:"client_certificate,omitempty"`
	// Client private key(base64 encoded)
	ClientKey string `json:"clientKey,omitempty" yaml:"client_key,omitempty"`
	// Node count in the cluster
	NodeCount int64 `json:"nodeCount,omitempty" yaml:"node_count,omitempty"`

	// Metadata store specific driver options per cloud provider
	Metadata map[string]string

	PersistStore PersistStore `json:"-,omitempty" yaml:"-,omitempty"`

	ConfigGetter ConfigGetter `json:"-,omitempty" yaml:"-,omitempty"`
}

type PersistStore interface {
	Check(name string) (bool, error)
	Store(cluster Cluster) error
}

type ConfigGetter interface {
	GetConfig() (rpcDriver.DriverOptions, error)
}

// Driver defines how a cluster should be created and managed.
// Different drivers represents different providers.
type Driver interface {
	// Create creates a cluster
	Create() error

	// Update updates a cluster
	Update() error

	// Get a cluster info
	Get(name string) (rpcDriver.ClusterInfo, error)

	// Remove removes a cluster
	Remove() error

	// DriverName returns the driver name
	DriverName() string

	// Get driver create options flags for creating clusters
	GetDriverCreateOptions() (rpcDriver.DriverFlags, error)

	// Get driver update options flags for updating cluster
	GetDriverUpdateOptions() (rpcDriver.DriverFlags, error)

	// Set driver options for cluster driver
	SetDriverOptions(options rpcDriver.DriverOptions) error
}

// Create creates a cluster
func (c *Cluster) Create() error {
	// check if it is already created
	if ok, err := c.IsCreated(); err == nil && ok {
		logrus.Warnf("Cluster %s already exists.", c.Name)
		return nil
	} else if err != nil {
		return err
	}

	// get cluster config from cli flags or json config
	driverOpts, err := c.ConfigGetter.GetConfig()
	if err != nil {
		return err
	}

	// pass cluster config to rpc driver
	if err := c.Driver.SetDriverOptions(driverOpts); err != nil {
		return err
	}

	// create cluster
	if err := c.Driver.Create(); err != nil {
		return err
	}

	// receive cluster info back
	if info, err := c.Driver.Get(c.Name); err != nil {
		return err
	} else {
		transformClusterInfo(c, info)
	}

	// persist cluster info
	return c.Store()
}

// Update updates a cluster
func (c *Cluster) Update() error {
	driverOpts, err := c.ConfigGetter.GetConfig()
	if err != nil {
		return err
	}
	driverOpts.StringOptions["name"] = c.Name
	for k, v := range c.Metadata {
		driverOpts.StringOptions[k] = v
	}
	if err := c.Driver.SetDriverOptions(driverOpts); err != nil {
		return err
	}
	if err := c.Driver.Update(); err != nil {
		return err
	}
	info, err := c.Driver.Get(c.Name)
	if err != nil {
		return err
	}
	transformClusterInfo(c, info)
	return c.Store()
}

func transformClusterInfo(c *Cluster, clusterInfo rpcDriver.ClusterInfo) {
	c.ClientCertificate = clusterInfo.ClientCertificate
	c.ClientKey = clusterInfo.ClientKey
	c.RootCACert = clusterInfo.RootCaCertificate
	c.Username = clusterInfo.Username
	c.Password = clusterInfo.Password
	c.Version = clusterInfo.Version
	c.Endpoint = clusterInfo.Endpoint
	c.NodeCount = clusterInfo.NodeCount
	c.Metadata = clusterInfo.Metadata
	c.ServiceAccountToken = clusterInfo.ServiceAccountToken
}

// Remove removes a cluster
func (c *Cluster) Remove() error {
	driverOptions, err := c.ConfigGetter.GetConfig()
	if err != nil {
		return err
	}
	for k, v := range c.Metadata {
		driverOptions.StringOptions[k] = v
	}
	driverOptions.StringOptions["name"] = c.Name
	if err := c.Driver.SetDriverOptions(driverOptions); err != nil {
		return err
	}
	return c.Driver.Remove()
}

func (c *Cluster) IsCreated() (bool, error) {
	return c.PersistStore.Check(c.Name)
}

// Store persists cluster information
func (c *Cluster) Store() error {
	return c.PersistStore.Store(*c)
}

// NewCluster create a cluster interface to do operations
func NewCluster(driverName, addr, name string, configGetter ConfigGetter, persistStore PersistStore) (*Cluster, error) {
	rpcClient, err := rpcDriver.NewClient(driverName, addr)
	if err != nil {
		return nil, err
	}
	return &Cluster{
		Driver:       rpcClient,
		DriverName:   driverName,
		Name:         name,
		ConfigGetter: configGetter,
		PersistStore: persistStore,
	}, nil
}
