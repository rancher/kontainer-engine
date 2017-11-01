package gke

import (
	"strings"

	"encoding/base64"
	"fmt"
	"time"

	"os"

	generic "github.com/rancher/kontainer-engine/driver"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	raw "google.golang.org/api/container/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/pkg/api/v1"
	v1beta1 "k8s.io/client-go/pkg/apis/rbac/v1beta1"
	// to register gcp auth provider
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd/api"
)

const (
	defaultNodeCount     = 3
	runningStatus        = "RUNNING"
	defaultNamespace     = "default"
	clusterAdmin         = "cluster-admin"
	netesDefault         = "netes-default"
	defaultCredentialEnv = "GOOGLE_APPLICATION_CREDENTIALS"
)

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
	InitialNodeCount int64
	// The authentication information for accessing the master
	MasterAuth *raw.MasterAuth
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

	// Update Config
	UpdateConfig updateConfig
}

type updateConfig struct {
	// the number of node
	NodeCount int64
	// Master kubernetes version
	MasterVersion string
	// Node kubernetes version
	NodeVersion string
}

// NewDriver creates a gke Driver
func NewDriver() *Driver {
	return &Driver{
		NodeConfig: &raw.NodeConfig{
			Labels: map[string]string{},
		},
	}
}

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
	driverFlag.Options["initial-node-count"] = &generic.Flag{
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

func (d *Driver) SetDriverOptions(driverOptions *generic.DriverOptions) error {
	d.Name = driverOptions.StringOptions["name"]
	d.ProjectID = driverOptions.StringOptions["projectId"]
	d.Zone = driverOptions.StringOptions["zone"]
	d.NodePoolID = driverOptions.StringOptions["nodePool"]
	d.ClusterIpv4Cidr = driverOptions.StringOptions["cluster-ipv4-cidr"]
	d.Description = driverOptions.StringOptions["description"]
	d.NodeConfig.DiskSizeGb = driverOptions.IntOptions["disk-size-gb"]
	d.NodeConfig.MachineType = driverOptions.StringOptions["machine-type"]
	d.CredentialPath = driverOptions.StringOptions["gke-credential-path"]
	d.EnableAlphaFeature = driverOptions.BoolOptions["enable-alpha-feature"]
	if v, ok := driverOptions.IntOptions["initial-node-count"]; ok {
		d.InitialNodeCount = v
	}
	if v, ok := driverOptions.StringSliceOptions["machine-type"]; ok {
		for _, part := range v.Value {
			kv := strings.Split(part, "=")
			if len(kv) == 2 {
				d.NodeConfig.Labels[kv[0]] = kv[1]
			}
		}
	}

	// updateConfig
	d.UpdateConfig.NodeCount = driverOptions.IntOptions["node-count"]
	d.UpdateConfig.MasterVersion = driverOptions.StringOptions["master-version"]
	d.UpdateConfig.NodeVersion = driverOptions.StringOptions["node-version"]
	return d.validate()
}

func (d *Driver) validate() error {
	if d.ProjectID == "" || d.Zone == "" || d.Name == "" {
		logrus.Errorf("ProjectID or Zone or Name is required")
		return fmt.Errorf("projectID or zone or name is required")
	}
	if d.InitialNodeCount == 0 {
		d.InitialNodeCount = defaultNodeCount
	}
	return nil
}

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

func (d *Driver) Update() error {
	svc, err := d.getServiceClient()
	if err != nil {
		return err
	}
	logrus.Debugf("Received Update config %v", d.UpdateConfig)
	if d.UpdateConfig.MasterVersion != "" {
		logrus.Infof("Updating master to %v", d.UpdateConfig.MasterVersion)
		operation, err := svc.Projects.Zones.Clusters.Update(d.ProjectID, d.Zone, d.Name, &raw.UpdateClusterRequest{
			Update: &raw.ClusterUpdate{
				DesiredMasterVersion: d.UpdateConfig.MasterVersion,
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

	if d.UpdateConfig.NodeVersion != "" {
		logrus.Infof("Updating node version to %v", d.UpdateConfig.NodeVersion)
		operation, err := svc.Projects.Zones.Clusters.NodePools.Update(d.ProjectID, d.Zone, d.Name, d.NodePoolID, &raw.UpdateNodePoolRequest{
			NodeVersion: d.UpdateConfig.NodeVersion,
		}).Context(context.Background()).Do()
		if err != nil {
			return err
		}
		logrus.Debugf("Nodepool %s update is called for project %s, zone %s and cluster %s. Status Code %v", d.NodePoolID, d.ProjectID, d.Zone, d.Name, operation.HTTPStatusCode)
		if err := d.waitNodePool(svc); err != nil {
			return err
		}
	}

	if d.UpdateConfig.NodeCount != 0 {
		logrus.Infof("Updating node number to %v", d.UpdateConfig.NodeCount)
		operation, err := svc.Projects.Zones.Clusters.NodePools.SetSize(d.ProjectID, d.Zone, d.Name, d.NodePoolID, &raw.SetNodePoolSizeRequest{
			NodeCount: d.UpdateConfig.NodeCount,
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
	request.Cluster.InitialNodeCount = d.InitialNodeCount
	request.Cluster.ClusterIpv4Cidr = d.ClusterIpv4Cidr
	request.Cluster.Description = d.Description
	request.Cluster.EnableKubernetesAlpha = d.EnableAlphaFeature
	request.Cluster.MasterAuth = &raw.MasterAuth{
		Username: "admin",
	}
	request.Cluster.NodeConfig = d.NodeConfig
	return &request
}

func (d *Driver) Get(request *generic.ClusterGetRequest) (*generic.ClusterInfo, error) {
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
	info.Metadata["gke-credential-path"] = d.CredentialPath
	info.Metadata["nodePool"] = cluster.NodePools[0].Name
	serviceAccountToken, err := generateServiceAccountToken(cluster)
	if err != nil {
		return nil, err
	}
	info.ServiceAccountToken = serviceAccountToken

	return info, nil
}

func (d *Driver) Remove() error {
	svc, err := d.getServiceClient()
	if err != nil {
		return err
	}
	logrus.Debugf("Removing cluster %v from project %v, zone %v", d.Name, d.ProjectID, d.Zone)
	operation, err := svc.Projects.Zones.Clusters.Delete(d.ProjectID, d.Zone, d.Name).Context(context.Background()).Do()
	if err != nil {
		return err
	}
	logrus.Debugf("Cluster %v delete is called. Status Code %v", d.Name, operation.HTTPStatusCode)
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

// todo: this function might be generic to all the drivers
func generateServiceAccountToken(cluster *raw.Cluster) (string, error) {
	ts, err := google.DefaultTokenSource(context.Background(), raw.CloudPlatformScope)
	if err != nil {
		return "", err
	}
	token, err := ts.Token()
	if err != nil {
		return "", err
	}
	capem, err := base64.StdEncoding.DecodeString(cluster.MasterAuth.ClusterCaCertificate)
	if err != nil {
		return "", err
	}
	certpem, err := base64.StdEncoding.DecodeString(cluster.MasterAuth.ClientCertificate)
	if err != nil {
		return "", err
	}
	keypem, err := base64.StdEncoding.DecodeString(cluster.MasterAuth.ClientKey)
	if err != nil {
		return "", err
	}
	config := &rest.Config{
		Host: fmt.Sprintf("https://%s", cluster.Endpoint),
		TLSClientConfig: rest.TLSClientConfig{
			CAData:   capem,
			CertData: certpem,
			KeyData:  keypem,
		},
		AuthProvider: &api.AuthProviderConfig{
			Name: "gcp",
			Config: map[string]string{
				"access-token": token.AccessToken,
			},
		},
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", err
	}
	serviceAccount := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: netesDefault,
		},
	}

	_, err = clientset.CoreV1().ServiceAccounts(defaultNamespace).Create(serviceAccount)
	if err != nil && !errors.IsAlreadyExists(err) {
		return "", err
	}

	clusterAdminRole, err := clientset.RbacV1beta1().ClusterRoles().Get(clusterAdmin, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	clusterRoleBinding := &v1beta1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "netes-default-clusterRoleBinding",
		},
		Subjects: []v1beta1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      serviceAccount.Name,
				Namespace: "default",
				APIGroup:  v1.GroupName,
			},
		},
		RoleRef: v1beta1.RoleRef{
			Kind:     "ClusterRole",
			Name:     clusterAdminRole.Name,
			APIGroup: v1beta1.GroupName,
		},
	}
	if _, err = clientset.RbacV1beta1().ClusterRoleBindings().Create(clusterRoleBinding); err != nil && !errors.IsAlreadyExists(err) {
		return "", err
	}

	if serviceAccount, err = clientset.CoreV1().ServiceAccounts(defaultNamespace).Get(serviceAccount.Name, metav1.GetOptions{}); err != nil {
		return "", err
	}

	if len(serviceAccount.Secrets) > 0 {
		secret := serviceAccount.Secrets[0]
		secretObj, err := clientset.CoreV1().Secrets(defaultNamespace).Get(secret.Name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		if token, ok := secretObj.Data["token"]; ok {
			return string(token), nil
		}
	}
	return "", fmt.Errorf("failed to configure serviceAccountToken for cluster name %v", cluster.Name)
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
