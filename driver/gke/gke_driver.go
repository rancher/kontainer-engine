package gke

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	generic "github.com/rancher/kontainer-engine/driver"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	raw "google.golang.org/api/container/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	runningStatus        = "RUNNING"
	defaultCredentialEnv = "GOOGLE_APPLICATION_CREDENTIALS"
)

// Driver defines the struct of gke driver
type Driver struct {
	sync.Mutex
}

type state struct {
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
	// The content of the credential
	CredentialContent string
	// Enable alpha feature
	EnableAlphaFeature bool
	// Configuration for the HTTP (L7) load balancing controller addon
	HTTPLoadBalancing bool
	// Configuration for the horizontal pod autoscaling feature, which increases or decreases the number of replica pods a replication controller has based on the resource usage of the existing pods
	HorizontalPodAutoscaling bool
	// Configuration for the Kubernetes Dashboard
	KubernetesDashboard bool
	// Configuration for NetworkPolicy
	NetworkPolicyConfig bool
	// The list of Google Compute Engine locations in which the cluster's nodes should be located
	Locations []string
	// Network
	Network string
	// Sub Network
	SubNetwork string
	// Configuration for LegacyAbac
	LegacyAbac bool
	// NodePool id
	NodePoolID string
	// cluster info
	ClusterInfo generic.ClusterInfo
}

// NewDriver creates a gke Driver
func NewDriver() *Driver {
	return &Driver{}
}

// GetDriverCreateOptions implements driver interface
func (d *Driver) GetDriverCreateOptions() (*generic.DriverFlags, error) {
	driverFlag := generic.DriverFlags{
		Options: make(map[string]*generic.Flag),
	}
	driverFlag.Options["project-id"] = &generic.Flag{
		Type:  generic.StringType,
		Usage: "the ID of your project to use when creating a cluster",
	}
	driverFlag.Options["zone"] = &generic.Flag{
		Type:  generic.StringType,
		Usage: "The zone to launch the cluster",
		Value: "us-central1-a",
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
		Value: "3",
	}
	driverFlag.Options["disk-size-gb"] = &generic.Flag{
		Type:  generic.IntType,
		Usage: "Size of the disk attached to each node",
		Value: "100",
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
		Usage: "The node number for your cluster to update. 0 means no updates",
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
func getStateFromOpts(driverOptions *generic.DriverOptions) (state, error) {
	d := state{
		NodeConfig: &raw.NodeConfig{
			Labels: map[string]string{},
		},
		ClusterInfo: generic.ClusterInfo{
			Metadata: map[string]string{},
		},
	}
	d.Name = getValueFromDriverOptions(driverOptions, generic.StringType, "name").(string)
	d.ProjectID = getValueFromDriverOptions(driverOptions, generic.StringType, "project-id", "projectId").(string)
	d.Zone = getValueFromDriverOptions(driverOptions, generic.StringType, "zone").(string)
	d.NodePoolID = getValueFromDriverOptions(driverOptions, generic.StringType, "nodePool").(string)
	d.ClusterIpv4Cidr = getValueFromDriverOptions(driverOptions, generic.StringType, "cluster-ipv4-cidr", "clusterIpv4Cidr").(string)
	d.Description = getValueFromDriverOptions(driverOptions, generic.StringType, "description").(string)
	d.MasterVersion = getValueFromDriverOptions(driverOptions, generic.StringType, "master-version", "masterVersion").(string)
	d.NodeVersion = getValueFromDriverOptions(driverOptions, generic.StringType, "node-version", "nodeVersion").(string)
	d.NodeConfig.DiskSizeGb = getValueFromDriverOptions(driverOptions, generic.IntType, "disk-size-gb", "diskSizeGb").(int64)
	d.NodeConfig.MachineType = getValueFromDriverOptions(driverOptions, generic.StringType, "machine-type", "machineType").(string)
	d.CredentialPath = getValueFromDriverOptions(driverOptions, generic.StringType, "gke-credential-path").(string)
	d.CredentialContent = getValueFromDriverOptions(driverOptions, generic.StringType, "credential").(string)
	d.EnableAlphaFeature = getValueFromDriverOptions(driverOptions, generic.BoolType, "enable-alpha-feature", "enableAlphaFeature").(bool)
	d.HorizontalPodAutoscaling = getValueFromDriverOptions(driverOptions, generic.BoolType, "horizontalPodAutoscaling").(bool)
	d.HTTPLoadBalancing = getValueFromDriverOptions(driverOptions, generic.BoolType, "httpLoadBalancing").(bool)
	d.KubernetesDashboard = getValueFromDriverOptions(driverOptions, generic.BoolType, "kubernetesDashboard").(bool)
	d.NetworkPolicyConfig = getValueFromDriverOptions(driverOptions, generic.BoolType, "networkPolicyConfig").(bool)
	d.NodeConfig.ImageType = getValueFromDriverOptions(driverOptions, generic.StringType, "imageType").(string)
	d.Network = getValueFromDriverOptions(driverOptions, generic.StringType, "network").(string)
	d.SubNetwork = getValueFromDriverOptions(driverOptions, generic.StringType, "subNetwork").(string)
	d.LegacyAbac = getValueFromDriverOptions(driverOptions, generic.BoolType, "legacyAbac").(bool)
	d.Locations = []string{}
	locations := getValueFromDriverOptions(driverOptions, generic.StringSliceType, "locations").(*generic.StringSlice)
	for _, location := range locations.Value {
		d.Locations = append(d.Locations, location)
	}

	d.NodeCount = getValueFromDriverOptions(driverOptions, generic.IntType, "node-count", "nodeCount").(int64)
	labelValues := getValueFromDriverOptions(driverOptions, generic.StringSliceType, "labels").(*generic.StringSlice)
	for _, part := range labelValues.Value {
		kv := strings.Split(part, "=")
		if len(kv) == 2 {
			d.NodeConfig.Labels[kv[0]] = kv[1]
		}
	}
	return d, d.validate()
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

func (s *state) validate() error {
	if s.ProjectID == "" {
		return fmt.Errorf("project ID is required")
	} else if s.Zone == "" {
		return fmt.Errorf("zone is required")
	} else if s.Name == "" {
		return fmt.Errorf("cluster name is required")
	}
	return nil
}

// Create implements driver interface
func (d *Driver) Create(opts *generic.DriverOptions) (*generic.ClusterInfo, error) {
	state, err := getStateFromOpts(opts)
	if err != nil {
		return nil, err
	}

	svc, err := d.getServiceClient(state)
	if err != nil {
		return nil, err
	}

	operation, err := svc.Projects.Zones.Clusters.Create(state.ProjectID, state.Zone, d.generateClusterCreateRequest(state)).Context(context.Background()).Do()
	if err != nil && !strings.Contains(err.Error(), "alreadyExists") {
		return nil, err
	}

	if err == nil {
		logrus.Debugf("Cluster %s create is called for project %s and zone %s. Status Code %v", state.Name, state.ProjectID, state.Zone, operation.HTTPStatusCode)
	}

	if err := d.waitCluster(svc, &state); err != nil {
		return nil, err
	}

	info := &generic.ClusterInfo{}
	return info, storeState(info, state)
}

func storeState(info *generic.ClusterInfo, state state) error {
	bytes, err := json.Marshal(state)
	if err != nil {
		return err
	}
	if info.Metadata == nil {
		info.Metadata = map[string]string{}
	}
	info.Metadata["state"] = string(bytes)
	info.Metadata["project-id"] = state.ProjectID
	info.Metadata["zone"] = state.Zone
	return nil
}

func getState(info *generic.ClusterInfo) (state, error) {
	state := state{}
	// ignore error
	err := json.Unmarshal([]byte(info.Metadata["state"]), &state)
	return state, err
}

// Update implements driver interface
func (d *Driver) Update(info *generic.ClusterInfo, opts *generic.DriverOptions) (*generic.ClusterInfo, error) {
	state, err := getState(info)
	if err != nil {
		return nil, err
	}

	newState, err := getStateFromOpts(opts)
	if err != nil {
		return nil, err
	}

	svc, err := d.getServiceClient(state)
	if err != nil {
		return nil, err
	}

	if state.NodePoolID == "" {
		cluster, err := svc.Projects.Zones.Clusters.Get(state.ProjectID, state.Zone, state.Name).Context(context.Background()).Do()
		if err != nil {
			return nil, err
		}
		state.NodePoolID = cluster.NodePools[0].Name
	}

	logrus.Debugf("Updating config. MasterVersion: %s, NodeVersion: %s, NodeCount: %v", state.MasterVersion, state.NodeVersion, state.NodeCount)

	if newState.MasterVersion != "" {
		logrus.Infof("Updating master to %v", newState.MasterVersion)
		operation, err := svc.Projects.Zones.Clusters.Update(state.ProjectID, state.Zone, state.Name, &raw.UpdateClusterRequest{
			Update: &raw.ClusterUpdate{
				DesiredMasterVersion: newState.MasterVersion,
			},
		}).Context(context.Background()).Do()
		if err != nil {
			return nil, err
		}
		logrus.Debugf("Cluster %s update is called for project %s and zone %s. Status Code %v", state.Name, state.ProjectID, state.Zone, operation.HTTPStatusCode)
		if err := d.waitCluster(svc, &state); err != nil {
			return nil, err
		}
		state.MasterVersion = newState.MasterVersion
	}

	if newState.NodeVersion != "" {
		logrus.Infof("Updating node version to %v", newState.NodeVersion)
		operation, err := svc.Projects.Zones.Clusters.NodePools.Update(state.ProjectID, state.Zone, state.Name, state.NodePoolID, &raw.UpdateNodePoolRequest{
			NodeVersion: state.NodeVersion,
		}).Context(context.Background()).Do()
		if err != nil {
			return nil, err
		}
		logrus.Debugf("Nodepool %s update is called for project %s, zone %s and cluster %s. Status Code %v", state.NodePoolID, state.ProjectID, state.Zone, state.Name, operation.HTTPStatusCode)
		if err := d.waitNodePool(svc, &state); err != nil {
			return nil, err
		}
		state.NodeVersion = newState.NodeVersion
	}

	if newState.NodeCount != 0 {
		logrus.Infof("Updating node number to %v", newState.NodeCount)
		operation, err := svc.Projects.Zones.Clusters.NodePools.SetSize(state.ProjectID, state.Zone, state.Name, state.NodePoolID, &raw.SetNodePoolSizeRequest{
			NodeCount: newState.NodeCount,
		}).Context(context.Background()).Do()
		if err != nil {
			return nil, err
		}
		logrus.Debugf("Nodepool %s setSize is called for project %s, zone %s and cluster %s. Status Code %v", state.NodePoolID, state.ProjectID, state.Zone, state.Name, operation.HTTPStatusCode)
		if err := d.waitCluster(svc, &state); err != nil {
			return nil, err
		}
	}

	return info, storeState(info, state)
}

func (d *Driver) generateClusterCreateRequest(state state) *raw.CreateClusterRequest {
	request := raw.CreateClusterRequest{
		Cluster: &raw.Cluster{},
	}
	request.Cluster.Name = state.Name
	request.Cluster.Zone = state.Zone
	request.Cluster.InitialClusterVersion = state.MasterVersion
	request.Cluster.InitialNodeCount = state.NodeCount
	request.Cluster.ClusterIpv4Cidr = state.ClusterIpv4Cidr
	request.Cluster.Description = state.Description
	request.Cluster.EnableKubernetesAlpha = state.EnableAlphaFeature
	request.Cluster.AddonsConfig = &raw.AddonsConfig{
		HttpLoadBalancing:        &raw.HttpLoadBalancing{Disabled: !state.HTTPLoadBalancing},
		HorizontalPodAutoscaling: &raw.HorizontalPodAutoscaling{Disabled: !state.HorizontalPodAutoscaling},
		KubernetesDashboard:      &raw.KubernetesDashboard{Disabled: !state.KubernetesDashboard},
		NetworkPolicyConfig:      &raw.NetworkPolicyConfig{Disabled: !state.NetworkPolicyConfig},
	}
	request.Cluster.Network = state.Network
	request.Cluster.Subnetwork = state.SubNetwork
	request.Cluster.LegacyAbac = &raw.LegacyAbac{
		Enabled: state.LegacyAbac,
	}
	request.Cluster.MasterAuth = &raw.MasterAuth{
		Username: "admin",
	}
	request.Cluster.NodeConfig = state.NodeConfig
	return &request
}

func (d *Driver) PostCheck(info *generic.ClusterInfo) (*generic.ClusterInfo, error) {
	state, err := getState(info)
	if err != nil {
		return nil, err
	}

	svc, err := d.getServiceClient(state)
	if err != nil {
		return nil, err
	}

	if err := d.waitCluster(svc, &state); err != nil {
		return nil, err
	}

	cluster, err := svc.Projects.Zones.Clusters.Get(state.ProjectID, state.Zone, state.Name).Context(context.Background()).Do()
	if err != nil {
		return nil, err
	}

	info.Endpoint = cluster.Endpoint
	info.Version = cluster.CurrentMasterVersion
	info.Username = cluster.MasterAuth.Username
	info.Password = cluster.MasterAuth.Password
	info.RootCaCertificate = cluster.MasterAuth.ClusterCaCertificate
	info.ClientCertificate = cluster.MasterAuth.ClientCertificate
	info.ClientKey = cluster.MasterAuth.ClientKey
	info.NodeCount = cluster.CurrentNodeCount
	info.Metadata["nodePool"] = cluster.NodePools[0].Name
	serviceAccountToken, err := generateServiceAccountTokenForGke(cluster)
	if err != nil {
		return nil, err
	}
	info.ServiceAccountToken = serviceAccountToken
	return info, nil
}

// Remove implements driver interface
func (d *Driver) Remove(info *generic.ClusterInfo) error {
	state, err := getState(info)
	if err != nil {
		return err
	}

	svc, err := d.getServiceClient(state)
	if err != nil {
		return err
	}

	logrus.Debugf("Removing cluster %v from project %v, zone %v", state.Name, state.ProjectID, state.Zone)
	operation, err := svc.Projects.Zones.Clusters.Delete(state.ProjectID, state.Zone, state.Name).Context(context.Background()).Do()
	if err != nil && !strings.Contains(err.Error(), "notFound") {
		return err
	} else if err == nil {
		logrus.Debugf("Cluster %v delete is called. Status Code %v", state.Name, operation.HTTPStatusCode)
	} else {
		logrus.Debugf("Cluster %s doesn't exist", state.Name)
	}
	return nil
}

func (d *Driver) getServiceClient(state state) (*raw.Service, error) {
	// The google SDK has no sane way to pass in a TokenSource give all the different types (user, service account, etc)
	// So we actually set an environment variable and then unset it
	d.Lock()
	locked := true
	setEnv := false
	cleanup := func() {
		if setEnv {
			os.Unsetenv(defaultCredentialEnv)
		}

		if locked {
			d.Unlock()
			locked = false
		}
	}
	defer cleanup()

	if state.CredentialPath != "" {
		setEnv = true
		os.Setenv(defaultCredentialEnv, state.CredentialPath)
	}
	if state.CredentialContent != "" {
		file, err := ioutil.TempFile("", "credential-file")
		if err != nil {
			return nil, err
		}
		defer os.Remove(file.Name())
		defer file.Close()

		if _, err := io.Copy(file, strings.NewReader(state.CredentialContent)); err != nil {
			return nil, err
		}

		setEnv = true
		os.Setenv(defaultCredentialEnv, file.Name())
	}

	ts, err := google.DefaultTokenSource(context.Background(), raw.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	// Unlocks
	cleanup()

	client := oauth2.NewClient(context.Background(), ts)
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

func (d *Driver) waitCluster(svc *raw.Service, state *state) error {
	lastMsg := ""
	for {
		cluster, err := svc.Projects.Zones.Clusters.Get(state.ProjectID, state.Zone, state.Name).Context(context.TODO()).Do()
		if err != nil {
			return err
		}
		if cluster.Status == runningStatus {
			logrus.Infof("Cluster %v is running", state.Name)
			return nil
		}
		if cluster.Status != lastMsg {
			logrus.Infof("%v cluster %v......", strings.ToLower(cluster.Status), state.Name)
			lastMsg = cluster.Status
		}
		time.Sleep(time.Second * 5)
	}
}

func (d *Driver) waitNodePool(svc *raw.Service, state *state) error {
	lastMsg := ""
	for {
		nodepool, err := svc.Projects.Zones.Clusters.NodePools.Get(state.ProjectID, state.Zone, state.Name, state.NodePoolID).Context(context.TODO()).Do()
		if err != nil {
			return err
		}
		if nodepool.Status == runningStatus {
			logrus.Infof("Nodepool %v is running", state.Name)
			return nil
		}
		if nodepool.Status != lastMsg {
			logrus.Infof("%v nodepool %v......", strings.ToLower(nodepool.Status), state.NodePoolID)
			lastMsg = nodepool.Status
		}
		time.Sleep(time.Second * 5)
	}
}
