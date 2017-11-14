package stub

import (
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	rpcDriver "github.com/rancher/kontainer-engine/driver"
	"github.com/rancher/types/apis/cluster.cattle.io/v1"
	"gopkg.in/check.v1"
	"gopkg.in/yaml.v2"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) {
	check.TestingT(t)
}

type StubTestSuite struct {
}

var _ = check.Suite(&StubTestSuite{})

func (s *StubTestSuite) SetUpSuite(c *check.C) {
}

func (s *StubTestSuite) TestFlatten(c *check.C) {
	config := v1.GoogleKubernetesEngineConfig{
		ProjectID:  "test",
		Zone:       "test",
		DiskSizeGb: 50,
		Labels: map[string]string{
			"foo": "bar",
		},
		EnableAlphaFeature: true,
	}
	config.MasterVersion = "1.7.1"
	config.NodeVersion = "1.7.1"
	config.NodeCount = 3
	opts, err := toMap(config, "json")
	if err != nil {
		c.Fatal(err)
	}
	driverOptions := rpcDriver.DriverOptions{
		BoolOptions:        make(map[string]bool),
		StringOptions:      make(map[string]string),
		IntOptions:         make(map[string]int64),
		StringSliceOptions: make(map[string]*rpcDriver.StringSlice),
	}
	flatten(opts, &driverOptions)
	fmt.Println(driverOptions)
	boolResult := map[string]bool{
		"enableAlphaFeature": true,
	}
	stringResult := map[string]string{
		"projectId":     "test",
		"zone":          "test",
		"masterVersion": "1.7.1",
		"nodeVersion":   "1.7.1",
	}
	intResult := map[string]int64{
		"diskSizeGb": 50,
		"nodeCount":  3,
	}
	stringSliceResult := map[string]rpcDriver.StringSlice{
		"labels": {
			Value: []string{"foo=bar"},
		},
	}
	c.Assert(driverOptions.BoolOptions, check.DeepEquals, boolResult)
	c.Assert(driverOptions.IntOptions, check.DeepEquals, intResult)
	c.Assert(driverOptions.StringOptions, check.DeepEquals, stringResult)
	c.Assert(driverOptions.StringSliceOptions["labels"].Value, check.DeepEquals, stringSliceResult["labels"].Value)
}

func (s *StubTestSuite) unTestCreate(c *check.C) {
	time.Sleep(time.Second)
	config := v1.GoogleKubernetesEngineConfig{
		ProjectID:  "rancher-dev",
		Zone:       "us-central1-a",
		DiskSizeGb: 50,
		Labels: map[string]string{
			"foo": "bar",
		},
		MasterVersion: "1.8.2-gke.0",

		CredentialPath: "/Users/daishanpeng/Documents/gke/key.json",
	}
	cluster := v1.Cluster{}
	cluster.Spec.GoogleKubernetesEngineConfig = &config
	endpoint, serviceAccountToken, cacert, err := Create("daishan-test1", cluster.Spec)
	if err != nil {
		c.Fatal(err)
	}
	fmt.Println(endpoint)
	fmt.Println(cacert)
	fmt.Println(serviceAccountToken)
	c.Fatal("hello")
}

func (s *StubTestSuite) unTestUpdate(c *check.C) {
	time.Sleep(time.Second)
	config := v1.GoogleKubernetesEngineConfig{
		ProjectID:  "rancher-dev",
		Zone:       "us-central1-a",
		DiskSizeGb: 50,
		Labels: map[string]string{
			"foo": "bar",
		},
		EnableAlphaFeature: true,
		CredentialPath:     "/Users/daishanpeng/Documents/gke/key.json",
	}
	config.NodeCount = 4
	cluster := v1.Cluster{}
	cluster.Name = "daishan-test"
	cluster.Spec.GoogleKubernetesEngineConfig = &config
	_, _, _, err := Update(cluster)
	if err != nil {
		c.Fatal(err)
	}
}

func (s *StubTestSuite) unTestRemove(c *check.C) {
	time.Sleep(time.Second)
	config := v1.GoogleKubernetesEngineConfig{
		ProjectID:  "rancher-dev",
		Zone:       "us-central1-a",
		DiskSizeGb: 50,
		Labels: map[string]string{
			"foo": "bar",
		},
		EnableAlphaFeature: true,
		CredentialPath:     "/Users/daishanpeng/Documents/gke/key.json",
	}
	cluster := v1.Cluster{}
	cluster.Name = "daishan-test"
	cluster.Spec.GoogleKubernetesEngineConfig = &config
	err := Remove(cluster)
	if err != nil {
		c.Fatal(err)
	}
}

func (s *StubTestSuite) unTestRkeCreate(c *check.C) {
	configYaml, err := ioutil.ReadFile("/Users/daishanpeng/rke.yaml")
	if err != nil {
		c.Fatal(err)
	}
	cluster := v1.Cluster{}
	cluster.Name = "daishan-test"
	rkeConfig := v1.RancherKubernetesEngineConfig{}
	if err := yaml.Unmarshal(configYaml, &rkeConfig); err != nil {
		c.Fatal(err)
	}
	cluster.Spec.RancherKubernetesEngineConfig = &rkeConfig
	fmt.Printf("%+v", rkeConfig)
	endpoint, serviceAccountToken, cacert, err := Create("daishan-test", cluster.Spec)
	if err != nil {
		c.Fatal(err)
	}
	fmt.Println(endpoint)
	fmt.Println(serviceAccountToken)
	fmt.Println(cacert)
	c.Fatal("hello")
}

func (s *StubTestSuite) unTestRkeUpdate(c *check.C) {
	configYaml, err := ioutil.ReadFile("/Users/daishanpeng/rke.yaml")
	if err != nil {
		c.Fatal(err)
	}
	cluster := v1.Cluster{}
	cluster.Name = "daishan-test"
	rkeConfig := v1.RancherKubernetesEngineConfig{}
	if err := yaml.Unmarshal(configYaml, &rkeConfig); err != nil {
		c.Fatal(err)
	}
	cluster.Spec.RancherKubernetesEngineConfig = &rkeConfig
	fmt.Printf("%+v", rkeConfig)
	endpoint, serviceAccountToken, cacert, err := Update(cluster)
	if err != nil {
		c.Fatal(err)
	}
	fmt.Println(endpoint)
	fmt.Println(serviceAccountToken)
	fmt.Println(cacert)
	c.Fatal("hello")
}
