package stub

import (
	"testing"

	"fmt"
	"time"

	"github.com/alena1108/cluster-controller/client/v1"
	rpcDriver "github.com/rancher/kontainer-engine/driver"
	"gopkg.in/check.v1"
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
	config := v1.GKEConfig{
		ProjectID:  "test",
		Zone:       "test",
		DiskSizeGb: 50,
		Labels: map[string]string{
			"foo": "bar",
		},
		EnableAlphaFeature: true,
	}
	config.UpdateConfig.MasterVersion = "1.7.1"
	config.UpdateConfig.NodeVersion = "1.7.1"
	config.UpdateConfig.NodeCount = 3
	opts, err := toMap(config)
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
	config := v1.GKEConfig{
		ProjectID:  "rancher-dev",
		Zone:       "us-central1-a",
		DiskSizeGb: 50,
		Labels: map[string]string{
			"foo": "bar",
		},
		EnableAlphaFeature: true,
	}
	cluster := v1.Cluster{}
	cluster.Name = "daishan-test"
	cluster.Spec.GKEConfig = &config
	result, err := Create(cluster)
	if err != nil {
		c.Fatal(err)
	}
	fmt.Println(result)
}

func (s *StubTestSuite) unTestUpdate(c *check.C) {
	time.Sleep(time.Second)
	config := v1.GKEConfig{
		ProjectID:  "rancher-dev",
		Zone:       "us-central1-a",
		DiskSizeGb: 50,
		Labels: map[string]string{
			"foo": "bar",
		},
		EnableAlphaFeature: true,
	}
	config.UpdateConfig.NodeCount = 4
	cluster := v1.Cluster{}
	cluster.Name = "daishan-test"
	cluster.Spec.GKEConfig = &config
	err := Update(cluster)
	if err != nil {
		c.Fatal(err)
	}
}

func (s *StubTestSuite) unTestRemove(c *check.C) {
	time.Sleep(time.Second)
	config := v1.GKEConfig{
		ProjectID:  "rancher-dev",
		Zone:       "us-central1-a",
		DiskSizeGb: 50,
		Labels: map[string]string{
			"foo": "bar",
		},
		EnableAlphaFeature: true,
	}
	cluster := v1.Cluster{}
	cluster.Name = "daishan-test"
	cluster.Spec.GKEConfig = &config
	err := Remove(cluster)
	if err != nil {
		c.Fatal(err)
	}
}
