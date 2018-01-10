package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/rancher/kontainer-engine/service"
	"github.com/rancher/kontainer-engine/store"
	"github.com/rancher/types/apis/management.cattle.io/v3"
	"github.com/sirupsen/logrus"
)

func main() {
	time.Sleep(time.Second * 2)
	credentialPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	data, err := ioutil.ReadFile(credentialPath)
	if err != nil {
		logrus.Fatal(err)
	}
	gkeSpec := &v3.GoogleKubernetesEngineConfig{
		ProjectID:           "rancher-dev",
		Zone:                "us-central1-a",
		NodeCount:           1,
		KubernetesDashboard: true,
		HTTPLoadBalancing:   true,
		ImageType:           "ubuntu",
		LegacyAbac:          true,
		Locations:           []string{"us-central1-a", "us-central1-b"},
		Credential:          string(data),
	}
	spec := v3.ClusterSpec{
		GoogleKubernetesEngineConfig: gkeSpec,
	}

	// You should really implement your own store
	store := store.CLIPersistStore{}
	service := service.NewEngineService(store)

	endpoint, token, cert, err := service.Create("daishan-test", spec)
	if err != nil {
		logrus.Fatal(err)
	}
	fmt.Println(endpoint)
	fmt.Println(token)
	fmt.Println(cert)
	err = service.Remove("daishan-test", spec)
	if err != nil {
		logrus.Fatal(err)
	}
}
