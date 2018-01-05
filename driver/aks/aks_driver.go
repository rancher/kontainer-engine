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
)

type Driver struct {
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
	}
	driverFlag.Options["os-disk-size"] = &generic.Flag{
		Type:  generic.StringType,
		Usage: "OS Disk Size in GB to be used to specify the disk size for every machine in this master/agent pool. If you specify 0, it will apply the default osDisk size according to the vmSize specified.",
	}
	driverFlag.Options["node-vm-size"] = &generic.Flag{
		Type:  generic.StringType,
		Usage: "Size of agent VMs",
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
	}
	driverFlag.Options["base-url"] = &generic.Flag{
		Type:  generic.StringType,
		Usage: "Different base API url to use",
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
func (d *Driver) SetDriverOptions(driverOptions *generic.DriverOptions) error {
	d.Name = generic.GetValueFromDriverOptions(driverOptions, generic.StringType, "name").(string)
	d.AgentDnsPrefix = generic.GetValueFromDriverOptions(driverOptions, generic.StringType, "node-dns-prefix").(string)
	d.AgentVMSize = generic.GetValueFromDriverOptions(driverOptions, generic.StringType, "node-vm-size").(string)
	d.Count = generic.GetValueFromDriverOptions(driverOptions, generic.IntType, "node-count").(int64)
	d.KubernetesVersion = generic.GetValueFromDriverOptions(driverOptions, generic.StringType, "kubernetes-version").(string)
	d.Location = generic.GetValueFromDriverOptions(driverOptions, generic.StringType, "location").(string)
	d.OsDiskSizeGB = generic.GetValueFromDriverOptions(driverOptions, generic.IntType, "os-disk-size").(int64)
	d.SubscriptionID = generic.GetValueFromDriverOptions(driverOptions, generic.StringType, "subscription-id").(string)
	d.ResourceGroup = generic.GetValueFromDriverOptions(driverOptions, generic.StringType, "resource-group").(string)
	d.AgentPoolName = generic.GetValueFromDriverOptions(driverOptions, generic.StringType, "node-pool-name").(string)
	d.MasterDNSPrefix = generic.GetValueFromDriverOptions(driverOptions, generic.StringType, "master-dns-prefix").(string)
	d.SSHPublicKeyPath = generic.GetValueFromDriverOptions(driverOptions, generic.StringType, "public-key").(string)
	d.AdminUsername = generic.GetValueFromDriverOptions(driverOptions, generic.StringType, "admin-username").(string)
	d.BaseUrl = generic.GetValueFromDriverOptions(driverOptions, generic.StringType, "base-url").(string)
	d.ClientID = generic.GetValueFromDriverOptions(driverOptions, generic.StringType, "client-id").(string)
	d.ClientSecret = generic.GetValueFromDriverOptions(driverOptions, generic.StringType, "client-secret").(string)
	tagValues := generic.GetValueFromDriverOptions(driverOptions, generic.StringSliceType).(*generic.StringSlice)
	for _, part := range tagValues.Value {
		kv := strings.Split(part, "=")
		if len(kv) == 2 {
			d.Tag[kv[0]] = kv[1]
		}
	}
	return d.validate()
}

func (d *Driver) validate() error {
	if d.Name == "" {
		return fmt.Errorf("cluster name is required")
	}

	if d.ResourceGroup == "" {
		return fmt.Errorf("resource group is required")
	}

	if d.SSHPublicKeyPath == "" {
		return fmt.Errorf("path to ssh public key is required")
	}

	if d.AdminUsername == "" {
		return fmt.Errorf("admin username is required")
	}

	if d.AgentDnsPrefix == "" {
		return fmt.Errorf("agent dns prefix is required")
	}

	if d.AgentPoolName == "" {
		return fmt.Errorf("agent pool name is required")
	}

	if d.ClientID == "" {
		return fmt.Errorf("client id is required")
	}

	if d.ClientSecret == "" {
		return fmt.Errorf("client secret is required")
	}

	if d.SubscriptionID == "" {
		return fmt.Errorf("subscription id is required")
	}

	return nil
}

const failedStatus = "Failed"
const succeededStatus = "Succeeded"
const creatingStatus = "Creating"

const pollInterval = 30

// Create implements driver interface
func (d *Driver) Create() error {
	authorizer, err := utils.GetAuthorizer(azure.PublicCloud)

	if err != nil {
		return err
	}

	var client containerservice.ManagedClustersClient

	if d.BaseUrl == "" {
		client = containerservice.NewManagedClustersClient(d.SubscriptionID)
	} else {
		client = containerservice.NewManagedClustersClientWithBaseURI(d.BaseUrl, d.SubscriptionID)
	}

	masterDNSPrefix := d.MasterDNSPrefix
	if masterDNSPrefix == "" {
		masterDNSPrefix = d.Name
	}

	agentDNSPrefix := d.AgentDnsPrefix
	if agentDNSPrefix == "" {
		agentDNSPrefix = d.Name + "-agent"
	}

	agentPoolName := d.AgentPoolName
	if agentPoolName == "" {
		agentPoolName = d.Name + "-agent-pool"
	}

	client.Authorizer = authorizer

	publicKey, err := ioutil.ReadFile(d.SSHPublicKeyPath)

	if err != nil {
		return err
	}

	publicKeyContents := string(publicKey)

	myMap := make(map[string]*string)

	ctx := context.Background()
	_, err = client.CreateOrUpdate(ctx, d.ResourceGroup, d.Name, containerservice.ManagedCluster{
		Location: to.StringPtr(d.Location),
		Tags:     &myMap,
		ManagedClusterProperties: &containerservice.ManagedClusterProperties{
			DNSPrefix: to.StringPtr(masterDNSPrefix),
			LinuxProfile: &containerservice.LinuxProfile{
				AdminUsername: to.StringPtr(d.AdminUsername),
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
					Name:      to.StringPtr(d.AgentPoolName),
					VMSize:    containerservice.StandardA0,
				},
			},
			ServicePrincipalProfile: &containerservice.ServicePrincipalProfile{
				ClientID: to.StringPtr(d.ClientID),
				Secret:   to.StringPtr(d.ClientSecret),
			},
		},
	})

	if err != nil {
		return err
	}

	logrus.Info("Request submitted, waiting for cluster to finish creating")

	for {
		result, err := client.Get(ctx, d.ResourceGroup, d.Name)

		if err != nil {
			return err
		}

		state := *result.ProvisioningState

		if state == failedStatus {
			return fmt.Errorf("cluster create has completed with status of 'Failed'")
		}

		if state == succeededStatus {
			logrus.Info("Cluster provisioned successfully")
			return nil
		}

		if state != creatingStatus {
			return fmt.Errorf("unexpected state %v", state)
		}

		logrus.Infof("Cluster has not yet completed provisioning, waiting another %v seconds", pollInterval)

		time.Sleep(pollInterval * time.Second)
	}

	return nil
}

// Update implements driver interface
func (d *Driver) Update() error {
	// todo: implement
	return nil
}

// Get implements driver interface
func (d *Driver) Get() (*generic.ClusterInfo, error) {
	// todo: implement
	return &generic.ClusterInfo{}, nil
}

func (d *Driver) PostCheck() error {
	// todo: implement
	return nil
}

// Remove implements driver interface
func (d *Driver) Remove() error {
	// todo: implement
	return nil
}
