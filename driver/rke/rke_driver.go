package rke

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"strings"

	generic "github.com/rancher/kontainer-engine/driver"
	rpcDriver "github.com/rancher/kontainer-engine/driver"
	"github.com/rancher/rke/cmd"
	"github.com/rancher/rke/hosts"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Driver is the struct of rke driver
type Driver struct {
	DockerDialer hosts.DialerFactory
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
	driverFlag := generic.DriverFlags{
		Options: make(map[string]*generic.Flag),
	}
	driverFlag.Options["config-file-path"] = &generic.Flag{
		Type:  generic.StringType,
		Usage: "the path to the config file",
	}
	return &driverFlag, nil
}

// SetDriverOptions sets the drivers options to rke driver
func getYAML(driverOptions *generic.DriverOptions) (string, error) {
	// first look up the file path then look up raw rkeConfig
	if path, ok := driverOptions.StringOptions["config-file-path"]; ok {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	return driverOptions.StringOptions["rkeConfig"], nil
}

// Create creates the rke cluster
func (d *Driver) Create(opts *rpcDriver.DriverOptions) (*rpcDriver.ClusterInfo, error) {
	yaml, err := getYAML(opts)
	if err != nil {
		return nil, err
	}

	rkeConfig, err := generic.ConvertToRkeConfig(yaml)
	if err != nil {
		return nil, err
	}

	APIURL, caCrt, clientCert, clientKey, err := cmd.ClusterUp(context.Background(), &rkeConfig, d.DockerDialer, nil)
	if err != nil {
		return nil, err
	}
	return &rpcDriver.ClusterInfo{
		Metadata: map[string]string{
			"Endpoint":   APIURL,
			"RootCA":     caCrt,
			"ClientCert": clientCert,
			"ClientKey":  clientKey,
			"Config":     yaml,
		},
	}, nil
}

// Update updates the rke cluster
func (d *Driver) Update(clusterInfo *rpcDriver.ClusterInfo, opts *rpcDriver.DriverOptions) (*rpcDriver.ClusterInfo, error) {
	yaml, err := getYAML(opts)
	if err != nil {
		return nil, err
	}

	rkeConfig, err := generic.ConvertToRkeConfig(yaml)
	if err != nil {
		return nil, err
	}

	APIURL, caCrt, clientCert, clientKey, err := cmd.ClusterUp(context.Background(), &rkeConfig, d.DockerDialer, nil)
	if err != nil {
		return nil, err
	}

	if clusterInfo.Metadata == nil {
		clusterInfo.Metadata = map[string]string{}
	}

	clusterInfo.Metadata["Endpoint"] = APIURL
	clusterInfo.Metadata["RootCA"] = caCrt
	clusterInfo.Metadata["ClientCert"] = clientCert
	clusterInfo.Metadata["ClientKey"] = clientKey
	clusterInfo.Metadata["Config"] = yaml

	return clusterInfo, nil
}

// PostCheck does post action
func (d *Driver) PostCheck(info *rpcDriver.ClusterInfo) (*rpcDriver.ClusterInfo, error) {
	info.Endpoint = info.Metadata["Endpoint"]
	info.ClientCertificate = base64.StdEncoding.EncodeToString([]byte(info.Metadata["ClientCert"]))
	info.ClientKey = base64.StdEncoding.EncodeToString([]byte(info.Metadata["ClientKey"]))
	info.RootCaCertificate = base64.StdEncoding.EncodeToString([]byte(info.Metadata["RootCA"]))

	host := info.Endpoint
	if !strings.HasPrefix(host, "https://") {
		host = fmt.Sprintf("https://%s", host)
	}
	config := &rest.Config{
		Host: host,
		TLSClientConfig: rest.TLSClientConfig{
			CAData:   []byte(info.RootCaCertificate),
			CertData: []byte(info.ClientCertificate),
			KeyData:  []byte(info.ClientKey),
		},
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	serverVersion, err := clientset.DiscoveryClient.ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to get Kubernetes server version: %v", err)
	}

	token, err := generic.GenerateServiceAccountToken(clientset)
	if err != nil {
		return nil, err
	}

	info.Version = serverVersion.GitVersion
	info.ServiceAccountToken = token
	return info, nil
}

// Remove removes the cluster
func (d *Driver) Remove(clusterInfo *rpcDriver.ClusterInfo) error {
	rkeConfig, err := generic.ConvertToRkeConfig(clusterInfo.Metadata["Config"])
	if err != nil {
		return err
	}
	return cmd.ClusterRemove(context.Background(), &rkeConfig, d.DockerDialer)
}
