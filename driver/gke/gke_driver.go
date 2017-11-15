package gke

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	generic "github.com/rancher/kontainer-engine/driver"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	raw "google.golang.org/api/container/v1"
	"k8s.io/client-go/kubernetes"
	// to register gcp auth provider
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
)

const (
	defaultNodeCount     = 3
	runningStatus        = "RUNNING"
	defaultNamespace     = "default"
	clusterAdmin         = "cluster-admin"
	netesDefault         = "netes-default"
	defaultCredentialEnv = "GOOGLE_APPLICATION_CREDENTIALS"
)

// Driver defines the struct of gke driver
type Driver struct {
	// ProjectID is the ID of your project to use when creating a cluster
	ProjectID string
	// The zone to launch the cluster
	Zone string
	// The IP address range of the container pods
	ClusterIpv4Cidr string
	// An optional description of this cluster
	Description string
	// The number of nodes to create in this cluster
	NodeCount int64
	// the kubernetes master version
	MasterVersion string
	// The authentication information for accessing the master
	MasterAuth *raw.MasterAuth
	// the kubernetes node version
	NodeVersion string
	// The name of this cluster
	Name string
	// Parameters used in creating the cluster's nodes
	NodeConfig *raw.NodeConfig
	// The path to the credential file(key.json)
	CredentialPath string
	// Enable alpha feature
	EnableAlphaFeature bool
	// NodePool id
	NodePoolID string
}

// NewDriver creates a gke Driver
func NewDriver() *Driver {
	return &Driver{
		NodeConfig: &raw.NodeConfig{
			Labels: map[string]string{},
		},
	}
}

// GetDriverCreateOptions implements driver interface
func (d *Driver) GetDriverCreateOptions() (*generic.DriverFlags, error) {
	driverFlag := generic.DriverFlags{
		Options: make(map[string]*generic.Flag),
	}
	driverFlag.Options["projectId"] = &generic.Flag{
		Type:  generic.StringType,
		Usage: "the ID of your project to use when creating a cluster",
	}
	driverFlag.Options["zone"] = &generic.Flag{
		Type:  generic.StringType,
		Usage: "The zone to launch the cluster",
	}
	driverFlag.Options["gke-credential-path"] = &generic.Flag{
		Type:  generic.StringType,
		Usage: "the path to the credential json file(example: $HOME/key.json)",
	}
	driverFlag.Options["cluster-ipv4-cidr"] = &generic.Flag{
		Type:  generic.StringType,
		Usage: "The IP address range of the container pods",
	}
	driverFlag.Options["description"] = &generic.Flag{
		Type:  generic.StringType,
		Usage: "An optional description of this cluster",
	}
	driverFlag.Options["master-version"] = &generic.Flag{
		Type:  generic.StringType,
		Usage: "The kubernetes master version",
	}
	driverFlag.Options["node-count"] = &generic.Flag{
		Type:  generic.IntType,
		Usage: "The number of nodes to create in this cluster",
	}
	driverFlag.Options["disk-size-gb"] = &generic.Flag{
		Type:  generic.IntType,
		Usage: "Size of the disk attached to each node",
	}
	driverFlag.Options["labels"] = &generic.Flag{
		Type:  generic.StringSliceType,
		Usage: "The map of Kubernetes labels (key/value pairs) to be applied to each node",
	}
	driverFlag.Options["machine-type"] = &generic.Flag{
		Type:  generic.StringType,
		Usage: "The machine type of a Google Compute Engine",
	}
	driverFlag.Options["enable-alpha-feature"] = &generic.Flag{
		Type:  generic.BoolType,
		Usage: "To enable kubernetes alpha feature",
	}
	return &driverFlag, nil
}

// GetDriverUpdateOptions implements driver interface
func (d *Driver) GetDriverUpdateOptions() (*generic.DriverFlags, error) {
	driverFlag := generic.DriverFlags{
		Options: make(map[string]*generic.Flag),
	}
	driverFlag.Options["node-count"] = &generic.Flag{
		Type:  generic.IntType,
		Usage: "The node number for your cluster to update",
	}
	driverFlag.Options["master-version"] = &generic.Flag{
		Type:  generic.StringType,
		Usage: "The kubernetes master version to update",
	}
	driverFlag.Options["node-version"] = &generic.Flag{
		Type:  generic.StringType,
		Usage: "The kubernetes node version to update",
	}
	return &driverFlag, nil
}

// SetDriverOptions implements driver interface
func (d *Driver) SetDriverOptions(driverOptions *generic.DriverOptions) error {
	d.Name = getValueFromDriverOptions(driverOptions, generic.StringType, "name").(string)
	d.ProjectID = getValueFromDriverOptions(driverOptions, generic.StringType, "projectId").(string)
	d.Zone = getValueFromDriverOptions(driverOptions, generic.StringType, "zone").(string)
	d.NodePoolID = getValueFromDriverOptions(driverOptions, generic.StringType, "nodePool").(string)
	d.ClusterIpv4Cidr = getValueFromDriverOptions(driverOptions, generic.StringType, "cluster-ipv4-cidr", "clusterIpv4Cidr").(string)
	d.Description = getValueFromDriverOptions(driverOptions, generic.StringType, "description").(string)
	d.MasterVersion = getValueFromDriverOptions(driverOptions, generic.StringType, "master-version", "masterVersion").(string)
	d.NodeVersion = getValueFromDriverOptions(driverOptions, generic.StringType, "node-version", "nodeVersion").(string)
	d.NodeConfig.DiskSizeGb = getValueFromDriverOptions(driverOptions, generic.IntType, "disk-size-gb", "diskSizeGb").(int64)
	d.NodeConfig.MachineType = getValueFromDriverOptions(driverOptions, generic.StringType, "machine-type", "machineType").(string)
	d.CredentialPath = getValueFromDriverOptions(driverOptions, generic.StringType, "gke-credential-path", "credentialPath").(string)
	d.EnableAlphaFeature = getValueFromDriverOptions(driverOptions, generic.BoolType, "enable-alpha-feature", "enableAlphaFeature").(bool)
	d.NodeCount = getValueFromDriverOptions(driverOptions, generic.IntType, "node-count", "nodeCount").(int64)
	labelValues := getValueFromDriverOptions(driverOptions, generic.StringSliceType, "labels").(*generic.StringSlice)
	for _, part := range labelValues.Value {
		kv := strings.Split(part, "=")
		if len(kv) == 2 {
			d.NodeConfig.Labels[kv[0]] = kv[1]
		}
	}
	// updateConfig
	return d.validate()
}

func getValueFromDriverOptions(driverOptions *generic.DriverOptions, optionType string, keys ...string) interface{} {
	switch optionType {
	case generic.IntType:
		for _, key := range keys {
			if value, ok := driverOptions.IntOptions[key]; ok {
				return value
			}
		}
		return int64(0)
	case generic.StringType:
		for _, key := range keys {
			if value, ok := driverOptions.StringOptions[key]; ok {
				return value
			}
		}
		return ""
	case generic.BoolType:
		for _, key := range keys {
			if value, ok := driverOptions.BoolOptions[key]; ok {
				return value
			}
		}
		return false
	case generic.StringSliceType:
		for _, key := range keys {
			if value, ok := driverOptions.StringSliceOptions[key]; ok {
				return value
			}
		}
		return &generic.StringSlice{}
	}
	return nil
}

func (d *Driver) validate() error {
	if d.ProjectID == "" || d.Zone == "" || d.Name == "" {
		logrus.Errorf("ProjectID or Zone or Name is required")
		return fmt.Errorf("projectID or zone or name is required")
	}
	if d.NodeCount == 0 {
		d.NodeCount = defaultNodeCount
	}
	return nil
}

// Create implements driver interface
func (d *Driver) Create() error {
	svc, err := d.getServiceClient()
	if err != nil {
		return err
	}
	operation, err := svc.Projects.Zones.Clusters.Create(d.ProjectID, d.Zone, d.generateClusterCreateRequest()).Context(context.Background()).Do()
	if err != nil {
		return err
	}
	logrus.Debugf("Cluster %s create is called for project %s and zone %s. Status Code %v", d.Name, d.ProjectID, d.Zone, operation.HTTPStatusCode)
	return d.waitCluster(svc)
}

// Update implements driver interface
func (d *Driver) Update() error {
	svc, err := d.getServiceClient()
	if err != nil {
		return err
	}
	logrus.Debugf("Updating config. MasterVersion: %s, NodeVersion: %s, NodeCount: %v", d.MasterVersion, d.NodeVersion, d.NodeCount)
	if d.NodePoolID == "" {
		cluster, err := svc.Projects.Zones.Clusters.Get(d.ProjectID, d.Zone, d.Name).Context(context.Background()).Do()
		if err != nil {
			return err
		}
		d.NodePoolID = cluster.NodePools[0].Name
	}

	if d.MasterVersion != "" {
		logrus.Infof("Updating master to %v", d.MasterVersion)
		operation, err := svc.Projects.Zones.Clusters.Update(d.ProjectID, d.Zone, d.Name, &raw.UpdateClusterRequest{
			Update: &raw.ClusterUpdate{
				DesiredMasterVersion: d.MasterVersion,
			},
		}).Context(context.Background()).Do()
		if err != nil {
			return err
		}
		logrus.Debugf("Cluster %s update is called for project %s and zone %s. Status Code %v", d.Name, d.ProjectID, d.Zone, operation.HTTPStatusCode)
		if err := d.waitCluster(svc); err != nil {
			return err
		}
	}

	if d.NodeVersion != "" {
		logrus.Infof("Updating node version to %v", d.NodeVersion)
		operation, err := svc.Projects.Zones.Clusters.NodePools.Update(d.ProjectID, d.Zone, d.Name, d.NodePoolID, &raw.UpdateNodePoolRequest{
			NodeVersion: d.NodeVersion,
		}).Context(context.Background()).Do()
		if err != nil {
			return err
		}
		logrus.Debugf("Nodepool %s update is called for project %s, zone %s and cluster %s. Status Code %v", d.NodePoolID, d.ProjectID, d.Zone, d.Name, operation.HTTPStatusCode)
		if err := d.waitNodePool(svc); err != nil {
			return err
		}
	}

	if d.NodeCount != 0 {
		logrus.Infof("Updating node number to %v", d.NodeCount)
		operation, err := svc.Projects.Zones.Clusters.NodePools.SetSize(d.ProjectID, d.Zone, d.Name, d.NodePoolID, &raw.SetNodePoolSizeRequest{
			NodeCount: d.NodeCount,
		}).Context(context.Background()).Do()
		if err != nil {
			return err
		}
		logrus.Debugf("Nodepool %s setSize is called for project %s, zone %s and cluster %s. Status Code %v", d.NodePoolID, d.ProjectID, d.Zone, d.Name, operation.HTTPStatusCode)
	}
	return nil
}

func (d *Driver) generateClusterCreateRequest() *raw.CreateClusterRequest {
	request := raw.CreateClusterRequest{
		Cluster: &raw.Cluster{},
	}
	request.Cluster.Name = d.Name
	request.Cluster.Zone = d.Zone
	request.Cluster.InitialClusterVersion = d.MasterVersion
	request.Cluster.InitialNodeCount = d.NodeCount
	request.Cluster.ClusterIpv4Cidr = d.ClusterIpv4Cidr
	request.Cluster.Description = d.Description
	request.Cluster.EnableKubernetesAlpha = d.EnableAlphaFeature
	request.Cluster.MasterAuth = &raw.MasterAuth{
		Username: "admin",
	}
	request.Cluster.NodeConfig = d.NodeConfig
	return &request
}

// Get implements driver interface
func (d *Driver) Get() (*generic.ClusterInfo, error) {
	svc, err := d.getServiceClient()
	if err != nil {
		return nil, err
	}
	cluster, err := svc.Projects.Zones.Clusters.Get(d.ProjectID, d.Zone, d.Name).Context(context.Background()).Do()
	if err != nil {
		return nil, err
	}
	info := &generic.ClusterInfo{
		Metadata: map[string]string{},
	}
	info.Endpoint = cluster.Endpoint
	info.Version = cluster.CurrentMasterVersion
	info.Username = cluster.MasterAuth.Username
	info.Password = cluster.MasterAuth.Password
	info.RootCaCertificate = cluster.MasterAuth.ClusterCaCertificate
	info.ClientCertificate = cluster.MasterAuth.ClientCertificate
	info.ClientKey = cluster.MasterAuth.ClientKey
	info.NodeCount = cluster.CurrentNodeCount

	info.Metadata["projectId"] = d.ProjectID
	info.Metadata["zone"] = d.Zone
	info.Metadata["gke-credential-path"] = os.Getenv(defaultCredentialEnv)
	info.Metadata["nodePool"] = cluster.NodePools[0].Name
	serviceAccountToken, err := generateServiceAccountTokenForGke(cluster)
	if err != nil {
		return nil, err
	}
	info.ServiceAccountToken = serviceAccountToken

	return info, nil
}

// Remove implements driver interface
func (d *Driver) Remove() error {
	svc, err := d.getServiceClient()
	if err != nil {
		return err
	}
	logrus.Debugf("Removing cluster %v from project %v, zone %v", d.Name, d.ProjectID, d.Zone)
	operation, err := svc.Projects.Zones.Clusters.Delete(d.ProjectID, d.Zone, d.Name).Context(context.Background()).Do()
	if err != nil && !strings.Contains(err.Error(), "notFound") {
		return err
	} else if err == nil {
		logrus.Debugf("Cluster %v delete is called. Status Code %v", d.Name, operation.HTTPStatusCode)
	} else {
		logrus.Debugf("Cluster %s doesn't exist", d.Name)
	}
	return nil
}

func (d *Driver) getServiceClient() (*raw.Service, error) {
	if d.CredentialPath != "" {
		os.Setenv(defaultCredentialEnv, d.CredentialPath)
	}
	client, err := google.DefaultClient(context.Background(), raw.CloudPlatformScope)
	if err != nil {
		return nil, err
	}
	service, err := raw.New(client)
	if err != nil {
		return nil, err
	}
	return service, nil
}

func generateServiceAccountTokenForGke(cluster *raw.Cluster) (string, error) {
	capem, err := base64.StdEncoding.DecodeString(cluster.MasterAuth.ClusterCaCertificate)
	if err != nil {
		return "", err
	}
	host := cluster.Endpoint
	if !strings.HasPrefix(host, "https://") {
		host = fmt.Sprintf("https://%s", host)
	}
	// in here we have to use http basic auth otherwise we can't get the permission to create cluster role
	config := &rest.Config{
		Host: host,
		TLSClientConfig: rest.TLSClientConfig{
			CAData: capem,
		},
		Username: cluster.MasterAuth.Username,
		Password: cluster.MasterAuth.Password,
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", err
	}

	return generic.GenerateServiceAccountToken(clientset)
}

func (d *Driver) waitCluster(svc *raw.Service) error {
	lastMsg := ""
	for {
		cluster, err := svc.Projects.Zones.Clusters.Get(d.ProjectID, d.Zone, d.Name).Context(context.TODO()).Do()
		if err != nil {
			return err
		}
		if cluster.Status == runningStatus {
			logrus.Infof("Cluster %v is running", d.Name)
			return nil
		}
		if cluster.Status != lastMsg {
			logrus.Infof("%v cluster %v......", strings.ToLower(cluster.Status), d.Name)
			lastMsg = cluster.Status
		}
		time.Sleep(time.Second * 5)
	}
}

func (d *Driver) waitNodePool(svc *raw.Service) error {
	lastMsg := ""
	for {
		nodepool, err := svc.Projects.Zones.Clusters.NodePools.Get(d.ProjectID, d.Zone, d.Name, d.NodePoolID).Context(context.TODO()).Do()
		if err != nil {
			return err
		}
		if nodepool.Status == runningStatus {
			logrus.Infof("Nodepool %v is running", d.Name)
			return nil
		}
		if nodepool.Status != lastMsg {
			logrus.Infof("%v nodepool %v......", strings.ToLower(nodepool.Status), d.NodePoolID)
			lastMsg = nodepool.Status
		}
		time.Sleep(time.Second * 5)
	}
}
