package rke

import (
	"encoding/base64"
	"io/ioutil"

	generic "github.com/rancher/kontainer-engine/driver"
	"github.com/rancher/rke/cmd"
)

// Driver is the struct of rke driver
type Driver struct {
	// The string representation of Config Yaml
	ConfigYaml string
	// Kubernetes master endpoint
	Endpoint string
	// Root certificates
	RootCA string
	// Client certificates
	ClientCert string
	// Client key
	ClientKey string
}

// NewDriver creates a new rke driver
func NewDriver() *Driver {
	return &Driver{}
}

// GetDriverCreateOptions returns create flags for rke driver
func (d *Driver) GetDriverCreateOptions() (*generic.DriverFlags, error) {
	driverFlag := generic.DriverFlags{
		Options: make(map[string]*generic.Flag),
	}
	driverFlag.Options["config-file-path"] = &generic.Flag{
		Type:  generic.StringType,
		Usage: "the path to the config file",
	}
	return &driverFlag, nil
}

// GetDriverUpdateOptions returns update flags for rke driver
func (d *Driver) GetDriverUpdateOptions() (*generic.DriverFlags, error) {
	// todo: rke doesn't have update
	return nil, nil
}

// SetDriverOptions sets the drivers options to rke driver
func (d *Driver) SetDriverOptions(driverOptions *generic.DriverOptions) error {
	// first look up the file path then look up raw rkeConfig
	if path, ok := driverOptions.StringOptions["config-file-path"]; ok {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		d.ConfigYaml = string(data)
		return nil
	}
	d.ConfigYaml = driverOptions.StringOptions["rkeConfig"]
	return nil
}

// Create creates the rke cluster
func (d *Driver) Create() error {
	APIURL, caCrt, clientCert, clientKey, err := cmd.ClusterUp(d.ConfigYaml)
	if err != nil {
		return err
	}
	d.Endpoint = APIURL
	d.RootCA = caCrt
	d.ClientCert = clientCert
	d.ClientKey = clientKey
	return nil
}

// Update updates the rke cluster
func (d *Driver) Update() error {
	APIURL, caCrt, clientCert, clientKey, err := cmd.ClusterUp(d.ConfigYaml)
	if err != nil {
		return err
	}
	d.Endpoint = APIURL
	d.RootCA = caCrt
	d.ClientCert = clientCert
	d.ClientKey = clientKey
	return nil
}

// Get retrieve the cluster info by name
func (d *Driver) Get() (*generic.ClusterInfo, error) {
	// TODO:
	info := &generic.ClusterInfo{}
	info.Endpoint = d.Endpoint
	info.ClientCertificate = base64.StdEncoding.EncodeToString([]byte(d.ClientCert))
	info.ClientKey = base64.StdEncoding.EncodeToString([]byte(d.ClientKey))
	info.RootCaCertificate = base64.StdEncoding.EncodeToString([]byte(d.RootCA))

	token, err := generic.GenerateServiceAccountToken(d.Endpoint, d.RootCA, d.ClientCert, d.ClientKey)
	if err != nil {
		return nil, err
	}
	info.ServiceAccountToken = token
	return info, nil
}

// Remove removes the cluster
func (d *Driver) Remove() error {
	//TODO:
	return nil
}
