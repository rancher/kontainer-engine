package cmd

import (
	generic "github.com/rancher/kontainer-engine/driver"
	"github.com/rancher/kontainer-engine/driver/gke"
)

func runDriver(driverName string) error {
	switch driverName {
	case "gke":
		gkeDriver := gke.NewDriver()
		// todo: change it to be not hard-coded
		addr := "127.0.0.1:9001"
		go startRPCServer(gke.NewGkeRPCServer(gkeDriver, addr))
	}
	return nil
}

func startRPCServer(server generic.RPCServer) {
	server.Serve()
}
