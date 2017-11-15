package stub

import (
	"fmt"
	"testing"

	rpcDriver "github.com/rancher/kontainer-engine/driver"
	"github.com/rancher/types/apis/cluster.cattle.io/v1"
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
