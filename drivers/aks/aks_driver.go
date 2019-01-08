package aks

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2018-03-31/containerservice"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2017-05-10/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/rancher/kontainer-engine/drivers/options"
	"github.com/rancher/kontainer-engine/drivers/util"
	"github.com/rancher/kontainer-engine/types"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"regexp"
	"strings"
	"time"
)

var truePointer = true
var redactionRegex = regexp.MustCompile("\"(clientId|secret)\": \"(.*)\"")

type Driver struct {
	driverCapabilities types.Capabilities
}

type state struct {
	// Path to the public key to use for SSH into cluster
	SSHPublicKeyPath string `json:"sshPublicKeyPath,omitempty"`

	// Cluster Name
	Name string

	// The name that is displayed to the user on the Rancher UI
	DisplayName string

	AgentDNSPrefix                     string
	AgentVMSize                        string
	AgentMaxPods                       int64
	AgentStorageProfile                string
	NetworkPolicy                      string
	NetworkPlugin                      string
	PodCIDR                            string
	AADClientAppID                     string
	AADServerAppID                     string
	AADServerAppSecret                 string
	AADTenantID                        string
	AddonHTTPApplicationRoutingEnabled *bool
	AddonMonitoringEnabled             *bool
	LogAnalyticsWorkspaceResourceID    string
	Count                              int64
	KubernetesVersion                  string
	Location                           string
	AgentOsDiskSizeGB                  int64
	SubscriptionID                     string
	ResourceGroup                      string
	AgentPoolName                      string
	MasterDNSPrefix                    string
	SSHPublicKeyContents               string
	AdminUsername                      string
	BaseURL                            string
	ClientID                           string
	TenantID                           string
	ClientSecret                       string
	VirtualNetwork                     string
	Subnet                             string
	VirtualNetworkResourceGroup        string
	Tag                                map[string]string
	ServiceCIDR                        string
	DNSServiceIP                       string
	DockerBridgeCIDR                   string

	// Cluster info
	ClusterInfo types.ClusterInfo

	AddonHTTPApplicationRoutingDisabled *bool
	AddonMonitoringDisabled             *bool
	AuthBaseURL                         string
}

func NewDriver() types.Driver {
	driver := &Driver{
		driverCapabilities: types.Capabilities{
			Capabilities: make(map[int64]bool),
		},
	}

	driver.driverCapabilities.AddCapability(types.GetVersionCapability)
	driver.driverCapabilities.AddCapability(types.SetVersionCapability)
	driver.driverCapabilities.AddCapability(types.GetClusterSizeCapability)
	driver.driverCapabilities.AddCapability(types.SetClusterSizeCapability)

	return driver
}

// GetDriverCreateOptions implements driver interface
func (d *Driver) GetDriverCreateOptions(ctx context.Context) (*types.DriverFlags, error) {
	driverFlag := types.DriverFlags{
		Options: make(map[string]*types.Flag),
	}
	driverFlag.Options["name"] = &types.Flag{
		Type:  types.StringType,
		Usage: "The internal name of the cluster in Rancher",
	}
	driverFlag.Options["display-name"] = &types.Flag{
		Type:  types.StringType,
		Usage: "The name that is displayed to the user on the Rancher UI",
	}
	driverFlag.Options["subscription-id"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Subscription credentials which uniquely identify Microsoft Azure subscription",
	}
	driverFlag.Options["resource-group"] = &types.Flag{
		Type:  types.StringType,
		Usage: "The name of the resource group",
	}
	driverFlag.Options["location"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Resource location",
		Default: &types.Default{
			DefaultString: "eastus",
		},
	}
	driverFlag.Options["tags"] = &types.Flag{
		Type:  types.StringSliceType,
		Usage: "Resource tags. For example, foo=bar",
	}
	driverFlag.Options["count"] = &types.Flag{
		Type:  types.IntType,
		Usage: "Number of agents (VMs) to host docker containers. Allowed values must be in the range of 1 to 100 (inclusive)",
		Default: &types.Default{
			DefaultInt: 1,
		},
	}
	driverFlag.Options["max-pods"] = &types.Flag{
		Type:  types.IntType,
		Usage: "Maximum number of pods that can run on an agent.",
		Default: &types.Default{
			DefaultInt: 110,
		},
	}
	driverFlag.Options["agent-dns-prefix"] = &types.Flag{
		Type:  types.StringType,
		Usage: "DNS prefix to be used to create the FQDN for the agent pool",
	}
	driverFlag.Options["agent-pool-name"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Name for the agent pool",
		Default: &types.Default{
			DefaultString: "agentpool0",
		},
	}
	driverFlag.Options["agent-osdisk-size"] = &types.Flag{
		Type:  types.IntType,
		Usage: `OS Disk Size in GB to be used to specify the disk size for every machine in this master/agent pool. If you specify 0, it will apply the default osDisk size according to the "--agent-vm-size" specified.`,
	}
	driverFlag.Options["agent-vm-size"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Size of agent VMs",
		Default: &types.Default{
			DefaultString: "Standard_D1_v2",
		},
	}
	driverFlag.Options["agent-storage-profile"] = &types.Flag{
		Type:  types.StringType,
		Usage: fmt.Sprintf("Storage profile specifies what kind of storage used. Chooses from %v.", containerservice.PossibleStorageProfileTypesValues()),
		Default: &types.Default{
			DefaultString: string(containerservice.ManagedDisks),
		},
	}
	driverFlag.Options["kubernetes-version"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Version of Kubernetes specified when creating the managed cluster",
		Default: &types.Default{
			DefaultString: "1.7.9",
		},
	}
	driverFlag.Options["public-key"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Path to the SSH public key to use for the cluster",
	}
	driverFlag.Options["ssh-public-key-contents"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Contents of the SSH public key to use for the cluster",
	}
	driverFlag.Options["master-dns-prefix"] = &types.Flag{
		Type:  types.StringType,
		Usage: "DNS prefix to use for the master",
	}
	driverFlag.Options["admin-username"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Admin username to use for the cluster",
		Default: &types.Default{
			DefaultString: "azureuser",
		},
	}
	driverFlag.Options["base-url"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Different base API url to use",
		Default: &types.Default{
			DefaultString: containerservice.DefaultBaseURI,
		},
	}
	driverFlag.Options["client-id"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Azure client id to use",
	}
	driverFlag.Options["client-secret"] = &types.Flag{
		Type:     types.StringType,
		Password: true,
		Usage:    "Client secret associated with the client-id",
	}
	driverFlag.Options["tenant-id"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Azure tenant id to use",
	}
	driverFlag.Options["virtual-network"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Azure virtual network to use",
	}
	driverFlag.Options["subnet"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Azure subnet to use",
	}
	driverFlag.Options["virtual-network-resource-group"] = &types.Flag{
		Type:  types.StringType,
		Usage: "The resource group that the virtual network is in",
	}
	driverFlag.Options["network-policy"] = &types.Flag{
		Type:  types.StringType,
		Usage: fmt.Sprintf(`Network policy used for building Kubernetes network. Chooses from %v.`, containerservice.PossibleNetworkPolicyValues()),
	}
	driverFlag.Options["network-plugin"] = &types.Flag{
		Type:  types.StringType,
		Usage: fmt.Sprintf(`Network plugin used for building Kubernetes network. Chooses from %v.`, containerservice.PossibleNetworkPluginValues()),
		Default: &types.Default{
			DefaultString: string(containerservice.Azure),
		},
	}
	driverFlag.Options["pod-cidr"] = &types.Flag{
		Type:  types.StringType,
		Usage: fmt.Sprintf(`A CIDR notation IP range from which to assign pod IPs when "--network-plugin" is specified in %q.`, containerservice.Kubenet),
		Value: "172.244.0.0/16",
	}
	driverFlag.Options["service-cidr"] = &types.Flag{
		Type:  types.StringType,
		Usage: "A CIDR notation IP range from which to assign service cluster IPs. It must not overlap with any Subnet IP ranges.",
		Value: "10.0.0.0/16",
	}
	driverFlag.Options["dns-service-ip"] = &types.Flag{
		Type:  types.StringType,
		Usage: `An IP address assigned to the Kubernetes DNS service. It must be within the Kubernetes service address range specified in "--service-cidr".`,
		Value: "10.0.0.10",
	}
	driverFlag.Options["docker-bridge-cidr"] = &types.Flag{
		Type:  types.StringType,
		Usage: `A CIDR notation IP range assigned to the Docker bridge network. It must not overlap with any Subnet IP ranges or the Kubernetes service address range specified in "--service-cidr".`,
		Value: "172.17.0.1/16",
	}
	driverFlag.Options["aad-client-app-id"] = &types.Flag{
		Type:  types.StringType,
		Usage: `The ID of an Azure Active Directory client application of type "Native". This application is for user login via kubectl.`,
	}
	driverFlag.Options["aad-server-app-id"] = &types.Flag{
		Type:  types.StringType,
		Usage: `The ID of an Azure Active Directory server application of type "Web app/API". This application represents the managed cluster's apiserver (Server application).`,
	}
	driverFlag.Options["aad-server-app-secret"] = &types.Flag{
		Type:  types.StringType,
		Usage: `The secret of an Azure Active Directory server application.`,
	}
	driverFlag.Options["aad-tenant-id"] = &types.Flag{
		Type:  types.StringType,
		Usage: `The ID of an Azure Active Directory tenant.`,
	}
	driverFlag.Options["enable-http-application-routing"] = &types.Flag{
		Type:  types.BoolPointerType,
		Usage: `Enable the Kubernetes ingress with automatic public DNS name creation.`,
	}
	driverFlag.Options["enable-monitoring"] = &types.Flag{
		Type:  types.BoolPointerType,
		Usage: `Turn on Log Analytics monitoring. Uses the Log Analytics "Default" workspace if it exists, else creates one. if using an existing workspace, specifies "--log-analytics-workspace-resource-id".`,
	}
	driverFlag.Options["log-analytics-workspace-resource-id"] = &types.Flag{
		Type:  types.StringType,
		Usage: `The resource ID of an existing Log Analytics Workspace to use for storing monitoring data. If not specified, uses the default Log Analytics Workspace if it exists, otherwise creates one.`,
	}
	driverFlag.Options["auth-base-url"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Different authentication API url to use.",
		Default: &types.Default{
			DefaultString: azure.PublicCloud.ActiveDirectoryEndpoint,
		},
	}

	return &driverFlag, nil
}

// GetDriverUpdateOptions implements driver interface
func (d *Driver) GetDriverUpdateOptions(ctx context.Context) (*types.DriverFlags, error) {
	driverFlag := types.DriverFlags{
		Options: make(map[string]*types.Flag),
	}
	driverFlag.Options["location"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Resource location",
	}
	driverFlag.Options["count"] = &types.Flag{
		Type:  types.IntType,
		Usage: "Number of agents (VMs) to host docker containers. Allowed values must be in the range of 1 to 100 (inclusive)",
	}
	driverFlag.Options["kubernetes-version"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Version of Kubernetes specified when creating the managed cluster",
	}
	driverFlag.Options["agent-pool-name"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Name for the agent pool",
	}
	driverFlag.Options["tags"] = &types.Flag{
		Type:  types.StringSliceType,
		Usage: "Resource tags. For example, foo=bar",
	}
	driverFlag.Options["disable-http-application-routing"] = &types.Flag{
		Type:  types.BoolPointerType,
		Usage: `Disable the Kubernetes ingress with automatic public DNS name creation.`,
	}
	driverFlag.Options["disable-monitoring"] = &types.Flag{
		Type:  types.BoolPointerType,
		Usage: `Turn off Log Analytics monitoring.`,
	}
	driverFlag.Options["enable-monitoring"] = &types.Flag{
		Type:  types.BoolPointerType,
		Usage: `Turn on Log Analytics monitoring. Uses the Log Analytics "Default" workspace if it exists, else creates one. if using an existing workspace, specifies "--log-analytics-workspace-resource-id".`,
	}
	driverFlag.Options["log-analytics-workspace-resource-id"] = &types.Flag{
		Type:  types.StringType,
		Usage: `The resource ID of an existing Log Analytics Workspace to use for storing monitoring data. If not specified, uses the default Log Analytics Workspace if it exists, otherwise creates one.`,
	}
	driverFlag.Options["name"] = &types.Flag{
		Type:  types.StringType,
		Usage: "The internal name of the cluster in Rancher",
	}
	driverFlag.Options["subscription-id"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Subscription credentials which uniquely identify Microsoft Azure subscription",
	}
	driverFlag.Options["resource-group"] = &types.Flag{
		Type:  types.StringType,
		Usage: "The name of the resource group",
	}
	driverFlag.Options["base-url"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Different base API url to use",
		Default: &types.Default{
			DefaultString: containerservice.DefaultBaseURI,
		},
	}
	driverFlag.Options["client-id"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Azure client id to use",
	}
	driverFlag.Options["client-secret"] = &types.Flag{
		Type:     types.StringType,
		Password: true,
		Usage:    "Client secret associated with the client-id",
	}
	driverFlag.Options["tenant-id"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Azure tenant id to use",
	}
	driverFlag.Options["auth-base-url"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Different authentication API url to use.",
		Default: &types.Default{
			DefaultString: azure.PublicCloud.ActiveDirectoryEndpoint,
		},
	}

	return &driverFlag, nil
}

// SetDriverOptions implements driver interface
func getStateFromOptions(driverOptions *types.DriverOptions, isCreating bool) (state, error) {
	state := state{Tag: make(map[string]string)}
	state.Name = options.GetValueFromDriverOptions(driverOptions, types.StringType, "name").(string)
	state.DisplayName = options.GetValueFromDriverOptions(driverOptions, types.StringType, "display-name", "displayName").(string)
	state.AgentDNSPrefix = options.GetValueFromDriverOptions(driverOptions, types.StringType, "agent-dns-prefix", "agentDnsPrefix").(string)
	state.AgentVMSize = options.GetValueFromDriverOptions(driverOptions, types.StringType, "agent-vm-size", "agentVmSize").(string)
	state.Count = options.GetValueFromDriverOptions(driverOptions, types.IntType, "count").(int64)
	state.AgentMaxPods = options.GetValueFromDriverOptions(driverOptions, types.IntType, "max-pods", "maxPods").(int64)
	state.KubernetesVersion = options.GetValueFromDriverOptions(driverOptions, types.StringType, "kubernetes-version", "kubernetesVersion").(string)
	state.Location = options.GetValueFromDriverOptions(driverOptions, types.StringType, "location").(string)
	state.AgentOsDiskSizeGB = options.GetValueFromDriverOptions(driverOptions, types.IntType, "agent-osdisk-size", "agentOsDiskSize").(int64)
	state.AgentStorageProfile = options.GetValueFromDriverOptions(driverOptions, types.StringType, "agent-storage-profile").(string)
	state.AADClientAppID = options.GetValueFromDriverOptions(driverOptions, types.StringType, "aad-client-app-id", "addClientAppId").(string)
	state.AADServerAppID = options.GetValueFromDriverOptions(driverOptions, types.StringType, "aad-server-app-id", "addServerAppId").(string)
	state.AADServerAppSecret = options.GetValueFromDriverOptions(driverOptions, types.StringType, "aad-server-app-secret", "addServerAppSecret").(string)
	state.AADTenantID = options.GetValueFromDriverOptions(driverOptions, types.StringType, "aad-tenant-id", "addTenantId").(string)
	state.AddonHTTPApplicationRoutingEnabled = options.GetValueFromDriverOptions(driverOptions, types.BoolPointerType, "enable-http-application-routing", "enableHttpApplicationRouting").(*bool)
	state.AddonMonitoringEnabled = options.GetValueFromDriverOptions(driverOptions, types.BoolPointerType, "enable-monitoring", "enableMonitoring").(*bool)
	state.LogAnalyticsWorkspaceResourceID = options.GetValueFromDriverOptions(driverOptions, types.StringType, "log-analytics-workspace-resource-id", "logAnalyticsWorkspaceResourceId").(string)
	state.SubscriptionID = options.GetValueFromDriverOptions(driverOptions, types.StringType, "subscription-id", "subscriptionId").(string)
	state.ResourceGroup = options.GetValueFromDriverOptions(driverOptions, types.StringType, "resource-group", "resourceGroup").(string)
	state.AgentPoolName = options.GetValueFromDriverOptions(driverOptions, types.StringType, "agent-pool-name", "agentPoolName").(string)
	state.MasterDNSPrefix = options.GetValueFromDriverOptions(driverOptions, types.StringType, "master-dns-prefix", "masterDnsPrefix").(string)
	state.SSHPublicKeyPath = options.GetValueFromDriverOptions(driverOptions, types.StringType, "public-key", "publicKey").(string)
	state.SSHPublicKeyContents = options.GetValueFromDriverOptions(driverOptions, types.StringType, "ssh-public-key-contents", "sshPublicKeyContents").(string)
	state.AdminUsername = options.GetValueFromDriverOptions(driverOptions, types.StringType, "admin-username", "adminUsername").(string)
	state.BaseURL = options.GetValueFromDriverOptions(driverOptions, types.StringType, "base-url", "baseUrl").(string)
	state.ClientID = options.GetValueFromDriverOptions(driverOptions, types.StringType, "client-id", "clientId").(string)
	state.TenantID = options.GetValueFromDriverOptions(driverOptions, types.StringType, "tenant-id", "tenantId").(string)
	state.ClientSecret = options.GetValueFromDriverOptions(driverOptions, types.StringType, "client-secret", "clientSecret").(string)
	state.VirtualNetwork = options.GetValueFromDriverOptions(driverOptions, types.StringType, "virtual-network", "virtualNetwork").(string)
	state.Subnet = options.GetValueFromDriverOptions(driverOptions, types.StringType, "subnet").(string)
	state.VirtualNetworkResourceGroup = options.GetValueFromDriverOptions(driverOptions, types.StringType, "virtual-network-resource-group", "virtualNetworkResourceGroup").(string)
	state.NetworkPolicy = options.GetValueFromDriverOptions(driverOptions, types.StringType, "network-policy", "networkPolicy").(string)
	state.NetworkPlugin = options.GetValueFromDriverOptions(driverOptions, types.StringType, "network-plugin", "networkPlugin").(string)
	state.PodCIDR = options.GetValueFromDriverOptions(driverOptions, types.StringType, "pod-cidr", "podCidr").(string)
	state.ServiceCIDR = options.GetValueFromDriverOptions(driverOptions, types.StringType, "service-cidr", "serviceCidr").(string)
	state.DNSServiceIP = options.GetValueFromDriverOptions(driverOptions, types.StringType, "dns-service-ip", "dnsServiceIp").(string)
	state.DockerBridgeCIDR = options.GetValueFromDriverOptions(driverOptions, types.StringType, "docker-bridge-cidr", "dockerBridgeCidr").(string)
	tagValues := options.GetValueFromDriverOptions(driverOptions, types.StringSliceType, "tags").(*types.StringSlice)
	for _, part := range tagValues.Value {
		kv := strings.Split(part, "=")
		if len(kv) == 2 {
			state.Tag[kv[0]] = kv[1]
		}
	}

	state.AddonHTTPApplicationRoutingDisabled = options.GetValueFromDriverOptions(driverOptions, types.BoolPointerType, "disable-http-application-routing", "disableHttpApplicationRouting").(*bool)
	state.AddonMonitoringDisabled = options.GetValueFromDriverOptions(driverOptions, types.BoolPointerType, "disable-monitoring", "disableMonitoring").(*bool)
	state.AuthBaseURL = options.GetValueFromDriverOptions(driverOptions, types.StringType, "auth-base-url", "authBaseUrl").(string)

	return state, state.validate(isCreating)
}

func (state *state) validate(isCreating bool) error {
	if state.Name == "" {
		return fmt.Errorf("cluster name is required")
	}

	if state.ResourceGroup == "" {
		return fmt.Errorf("resource group is required")
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

	if isCreating {
		if state.SSHPublicKeyPath == "" && state.SSHPublicKeyContents == "" {
			return fmt.Errorf("path to ssh public key or public key contents is required")
		}
	} else {
		if state.Location == "" {
			return fmt.Errorf("location is required")
		}
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
	authBaseURL := state.AuthBaseURL
	if authBaseURL == "" {
		authBaseURL = azure.PublicCloud.ActiveDirectoryEndpoint
	}

	oauthConfig, err := adal.NewOAuthConfig(authBaseURL, state.TenantID)
	if err != nil {
		return nil, err
	}

	spToken, err := adal.NewServicePrincipalToken(*oauthConfig, state.ClientID, state.ClientSecret, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	authorizer := autorest.NewBearerAuthorizer(spToken)

	baseURL := state.BaseURL
	if baseURL == "" {
		baseURL = containerservice.DefaultBaseURI
	}

	client := containerservice.NewManagedClustersClientWithBaseURI(baseURL, state.SubscriptionID)
	client.Authorizer = authorizer

	return &client, nil
}

func newResourceGroupsClient(state state) (*resources.GroupsClient, error) {
	authBaseURL := state.AuthBaseURL
	if authBaseURL == "" {
		authBaseURL = azure.PublicCloud.ActiveDirectoryEndpoint
	}

	oauthConfig, err := adal.NewOAuthConfig(authBaseURL, state.TenantID)
	if err != nil {
		return nil, err
	}

	spToken, err := adal.NewServicePrincipalToken(*oauthConfig, state.ClientID, state.ClientSecret, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	authorizer := autorest.NewBearerAuthorizer(spToken)

	baseURL := state.BaseURL
	if baseURL == "" {
		baseURL = containerservice.DefaultBaseURI
	}

	client := resources.NewGroupsClientWithBaseURI(baseURL, state.SubscriptionID)
	client.Authorizer = authorizer

	return &client, nil
}

func newResourcesClient(state state) (*resources.Client, error) {
	authBaseURL := state.AuthBaseURL
	if authBaseURL == "" {
		authBaseURL = azure.PublicCloud.ActiveDirectoryEndpoint
	}

	oauthConfig, err := adal.NewOAuthConfig(authBaseURL, state.TenantID)
	if err != nil {
		return nil, err
	}

	spToken, err := adal.NewServicePrincipalToken(*oauthConfig, state.ClientID, state.ClientSecret, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	authorizer := autorest.NewBearerAuthorizer(spToken)

	baseURL := state.BaseURL
	if baseURL == "" {
		baseURL = containerservice.DefaultBaseURI
	}

	client := resources.NewClientWithBaseURI(baseURL, state.SubscriptionID)
	client.Authorizer = authorizer

	return &client, nil
}

const failedStatus = "Failed"
const succeededStatus = "Succeeded"
const creatingStatus = "Creating"
const updatingStatus = "Updating"

const pollInterval = 30

func (d *Driver) Create(ctx context.Context, options *types.DriverOptions, _ *types.ClusterInfo) (*types.ClusterInfo, error) {
	return d.createOrUpdate(ctx, options, true, true)
}

func (d *Driver) Update(ctx context.Context, info *types.ClusterInfo, options *types.DriverOptions) (*types.ClusterInfo, error) {
	return d.createOrUpdate(ctx, options, false, false)
}

func (d *Driver) createOrUpdate(ctx context.Context, options *types.DriverOptions, sendRBAC, isCreating bool) (*types.ClusterInfo, error) {
	driverState, err := getStateFromOptions(options, isCreating)
	if err != nil {
		return nil, err
	}

	clustersClient, err := newAzureClient(driverState)
	if err != nil {
		return nil, err
	}

	resourceGroupsClient, err := newResourceGroupsClient(driverState)
	if err != nil {
		return nil, err
	}

	resourcesClient, err := newResourcesClient(driverState)
	if err != nil {
		return nil, err
	}

	masterDNSPrefix := driverState.MasterDNSPrefix
	if masterDNSPrefix == "" {
		masterDNSPrefix = driverState.getDefaultDNSPrefix() + "-master"
	}

	tags := make(map[string]*string)
	for key, val := range driverState.Tag {
		if val != "" {
			tags[key] = to.StringPtr(val)
		}
	}
	displayName := driverState.DisplayName
	if displayName == "" {
		displayName = driverState.Name
	}
	tags["displayName"] = to.StringPtr(displayName)

	exists, err := d.resourceGroupExists(ctx, resourceGroupsClient, driverState.ResourceGroup)
	if err != nil {
		return nil, err
	}

	if !exists {
		logrus.Infof("resource group %v does not exist, creating", driverState.ResourceGroup)
		err = d.createResourceGroup(ctx, resourceGroupsClient, driverState)
		if err != nil {
			return nil, err
		}
	}

	var aadProfile *containerservice.ManagedClusterAADProfile
	if d.hasAzureActiveDirectoryProfile(driverState) {
		aadProfile = &containerservice.ManagedClusterAADProfile{
			ClientAppID: to.StringPtr(driverState.AADClientAppID),
			ServerAppID: to.StringPtr(driverState.AADServerAppID),
		}

		if driverState.AADServerAppSecret != "" {
			aadProfile.ServerAppSecret = to.StringPtr(driverState.AADServerAppSecret)
		}

		if driverState.AADTenantID != "" {
			aadProfile.TenantID = to.StringPtr(driverState.AADTenantID)
		}
	}

	var addonProfiles map[string]*containerservice.ManagedClusterAddonProfile
	if d.hasAddonProfile(driverState) {
		addonProfiles = make(map[string]*containerservice.ManagedClusterAddonProfile, 2)

		if to.Bool(driverState.AddonMonitoringEnabled) {
			workspaceResourceID, err := d.ensureLogAnalyticsWorkspaceForMonitoring(ctx, resourcesClient, driverState)
			if err != nil {
				return nil, err
			}

			addonProfiles["omsagent"] = &containerservice.ManagedClusterAddonProfile{
				Enabled: &truePointer,
				Config: map[string]*string{
					"logAnalyticsWorkspaceResourceID": to.StringPtr(workspaceResourceID),
				},
			}
		} else if to.Bool(driverState.AddonMonitoringDisabled) {
			addonProfiles["omsagent"] = &containerservice.ManagedClusterAddonProfile{
				Enabled: to.BoolPtr(false),
			}
		}

		if to.Bool(driverState.AddonHTTPApplicationRoutingEnabled) {
			addonProfiles["httpApplicationRouting"] = &containerservice.ManagedClusterAddonProfile{
				Enabled: &truePointer,
			}
		} else if to.Bool(driverState.AddonHTTPApplicationRoutingDisabled) {
			addonProfiles["httpApplicationRouting"] = &containerservice.ManagedClusterAddonProfile{
				Enabled: to.BoolPtr(false),
			}
		}
	}

	var vmNetSubnetID *string
	var networkProfile *containerservice.NetworkProfile
	if d.hasCustomVirtualNetwork(driverState) {
		virtualNetworkResourceGroup := driverState.ResourceGroup

		// if virtual network resource group is set, use it, otherwise assume it is the same as the cluster
		if driverState.VirtualNetworkResourceGroup != "" {
			virtualNetworkResourceGroup = driverState.VirtualNetworkResourceGroup
		}

		vmNetSubnetID = to.StringPtr(fmt.Sprintf(
			"/subscriptions/%v/resourceGroups/%v/providers/Microsoft.Network/virtualNetworks/%v/subnets/%v",
			driverState.SubscriptionID,
			virtualNetworkResourceGroup,
			driverState.VirtualNetwork,
			driverState.Subnet,
		))

		networkProfile = &containerservice.NetworkProfile{
			DNSServiceIP:     to.StringPtr(driverState.DNSServiceIP),
			DockerBridgeCidr: to.StringPtr(driverState.DockerBridgeCIDR),
			NetworkPlugin:    containerservice.NetworkPlugin(driverState.NetworkPlugin),
			ServiceCidr:      to.StringPtr(driverState.ServiceCIDR),
		}

		if networkProfile.NetworkPlugin == containerservice.Kubenet {
			networkProfile.PodCidr = to.StringPtr(driverState.PodCIDR)
		}

		if driverState.NetworkPolicy != "" {
			networkProfile.NetworkPolicy = containerservice.NetworkPolicy(driverState.NetworkPolicy)
		}
	}

	var agentPoolProfiles *[]containerservice.ManagedClusterAgentPoolProfile
	if d.hasAgentPoolProfile(driverState) {
		var countPointer *int32
		if driverState.Count > 0 {
			countPointer = to.Int32Ptr(int32(driverState.Count))
		}

		var maxPodsPointer *int32
		if driverState.AgentMaxPods > 0 {
			maxPodsPointer = to.Int32Ptr(int32(driverState.AgentMaxPods))
		}

		var osDiskSizeGBPointer *int32
		if driverState.AgentOsDiskSizeGB > 0 {
			osDiskSizeGBPointer = to.Int32Ptr(int32(driverState.AgentOsDiskSizeGB))
		}

		agentDNSPrefix := driverState.AgentDNSPrefix
		if agentDNSPrefix == "" {
			agentDNSPrefix = driverState.getDefaultDNSPrefix() + "-agent"
		}

		agentPoolProfiles = &[]containerservice.ManagedClusterAgentPoolProfile{
			{
				DNSPrefix:      to.StringPtr(agentDNSPrefix),
				Count:          countPointer,
				MaxPods:        maxPodsPointer,
				Name:           to.StringPtr(driverState.AgentPoolName),
				OsDiskSizeGB:   osDiskSizeGBPointer,
				OsType:         containerservice.Linux,
				StorageProfile: containerservice.StorageProfileTypes(driverState.AgentStorageProfile),
				VMSize:         containerservice.VMSizeTypes(driverState.AgentVMSize),
				VnetSubnetID:   vmNetSubnetID,
			},
		}
	}

	var linuxProfile *containerservice.LinuxProfile
	if d.hasLinuxProfile(driverState) {
		var publicKey []byte
		if driverState.SSHPublicKeyContents == "" {
			publicKey, err = ioutil.ReadFile(driverState.SSHPublicKeyPath)
			if err != nil {
				return nil, err
			}
		} else {
			publicKey = []byte(driverState.SSHPublicKeyContents)
		}
		publicKeyContents := string(publicKey)

		linuxProfile = &containerservice.LinuxProfile{
			AdminUsername: to.StringPtr(driverState.AdminUsername),
			SSH: &containerservice.SSHConfiguration{
				PublicKeys: &[]containerservice.SSHPublicKey{
					{
						KeyData: to.StringPtr(publicKeyContents),
					},
				},
			},
		}
	}

	managedCluster := containerservice.ManagedCluster{
		Location: to.StringPtr(driverState.Location),
		Tags:     tags,
		ManagedClusterProperties: &containerservice.ManagedClusterProperties{
			KubernetesVersion: to.StringPtr(driverState.KubernetesVersion),
			DNSPrefix:         to.StringPtr(masterDNSPrefix),
			AadProfile:        aadProfile,
			AddonProfiles:     addonProfiles,
			AgentPoolProfiles: agentPoolProfiles,
			LinuxProfile:      linuxProfile,
			NetworkProfile:    networkProfile,
			ServicePrincipalProfile: &containerservice.ServicePrincipalProfile{
				ClientID: to.StringPtr(driverState.ClientID),
				Secret:   to.StringPtr(driverState.ClientSecret),
			},
		},
	}

	if sendRBAC {
		managedCluster.ManagedClusterProperties.EnableRBAC = &truePointer
	}

	logClusterConfig(managedCluster)
	_, err = clustersClient.CreateOrUpdate(ctx, driverState.ResourceGroup, driverState.Name, managedCluster)
	if err != nil {
		return nil, err
	}

	logrus.Info("Request submitted, waiting for cluster to finish creating")

	failedCount := 0

	for {
		result, err := clustersClient.Get(ctx, driverState.ResourceGroup, driverState.Name)
		if err != nil {
			return nil, err
		}

		state := *result.ProvisioningState

		if state == failedStatus {
			if failedCount > 3 {
				logrus.Errorf("cluster recovery failed, retries depleted")
				return nil, fmt.Errorf("cluster create has completed with status of 'Failed'")
			}

			failedCount = failedCount + 1
			logrus.Infof("cluster marked as failed but waiting for recovery: retries left %v", 3-failedCount)
			time.Sleep(pollInterval * time.Second)
		}

		if state == succeededStatus {
			logrus.Info("Cluster provisioned successfully")
			info := &types.ClusterInfo{}
			err := storeState(info, driverState)

			return info, err
		}

		if state != creatingStatus && state != updatingStatus {
			logrus.Errorf("Azure failed to provision cluster with state: %v", state)
			return nil, fmt.Errorf("failed to provision Azure cluster")
		}

		logrus.Infof("Cluster has not yet completed provisioning, waiting another %v seconds", pollInterval)

		time.Sleep(pollInterval * time.Second)
	}
}

func (d *Driver) hasCustomVirtualNetwork(state state) bool {
	return state.VirtualNetwork != "" && state.Subnet != ""
}

func (d *Driver) hasAzureActiveDirectoryProfile(state state) bool {
	return state.AADClientAppID != "" && state.AADServerAppID != "" && state.AADServerAppSecret != ""
}

func (d *Driver) hasAddonProfile(state state) bool {
	for _, status := range []*bool{state.AddonMonitoringDisabled, state.AddonMonitoringEnabled, state.AddonHTTPApplicationRoutingDisabled, state.AddonHTTPApplicationRoutingEnabled} {
		if status != nil {
			return true
		}
	}
	return false
}

func (d *Driver) hasAgentPoolProfile(state state) bool {
	return state.AgentPoolName != ""
}

func (d *Driver) hasLinuxProfile(state state) bool {
	return state.AdminUsername != "" && (state.SSHPublicKeyContents != "" || state.SSHPublicKeyPath != "")
}

func (d *Driver) ensureLogAnalyticsWorkspaceForMonitoring(ctx context.Context, client *resources.Client, state state) (string, error) {
	workspaceResourceID := state.LogAnalyticsWorkspaceResourceID
	if workspaceResourceID == "" {
		// log analytics workspaces cannot be created in WCUS region due to capacity limits
		// so mapped to EUS per discussion with log analytics team
		locationToOmsRegionCodeMap := map[string]string{
			"eastus":             "EUS",
			"westeurope":         "WEU",
			"southeastasia":      "SEA",
			"australiasoutheast": "ASE",
			"usgovvirginia":      "USGV",
			"westcentralus":      "EUS",
			"japaneast":          "EJP",
			"uksouth":            "SUK",
			"canadacentral":      "CCA",
			"centralindia":       "CIN",
			"eastus2euap":        "EAP",
		}

		regionToOmsRegionMap := map[string]string{
			"australiaeast":      "australiasoutheast",
			"australiasoutheast": "australiasoutheast",
			"brazilsouth":        "eastus",
			"canadacentral":      "canadacentral",
			"canadaeast":         "canadacentral",
			"centralus":          "eastus",
			"eastasia":           "southeastasia",
			"eastus":             "eastus",
			"eastus2":            "eastus",
			"japaneast":          "japaneast",
			"japanwest":          "japaneast",
			"northcentralus":     "eastus",
			"northeurope":        "westeurope",
			"southcentralus":     "eastus",
			"southeastasia":      "southeastasia",
			"uksouth":            "uksouth",
			"ukwest":             "uksouth",
			"westcentralus":      "eastus",
			"westeurope":         "westeurope",
			"westus":             "eastus",
			"westus2":            "eastus",
			"centralindia":       "centralindia",
			"southindia":         "centralindia",
			"westindia":          "centralindia",
			"koreacentral":       "southeastasia",
			"koreasouth":         "southeastasia",
			"francecentral":      "westeurope",
			"francesouth":        "westeurope",
		}

		workspaceRegion := regionToOmsRegionMap[state.Location]
		workspaceRegionCode := locationToOmsRegionCodeMap[workspaceRegion]
		workspaceResourceGroup := fmt.Sprintf("DefaultResourceGroup-%s", workspaceRegionCode)
		workspaceName := fmt.Sprintf("DefaultWorkspace-%s-%s", state.SubscriptionID, workspaceRegionCode)

		workspaceResourceID = fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.OperationalInsights/workspaces/%s", state.SubscriptionID, workspaceResourceGroup, workspaceName)
	}

	if !strings.HasPrefix(workspaceResourceID, "/") {
		workspaceResourceID = "/" + workspaceResourceID
	}
	workspaceResourceID = strings.TrimSuffix(workspaceResourceID, "/")

	exist, err := d.resourceExists(ctx, client, workspaceResourceID)
	if err != nil {
		return "", err
	}
	if !exist {
		return "", fmt.Errorf("can't get default Log Analytics Workspace with ID '%s'", workspaceResourceID)
	}

	return workspaceResourceID, nil
}

func (d *Driver) resourceGroupExists(ctx context.Context, client *resources.GroupsClient, groupName string) (bool, error) {
	resp, err := client.CheckExistence(ctx, groupName)
	if err != nil {
		return false, fmt.Errorf("error getting resource group %v: %v", groupName, err)
	}

	return resp.StatusCode == 204, nil
}

func (d *Driver) createResourceGroup(ctx context.Context, client *resources.GroupsClient, state state) error {
	_, err := client.CreateOrUpdate(ctx, state.ResourceGroup, resources.Group{
		Name:     to.StringPtr(state.ResourceGroup),
		Location: to.StringPtr(state.Location),
	})
	if err != nil {
		return fmt.Errorf("error creating resource group %v: %v", state.ResourceGroup, err)
	}

	return nil
}

func (d *Driver) resourceExists(ctx context.Context, client *resources.Client, resourceID string) (bool, error) {
	resp, err := client.CheckExistenceByID(ctx, resourceID)
	if err != nil {
		return false, fmt.Errorf("error getting resource %v: %v", resourceID, err)
	}

	return resp.StatusCode == 204, nil
}

func storeState(info *types.ClusterInfo, state state) error {
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

func getState(info *types.ClusterInfo) (state, error) {
	state := state{}

	err := json.Unmarshal([]byte(info.Metadata["state"]), &state)

	if err != nil {
		logrus.Errorf("Error encountered while marshalling state: %v", err)
	}

	return state, err
}

func (d *Driver) GetVersion(ctx context.Context, info *types.ClusterInfo) (*types.KubernetesVersion, error) {
	state, err := getState(info)

	if err != nil {
		return nil, err
	}

	client, err := newAzureClient(state)

	if err != nil {
		return nil, err
	}

	cluster, err := client.Get(context.Background(), state.ResourceGroup, state.Name)

	if err != nil {
		return nil, fmt.Errorf("error getting cluster info: %v", err)
	}

	return &types.KubernetesVersion{Version: *cluster.KubernetesVersion}, nil
}

func (d *Driver) SetVersion(ctx context.Context, info *types.ClusterInfo, version *types.KubernetesVersion) error {
	state, err := getState(info)

	if err != nil {
		return err
	}

	client, err := newAzureClient(state)

	if err != nil {
		return err
	}

	cluster, err := client.Get(context.Background(), state.ResourceGroup, state.Name)

	if err != nil {
		return fmt.Errorf("error getting cluster info: %v", err)
	}

	cluster.KubernetesVersion = to.StringPtr(version.Version)

	_, err = client.CreateOrUpdate(context.Background(), state.ResourceGroup, state.Name, cluster)

	if err != nil {
		return fmt.Errorf("error updating kubernetes version: %v", err)
	}

	return nil
}

func (d *Driver) GetClusterSize(ctx context.Context, info *types.ClusterInfo) (*types.NodeCount, error) {
	state, err := getState(info)

	if err != nil {
		return nil, err
	}

	client, err := newAzureClient(state)

	if err != nil {
		return nil, err
	}

	result, err := client.Get(context.Background(), state.ResourceGroup, state.Name)

	if err != nil {
		return nil, fmt.Errorf("error getting cluster info: %v", err)
	}

	return &types.NodeCount{Count: int64(*(*result.AgentPoolProfiles)[0].Count)}, nil
}

func (d *Driver) SetClusterSize(ctx context.Context, info *types.ClusterInfo, size *types.NodeCount) error {
	state, err := getState(info)

	if err != nil {
		return err
	}

	client, err := newAzureClient(state)

	if err != nil {
		return err
	}

	cluster, err := client.Get(context.Background(), state.ResourceGroup, state.Name)

	if err != nil {
		return fmt.Errorf("error getting cluster info: %v", err)
	}

	// mutate struct
	(*cluster.ManagedClusterProperties.AgentPoolProfiles)[0].Count = to.Int32Ptr(int32(size.Count))

	// PUT same data
	_, err = client.CreateOrUpdate(context.Background(), state.ResourceGroup, state.Name, cluster)

	if err != nil {
		return fmt.Errorf("error updating cluster size: %v", err)
	}

	return nil
}

// KubeConfig struct for marshalling config files
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

const retries = 5

func (d *Driver) PostCheck(ctx context.Context, info *types.ClusterInfo) (*types.ClusterInfo, error) {
	logrus.Info("starting post-check")

	state, err := getState(info)

	if err != nil {
		return nil, err
	}

	client, err := newAzureClient(state)

	if err != nil {
		return nil, err
	}

	result, err := client.GetAccessProfile(context.Background(), state.ResourceGroup, state.Name, "clusterUser")

	if err != nil {
		return nil, err
	}

	clusterConfig := KubeConfig{}
	err = yaml.Unmarshal(*result.KubeConfig, &clusterConfig)

	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal kubeconfig: %v", err)
	}

	singleCluster := clusterConfig.Clusters[0]
	singleUser := clusterConfig.Users[0]

	info.Version = clusterConfig.APIVersion
	info.Endpoint = singleCluster.ClusterInfo.Server
	info.Password = singleUser.UserInfo.Token
	info.RootCaCertificate = singleCluster.ClusterInfo.CertificateAuthorityData
	info.ClientCertificate = singleUser.UserInfo.ClientCertificateData
	info.ClientKey = singleUser.UserInfo.ClientKeyData

	capem, err := base64.StdEncoding.DecodeString(singleCluster.ClusterInfo.CertificateAuthorityData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode CA: %v", err)
	}

	key, err := base64.StdEncoding.DecodeString(singleUser.UserInfo.ClientKeyData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode client key: %v", err)
	}

	cert, err := base64.StdEncoding.DecodeString(singleUser.UserInfo.ClientCertificateData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode client cert: %v", err)
	}

	host := singleCluster.ClusterInfo.Server
	if !strings.HasPrefix(host, "https://") {
		host = fmt.Sprintf("https://%s", host)
	}

	config := &rest.Config{
		Host: host,
		TLSClientConfig: rest.TLSClientConfig{
			CAData:   capem,
			KeyData:  key,
			CertData: cert,
		},
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating clientset: %v", err)
	}

	failureCount := 0

	for {
		info.ServiceAccountToken, err = util.GenerateServiceAccountToken(clientset)

		if err == nil {
			logrus.Info("service account token generated successfully")
			break
		} else {
			if failureCount < retries {
				logrus.Infof("service account token generation failed, retries left: %v", retries-failureCount)
				failureCount = failureCount + 1

				time.Sleep(pollInterval * time.Second)
			} else {
				logrus.Error("retries exceeded, failing post-check")
				return nil, err
			}
		}
	}

	logrus.Info("post-check completed successfully")

	return info, nil
}

func (d *Driver) Remove(ctx context.Context, info *types.ClusterInfo) error {
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

func (d *Driver) GetCapabilities(ctx context.Context) (*types.Capabilities, error) {
	return &d.driverCapabilities, nil
}

func logClusterConfig(config containerservice.ManagedCluster) {
	if logrus.GetLevel() == logrus.DebugLevel {
		out, err := json.Marshal(config)
		if err != nil {
			logrus.Error("Error marshalling config for logging")
			return
		}
		output := string(out)
		output = redactionRegex.ReplaceAllString(output, "$1: [REDACTED]")
		logrus.Debugf("Sending cluster config to AKS: %v", output)
	}
}

func (d *Driver) ETCDSave(ctx context.Context, clusterInfo *types.ClusterInfo, opts *types.DriverOptions, snapshotName string) error {
	return fmt.Errorf("ETCD backup operations are not implemented")
}

func (d *Driver) ETCDRestore(ctx context.Context, clusterInfo *types.ClusterInfo, opts *types.DriverOptions, snapshotName string) error {
	return fmt.Errorf("ETCD backup operations are not implemented")
}

func (d *Driver) GetK8SCapabilities(ctx context.Context, _ *types.DriverOptions) (*types.K8SCapabilities, error) {
	return &types.K8SCapabilities{
		L4LoadBalancer: &types.LoadBalancerCapabilities{
			Enabled:              true,
			Provider:             "Azure L4 LB",
			ProtocolsSupported:   []string{"TCP", "UDP"},
			HealthCheckSupported: true,
		},
	}, nil
}
