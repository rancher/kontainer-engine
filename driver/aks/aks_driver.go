package aks

import (
	"strings"

	generic "github.com/rancher/kontainer-engine/driver"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/utils"
	"github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2017-08-31/containerservice"
	"context"
	"github.com/Azure/go-autorest/autorest/to"
	"io/ioutil"
	"fmt"
	"time"
	"github.com/sirupsen/logrus"
	"encoding/base64"
	"gopkg.in/yaml.v2"
	"encoding/json"
)

type Driver struct{}

type state struct {
	// Subscription credentials which uniquely identify Microsoft Azure subscription. The subscription ID forms part of the URI for every service call.
	SubscriptionID string
	// The name of the resource group.
	ResourceGroup string
	// The name of the managed cluster resource.
	Name string
	// Resource location
	Location string
	// Resource tags
	Tag map[string]string
	// Number of agents (VMs) to host docker containers. Allowed values must be in the range of 1 to 100 (inclusive). The default value is 1.
	Count int64
	// DNS prefix to be used to create the FQDN for the agent pool.
	AgentDnsPrefix string
	// FDQN for the agent pool
	AgentPoolName string
	// OS Disk Size in GB to be used to specify the disk size for every machine in this master/agent pool. If you specify 0, it will apply the default osDisk size according to the vmSize specified.
	OsDiskSizeGB int64
	// Size of agent VMs
	AgentVMSize string
	// Version of Kubernetes specified when creating the managed cluster
	KubernetesVersion string
	// Path to the public key to use for SSH into cluster
	SSHPublicKeyPath string
	// Kubernetes Master DNS prefix (must be unique within Azure)
	MasterDNSPrefix string
	// Kubernetes admin username
	AdminUsername string
	// Different Base URL if required, usually needed for testing purposes
	BaseUrl string
	// Azure Client ID to use
	ClientID string
	// Secret associated with the Client ID
	ClientSecret string

	// Cluster info
	ClusterInfo generic.ClusterInfo
}

func NewDriver() *Driver {
	return &Driver{}
}

// GetDriverCreateOptions implements driver interface
func (d *Driver) GetDriverCreateOptions() (*generic.DriverFlags, error) {
	driverFlag := generic.DriverFlags{
		Options: make(map[string]*generic.Flag),
	}
	driverFlag.Options["subscription-id"] = &generic.Flag{
		Type:  generic.StringType,
		Usage: "Subscription credentials which uniquely identify Microsoft Azure subscription",
	}
	driverFlag.Options["resource-group"] = &generic.Flag{
		Type:  generic.StringType,
		Usage: "The name of the resource group",
	}
	driverFlag.Options["location"] = &generic.Flag{
		Type:  generic.StringType,
		Usage: "Resource location",
		Value: "eastus",
	}
	driverFlag.Options["tags"] = &generic.Flag{
		Type:  generic.StringSliceType,
		Usage: "Resource tags. For example, foo=bar",
	}
	driverFlag.Options["node-count"] = &generic.Flag{
		Type:  generic.StringType,
		Usage: "Number of agents (VMs) to host docker containers. Allowed values must be in the range of 1 to 100 (inclusive)",
		Value: "1",
	}
	driverFlag.Options["node-dns-prefix"] = &generic.Flag{
		Type:  generic.StringType,
		Usage: "DNS prefix to be used to create the FQDN for the agent pool",
	}
	driverFlag.Options["node-pool-name"] = &generic.Flag{
		Type:  generic.StringType,
		Usage: "Name for the agent pool",
		Value: "agentpool0",
	}
	driverFlag.Options["os-disk-size"] = &generic.Flag{
		Type:  generic.StringType,
		Usage: "OS Disk Size in GB to be used to specify the disk size for every machine in this master/agent pool. If you specify 0, it will apply the default osDisk size according to the vmSize specified.",
	}
	driverFlag.Options["node-vm-size"] = &generic.Flag{
		Type:  generic.StringType,
		Usage: "Size of agent VMs",
		Value: "Standard_D1_v2",
	}
	driverFlag.Options["kubernetes-version"] = &generic.Flag{
		Type:  generic.StringType,
		Usage: "Version of Kubernetes specified when creating the managed cluster",
	}
	driverFlag.Options["public-key"] = &generic.Flag{
		Type:  generic.StringType,
		Usage: "SSH public key to use for the cluster",
	}
	driverFlag.Options["master-dns-prefix"] = &generic.Flag{
		Type:  generic.StringType,
		Usage: "DNS prefix to use for the master",
	}
	driverFlag.Options["admin-username"] = &generic.Flag{
		Type:  generic.StringType,
		Usage: "Admin username to use for the cluster",
		Value: "azureuser",
	}
	driverFlag.Options["base-url"] = &generic.Flag{
		Type:  generic.StringType,
		Usage: "Different base API url to use",
		Value: containerservice.DefaultBaseURI,
	}
	driverFlag.Options["client-id"] = &generic.Flag{
		Type:  generic.StringType,
		Usage: "Azure client id to use",
	}
	driverFlag.Options["client-secret"] = &generic.Flag{
		Type:  generic.StringType,
		Usage: "Client secret associated with the client-id",
	}

	return &driverFlag, nil
}

// GetDriverUpdateOptions implements driver interface
func (d *Driver) GetDriverUpdateOptions() (*generic.DriverFlags, error) {
	driverFlag := generic.DriverFlags{
		Options: make(map[string]*generic.Flag),
	}
	driverFlag.Options["node-count"] = &generic.Flag{
		Type:  generic.StringType,
		Usage: "Number of agents (VMs) to host docker containers. Allowed values must be in the range of 1 to 100 (inclusive)",
		Value: "1",
	}
	driverFlag.Options["kubernetes-version"] = &generic.Flag{
		Type:  generic.StringType,
		Usage: "Version of Kubernetes specified when creating the managed cluster",
	}
	return &driverFlag, nil
}

// SetDriverOptions implements driver interface
func getStateFromOptions(driverOptions *generic.DriverOptions) (state, error) {
	state := state{}
	state.Name = generic.GetValueFromDriverOptions(driverOptions, generic.StringType, "name").(string)
	state.AgentDnsPrefix = generic.GetValueFromDriverOptions(driverOptions, generic.StringType, "node-dns-prefix").(string)
	state.AgentVMSize = generic.GetValueFromDriverOptions(driverOptions, generic.StringType, "node-vm-size").(string)
	state.Count = generic.GetValueFromDriverOptions(driverOptions, generic.IntType, "node-count").(int64)
	state.KubernetesVersion = generic.GetValueFromDriverOptions(driverOptions, generic.StringType, "kubernetes-version").(string)
	state.Location = generic.GetValueFromDriverOptions(driverOptions, generic.StringType, "location").(string)
	state.OsDiskSizeGB = generic.GetValueFromDriverOptions(driverOptions, generic.IntType, "os-disk-size").(int64)
	state.SubscriptionID = generic.GetValueFromDriverOptions(driverOptions, generic.StringType, "subscription-id").(string)
	state.ResourceGroup = generic.GetValueFromDriverOptions(driverOptions, generic.StringType, "resource-group").(string)
	state.AgentPoolName = generic.GetValueFromDriverOptions(driverOptions, generic.StringType, "node-pool-name").(string)
	state.MasterDNSPrefix = generic.GetValueFromDriverOptions(driverOptions, generic.StringType, "master-dns-prefix").(string)
	state.SSHPublicKeyPath = generic.GetValueFromDriverOptions(driverOptions, generic.StringType, "public-key").(string)
	state.AdminUsername = generic.GetValueFromDriverOptions(driverOptions, generic.StringType, "admin-username").(string)
	state.BaseUrl = generic.GetValueFromDriverOptions(driverOptions, generic.StringType, "base-url").(string)
	state.ClientID = generic.GetValueFromDriverOptions(driverOptions, generic.StringType, "client-id").(string)
	state.ClientSecret = generic.GetValueFromDriverOptions(driverOptions, generic.StringType, "client-secret").(string)
	tagValues := generic.GetValueFromDriverOptions(driverOptions, generic.StringSliceType).(*generic.StringSlice)
	for _, part := range tagValues.Value {
		kv := strings.Split(part, "=")
		if len(kv) == 2 {
			state.Tag[kv[0]] = kv[1]
		}
	}
	return state, state.validate()
}

func (state *state) validate() error {
	if state.Name == "" {
		return fmt.Errorf("cluster name is required")
	}

	if state.ResourceGroup == "" {
		return fmt.Errorf("resource group is required")
	}

	if state.SSHPublicKeyPath == "" {
		return fmt.Errorf("path to ssh public key is required")
	}

	if state.ClientID == "" {
		return fmt.Errorf("client id is required")
	}

	if state.ClientSecret == "" {
		return fmt.Errorf("client secret is required")
	}

	if state.SubscriptionID == "" {
		return fmt.Errorf("subscription id is required")
	}

	return nil
}

func safeSlice(toSlice string, index int) string {
	size := len(toSlice)

	if index >= size {
		index = size - 1
	}

	return toSlice[:index]
}

func (state *state) getDefaultDNSPrefix() string {
	namePart := safeSlice(state.Name, 10)
	groupPart := safeSlice(state.ResourceGroup, 16)
	subscriptionPart := safeSlice(state.SubscriptionID, 6)

	return fmt.Sprintf("%v-%v-%v", namePart, groupPart, subscriptionPart)
}

func newAzureClient(state state) (*containerservice.ManagedClustersClient, error) {
	authorizer, err := utils.GetAuthorizer(azure.PublicCloud)

	if err != nil {
		return nil, err
	}

	client := containerservice.NewManagedClustersClientWithBaseURI(state.BaseUrl, state.SubscriptionID)
	client.Authorizer = authorizer

	return &client, nil
}

const failedStatus = "Failed"
const succeededStatus = "Succeeded"
const creatingStatus = "Creating"

const pollInterval = 30

// Create implements driver interface
func (d *Driver) Create(options *generic.DriverOptions) (*generic.ClusterInfo, error) {
	driverState, err := getStateFromOptions(options)

	if err != nil {
		return nil, err
	}

	client, err := newAzureClient(driverState)

	if err != nil {
		return nil, err
	}

	masterDNSPrefix := driverState.MasterDNSPrefix
	if masterDNSPrefix == "" {
		masterDNSPrefix = driverState.getDefaultDNSPrefix() + "-master"
	}

	agentDNSPrefix := driverState.AgentDnsPrefix
	if agentDNSPrefix == "" {
		agentDNSPrefix = driverState.getDefaultDNSPrefix() + "-agent"
	}

	publicKey, err := ioutil.ReadFile(driverState.SSHPublicKeyPath)

	if err != nil {
		return nil, err
	}

	publicKeyContents := string(publicKey)

	tags := make(map[string]*string)

	ctx := context.Background()
	_, err = client.CreateOrUpdate(ctx, driverState.ResourceGroup, driverState.Name, containerservice.ManagedCluster{
		Location: to.StringPtr(driverState.Location),
		Tags:     &tags,
		ManagedClusterProperties: &containerservice.ManagedClusterProperties{
			DNSPrefix: to.StringPtr(masterDNSPrefix),
			LinuxProfile: &containerservice.LinuxProfile{
				AdminUsername: to.StringPtr(driverState.AdminUsername),
				SSH: &containerservice.SSHConfiguration{
					PublicKeys: &[]containerservice.SSHPublicKey{
						{
							KeyData: to.StringPtr(publicKeyContents),
						},
					},
				},
			},
			AgentPoolProfiles: &[]containerservice.AgentPoolProfile{
				{
					DNSPrefix: to.StringPtr(agentDNSPrefix),
					Name:      to.StringPtr(driverState.AgentPoolName),
					VMSize:    containerservice.VMSizeTypes(driverState.AgentVMSize),
				},
			},
			ServicePrincipalProfile: &containerservice.ServicePrincipalProfile{
				ClientID: to.StringPtr(driverState.ClientID),
				Secret:   to.StringPtr(driverState.ClientSecret),
			},
		},
	})

	if err != nil {
		return nil, err
	}

	logrus.Info("Request submitted, waiting for cluster to finish creating")

	for {
		result, err := client.Get(ctx, driverState.ResourceGroup, driverState.Name)

		if err != nil {
			return nil, err
		}

		state := *result.ProvisioningState

		if state == failedStatus {
			return nil, fmt.Errorf("cluster create has completed with status of 'Failed'")
		}

		if state == succeededStatus {
			logrus.Info("Cluster provisioned successfully")
			info := &generic.ClusterInfo{}
			err := storeState(info, driverState)

			fmt.Println("********")
			fmt.Println(info.Metadata["state"])
			fmt.Println("********")

			return info, err
		}

		if state != creatingStatus {
			return nil, fmt.Errorf("unexpected state %v", state)
		}

		logrus.Infof("Cluster has not yet completed provisioning, waiting another %v seconds", pollInterval)

		time.Sleep(pollInterval * time.Second)
	}
}

func storeState(info *generic.ClusterInfo, state state) error {
	data, err := json.Marshal(state)

	if err != nil {
		return err
	}

	if info.Metadata == nil {
		info.Metadata = map[string]string{}
	}

	info.Metadata["state"] = string(data)
	info.Metadata["resource-group"] = state.ResourceGroup
	info.Metadata["location"] = state.Location

	return nil
}

func getState(info *generic.ClusterInfo) (state, error) {
	state := state{}

	err := json.Unmarshal([]byte(info.Metadata["state"]), &state)

	if err != nil {
		logrus.Errorf("Error encountered while marshalling state: %v", err)
	}

	return state, err
}

// Update implements driver interface
func (d *Driver) Update(info *generic.ClusterInfo, options *generic.DriverOptions) (*generic.ClusterInfo, error) {
	// todo: implement
	return nil, fmt.Errorf("not implemented")
}

// shouldn't have to reimplement this but kubernetes' model won't serialize correctly for some reason
type KubeConfig struct {
	APIVersion string    `yaml:"apiVersion"`
	Kind       string    `yaml:"kind"`
	Clusters   []Cluster `yaml:"clusters"`
	Contexts   []Context `yaml:"contexts"`
	Users      []User    `yaml:"users"`
}

type Cluster struct {
	Name        string      `yaml:"name"`
	ClusterInfo ClusterInfo `yaml:"cluster"`
}

type ClusterInfo struct {
	Server                   string `yaml:"server"`
	CertificateAuthorityData string `yaml:"certificate-authority-data"`
}

type Context struct {
	ContextInfo ContextInfo `yaml:"context"`
	Name        string      `yaml:"name"`
}

type ContextInfo struct {
	Cluster string `yaml:"cluster"`
	User    string `yaml:"user"`
}

type User struct {
	UserInfo UserInfo `yaml:"user"`
	Name     string   `yaml:"name"`
}

type UserInfo struct {
	ClientCertificateData string `yaml:"client-certificate-data"`
	ClientKeyData         string `yaml:"client-key-data"`
	Token                 string `yaml:"token"`
}

func (d *Driver) PostCheck(info *generic.ClusterInfo) (*generic.ClusterInfo, error) {
	state, err := getState(info)

	if err != nil {
		return nil, err
	}

	client, err := newAzureClient(state)

	if err != nil {
		return nil, err
	}

	result, err := client.GetAccessProfiles(context.Background(), state.ResourceGroup, state.Name, "clusterUser")

	if err != nil {
		return nil, err
	}

	decoded := make([]byte, base64.StdEncoding.DecodedLen(len(*result.KubeConfig)))
	l, err := base64.StdEncoding.Decode(decoded, []byte(*result.KubeConfig))

	if err != nil {
		return nil, err
	}

	clusterConfig := KubeConfig{}
	err = yaml.Unmarshal(decoded[:l], &clusterConfig)

	if err != nil {
		return nil, err
	}

	singleCluster := clusterConfig.Clusters[0]
	singleUser := clusterConfig.Users[0]

	info.Version = clusterConfig.APIVersion
	info.Endpoint = singleCluster.ClusterInfo.Server
	info.Username = state.AdminUsername
	info.Password = singleUser.UserInfo.Token
	info.RootCaCertificate = singleCluster.ClusterInfo.CertificateAuthorityData
	info.ClientCertificate = singleUser.UserInfo.ClientCertificateData
	info.ClientKey = singleUser.UserInfo.ClientKeyData
	//info.NodeCount = int64(*(*cluster.AgentPoolProfiles)[0].Count)
	//info.Metadata["nodePool"] = strconv.Itoa(int(cluster.NodeCount))

	return info, nil
}

// Remove implements driver interface
func (driver *Driver) Remove(info *generic.ClusterInfo) error {
	state, err := getState(info)

	if err != nil {
		return err
	}

	client, err := newAzureClient(state)

	if err != nil {
		return err
	}

	_, err = client.Delete(context.Background(), state.ResourceGroup, state.Name)

	if err != nil {
		return err
	}

	logrus.Infof("Cluster %v removed successfully", state.Name)

	return nil
}
