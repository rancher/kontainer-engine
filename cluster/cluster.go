package cluster

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	generic "github.com/rancher/kontainer-engine/driver"
	"github.com/rancher/kontainer-engine/utils"
	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"
)

const (
	caPem      = "ca.pem"
	clientKey  = "key.pem"
	clientCert = "cert.pem"
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
	Get(name string) (generic.ClusterInfo, error)

	// Remove removes a cluster
	Remove() error

	// DriverName returns the driver name
	DriverName() string

	// Get driver create options flags for creating clusters
	GetDriverCreateOptions() (generic.DriverFlags, error)

	// Get driver update options flags for updating cluster
	GetDriverUpdateOptions() (generic.DriverFlags, error)

	// Set driver options for cluster driver
	SetDriverOptions(options generic.DriverOptions) error
}

// Create creates a cluster
func (c *Cluster) Create() error {
	if c.IsCreated() {
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
	return c.Driver.Update()
}

// Remove removes a cluster
func (c *Cluster) Remove() error {
	driverOptions := generic.DriverOptions{
		BoolOptions:        make(map[string]bool),
		StringOptions:      make(map[string]string),
		IntOptions:         make(map[string]int64),
		StringSliceOptions: make(map[string]*generic.StringSlice),
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

func transformClusterInfo(c *Cluster, clusterInfo generic.ClusterInfo) {
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
	if _, err := os.Stat(filepath.Join(c.getFileDir(), "config.json")); os.IsNotExist(err) {
		return false
	}
	return true
}

func (c *Cluster) getFileDir() string {
	return filepath.Join(utils.HomeDir(), ".netes", "clusters", c.Name)
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
		if err := utils.WriteToFile(data, filepath.Join(c.getFileDir(), v)); err != nil {
			return err
		}
	}
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return utils.WriteToFile(data, filepath.Join(c.getFileDir(), "config.json"))
}

type KubeConfig struct {
	APIVersion string `yaml:"apiVersion,omitempty"`
	Clusters   []struct {
		Cluster struct {
			CertificateAuthorityData string `yaml:"certificate-authority-data,omitempty"`
			Server                   string `yaml:"server,omitempty"`
		} `yaml:"cluster,omitempty"`
		Name string `yaml:"name,omitempty"`
	} `yaml:"clusters,omitempty"`
	Contexts []struct {
		Context struct {
			Cluster string `yaml:"cluster,omitempty"`
			User    string `yaml:"user,omitempty"`
		} `yaml:"context,omitempty"`
		Name string `yaml:"name,omitempty"`
	} `yaml:"contexts,omitempty"`
	CurrentContext string `yaml:"current-context,omitempty"`
	Kind           string `yaml:"kind,omitempty"`
	Preferences    struct {
	} `yaml:"preferences,omitempty"`
	Users []struct {
		Name string `yaml:"name,omitempty"`
		User struct {
			Token    string `yaml:"token,omitempty"`
			Username string `yaml:"username,omitempty"`
			Password string `yaml:"password,omitempty"`
		} `yaml:"user,omitempty"`
	} `yaml:"users,omitempty"`
}

func (c *Cluster) GenerateConfig() error {
	name := fmt.Sprintf("%s-%s", c.DriverName, c.Name)
	isBasicOn := false
	if c.Username != "" && c.Password != "" {
		isBasicOn = true
	}

	config := KubeConfig{
		Kind:       "Config",
		APIVersion: "v1",
		Clusters: make([]struct {
			Cluster struct {
				CertificateAuthorityData string `yaml:"certificate-authority-data,omitempty"`
				Server                   string `yaml:"server,omitempty"`
			} `yaml:"cluster,omitempty"`
			Name string `yaml:"name,omitempty"`
		}, 1),
		Contexts: make([]struct {
			Context struct {
				Cluster string `yaml:"cluster,omitempty"`
				User    string `yaml:"user,omitempty"`
			} `yaml:"context,omitempty"`
			Name string `yaml:"name,omitempty"`
		}, 1),
		Users: make([]struct {
			Name string `yaml:"name,omitempty"`
			User struct {
				Token    string `yaml:"token,omitempty"`
				Username string `yaml:"username,omitempty"`
				Password string `yaml:"password,omitempty"`
			} `yaml:"user,omitempty"`
		}, 1),
	}
	config.CurrentContext = name
	config.Clusters[0].Cluster.CertificateAuthorityData = string(c.RootCACert)
	config.Clusters[0].Cluster.Server = fmt.Sprintf("https://%s", c.Endpoint)
	config.Clusters[0].Name = name
	config.Contexts[0].Context.Cluster = name
	config.Contexts[0].Context.User = name
	config.Contexts[0].Name = name
	config.Users[0].Name = name
	if isBasicOn {
		config.Users[0].User.Username = c.Username
		config.Users[0].User.Password = c.Password
	} else {
		config.Users[0].User.Token = c.ServiceAccountToken
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	fileToWrite := filepath.Join(c.getFileDir(), ".kubeconfig")
	if err := utils.WriteToFile(data, fileToWrite); err != nil {
		return err
	}
	fmt.Printf("\n# KubeConfig files is saved to %s\n", fileToWrite)
	fmt.Printf("# Please run `eval \"$(./kontainer-engine env %s)\"` to change $KUBECONFIG to point to your config file or use --kubeconfig to point to that file\n\n", c.Name)
	fmt.Println(fmt.Sprintf("export KUBECONFIG=%v", fileToWrite))
	return nil
}

// NewCluster create a cluster interface to do operations
func NewCluster(driverName string, ctx *cli.Context) (*Cluster, error) {
	rpcClient, err := generic.NewRPCClient(driverName)
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
func getDriverOpts(ctx *cli.Context) generic.DriverOptions {
	driverOptions := generic.DriverOptions{
		BoolOptions:        make(map[string]bool),
		StringOptions:      make(map[string]string),
		IntOptions:         make(map[string]int64),
		StringSliceOptions: make(map[string]*generic.StringSlice),
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
			driverOptions.StringSliceOptions[flag.GetName()] = &generic.StringSlice{
				Value: ctx.StringSlice(flag.GetName()),
			}
		}
	}
	return driverOptions
}
