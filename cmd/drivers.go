package cmd

import (
	"github.com/rancher/netes-machine/driver/gke"
	"github.com/sirupsen/logrus"
	generic "github.com/rancher/netes-machine/driver"
)

func runDriver(driverName string) error {
	errStop := make(chan error)
	switch driverName {
	case "gke":
		gkeDriver := gke.NewDriver()
		// todo: change it to be not hard-coded
		addr := "127.0.0.1:9001"
		go startRpcServer(gke.NewGkeRpcServer(gkeDriver, addr), errStop)
	}
	return nil
}

func startRpcServer(server generic.RpcServer, errStop chan error) {
	server.Serve(errStop)
	err := <-errStop
	if err != nil {
		logrus.Fatal(err)
	}
}
