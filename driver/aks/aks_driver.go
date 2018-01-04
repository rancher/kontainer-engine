package aks

import (
	"strings"

	generic "github.com/rancher/kontainer-engine/driver"
	"fmt"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/utils"
	"github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2017-08-31/containerservice"
	"os"
	"context"
	"github.com/Azure/go-autorest/autorest/to"
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
	AgentPoolFQDN string
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
	driverFlag.Options["node-pool-fqdn"] = &generic.Flag{
		Type:  generic.StringType,
		Usage: "FDQN for the agent pool",
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
	d.AgentPoolFQDN = generic.GetValueFromDriverOptions(driverOptions, generic.StringType, "node-pool-fqdn").(string)
	d.AgentVMSize = generic.GetValueFromDriverOptions(driverOptions, generic.StringType, "node-vm-size").(string)
	d.Count = generic.GetValueFromDriverOptions(driverOptions, generic.IntType, "node-count").(int64)
	d.KubernetesVersion = generic.GetValueFromDriverOptions(driverOptions, generic.StringType, "kubernetes-version").(string)
	d.Location = generic.GetValueFromDriverOptions(driverOptions, generic.StringType, "location").(string)
	d.OsDiskSizeGB = generic.GetValueFromDriverOptions(driverOptions, generic.IntType, "os-disk-size").(int64)
	d.SubscriptionID = generic.GetValueFromDriverOptions(driverOptions, generic.StringType, "subscription-id").(string)
	d.ResourceGroup = generic.GetValueFromDriverOptions(driverOptions, generic.StringType, "resource-group").(string)
	d.AgentPoolFQDN = generic.GetValueFromDriverOptions(driverOptions, generic.StringType, "node-pool-fqdn").(string)
	d.MasterDNSPrefix = generic.GetValueFromDriverOptions(driverOptions, generic.StringType, "master-dns-prefix").(string)
	d.SSHPublicKeyPath = generic.GetValueFromDriverOptions(driverOptions, generic.StringType, "public-key").(string)
	d.AdminUsername = generic.GetValueFromDriverOptions(driverOptions, generic.StringType, "admin-username").(string)
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

	//if d.ResourceGroup == "" {
	//	return fmt.Errorf("resource group is required")
	//}

	//if d.SSHPublicKeyPath == "" {
	//	return fmt.Errorf("path to ssh public key is required")
	//}

	return nil
}

// Create implements driver interface
func (d *Driver) Create() error {
	subscriptionId := os.Getenv("AZURE_SUBSCRIPTION_ID")
	authorizer, err := utils.GetAuthorizer(azure.PublicCloud)

	if err != nil {
		return err
	}

	client := containerservice.NewManagedClustersClient(subscriptionId)
	client.Authorizer = authorizer

	// publicKey, err := ioutil.ReadFile(d.SSHPublicKeyPath)

	if err != nil {
		return err
	}

	myMap := make(map[string]*string)

	ctx := context.Background()
	_, err = client.CreateOrUpdate(ctx, "kube", "my-kubey-clsuter-1111", containerservice.ManagedCluster{
		ID:       to.StringPtr("my-id"),
		Location: to.StringPtr("eastus"),
		Tags:     &myMap,
		ManagedClusterProperties: &containerservice.ManagedClusterProperties{
			DNSPrefix: to.StringPtr("my-super-kuby-prefix-1111"),
			LinuxProfile: &containerservice.LinuxProfile{
				AdminUsername: to.StringPtr("ohadminmyadmin"),
				SSH: &containerservice.SSHConfiguration{
					PublicKeys: &[]containerservice.SSHPublicKey{
						{
							KeyData: to.StringPtr("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDGYwRDsL7LQ4NUSYfzT0nx/aVXNTx5HpgnYjQ9d4OT576JmSDoddQm1HSoqXXIqJxvCGfHmiLOUpR9yNWB57t5t6Hi/x3izp0qBB7XS0SQYzdScw7n8W1AuNzv1pi6kbIGe08IJBv2TbPvpH3GZRcb4uk5pAjQKyeGPww77hN6NqFHogrosRSpLvHNMNZXwKlg3M0PSMdDzPpBTVPW2Sh+06D+tp7LK31WaPUYhAU6jkQY/c6t3O0UCm+t+wwrD09znyKS1fpUDMrnTmNbE9hZ8Bo5X0TnuLc3J6dligr1539Of0ejhzKwpkciv66u+tB2z+udyaLk5sN9qa00oPGj nathanieljenan@MacBook-Pro.tempe.rancherlabs.com\n"),
						},
					},
				},
			},
			AgentPoolProfiles: &[]containerservice.AgentPoolProfile{
				{
					DNSPrefix: to.StringPtr("my-super-kuby-prefix"),
					Name:      to.StringPtr("kubagentpool"),
					VMSize:    containerservice.StandardA0,
				},
			},
			ServicePrincipalProfile: &containerservice.ServicePrincipalProfile{
				ClientID: to.StringPtr(os.Getenv("AZURE_CLIENT_ID")),
				Secret:   to.StringPtr(os.Getenv("AZURE_CLIENT_SECRET")),
			},
		},
	})

	if err != nil {
		return err
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
	return nil, nil
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
