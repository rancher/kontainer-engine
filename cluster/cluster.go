package cluster

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	rpcDriver "github.com/rancher/kontainer-engine/driver"
	"github.com/rancher/kontainer-engine/utils"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

const (
	caPem             = "ca.pem"
	clientKey         = "key.pem"
	clientCert        = "cert.pem"
	defaultConfigName = "config.json"
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

	Ctx *cli.Context `json:"-"`
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
	if c.IsCreated() {
		logrus.Warnf("Cluster %s already exists. If it doesn't exist on the provider, make sure to clean them up by running `kontainer-engine rm %s`", c.Name, c.Name)
		return nil
	}
	driverOpts := getDriverOpts(c.Ctx)
	driverOpts.StringOptions["name"] = c.Name
	err := c.Driver.SetDriverOptions(driverOpts)
	if err != nil {
		return err
	}
	if err := c.Driver.Create(); err != nil {
		return err
	}
	info, err := c.Driver.Get(c.Name)
	if err != nil {
		return err
	}
	transformClusterInfo(c, info)
	if err := c.StoreConfig(); err != nil {
		return err
	}
	return c.Store()
}

// Update updates a cluster
func (c *Cluster) Update() error {
	driverOpts := getDriverOpts(c.Ctx)
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

// Remove removes a cluster
func (c *Cluster) Remove() error {
	driverOptions := rpcDriver.DriverOptions{
		BoolOptions:        make(map[string]bool),
		StringOptions:      make(map[string]string),
		IntOptions:         make(map[string]int64),
		StringSliceOptions: make(map[string]*rpcDriver.StringSlice),
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

func (c *Cluster) IsCreated() bool {
	if _, err := os.Stat(filepath.Join(c.GetFileDir(), defaultConfigName)); os.IsNotExist(err) {
		return false
	}
	return true
}

func (c *Cluster) GetFileDir() string {
	return filepath.Join(utils.HomeDir(), "clusters", c.Name)
}

// Store persists cluster information
func (c *Cluster) Store() error {
	// todo: implement store logic to store the cluster info files. this might need to be a interface where we can store on disk or remote
	for k, v := range map[string]string{
		c.RootCACert:        caPem,
		c.ClientKey:         clientKey,
		c.ClientCertificate: clientCert,
	} {
		data, err := base64.StdEncoding.DecodeString(k)
		if err != nil {
			return err
		}
		if err := utils.WriteToFile(data, filepath.Join(c.GetFileDir(), v)); err != nil {
			return err
		}
	}
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return utils.WriteToFile(data, filepath.Join(c.GetFileDir(), defaultConfigName))
}

func (c *Cluster) StoreConfig() error {
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

// NewCluster create a cluster interface to do operations
func NewCluster(driverName string, ctx *cli.Context) (*Cluster, error) {
	addr := ctx.String("plugin-listen-addr")
	rpcClient, err := rpcDriver.NewClient(driverName, addr)
	if err != nil {
		return nil, err
	}
	name := ""
	if ctx.NArg() > 0 {
		name = ctx.Args().Get(0)
	}
	return &Cluster{
		Driver:     rpcClient,
		DriverName: driverName,
		Name:       name,
		Ctx:        ctx,
	}, nil
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
