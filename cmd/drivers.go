package cmd

import (
	rpcDriver "github.com/rancher/kontainer-engine/driver"
	"github.com/rancher/kontainer-engine/driver/gke"
)

var (
	builtInDrivers = map[string]bool{
		"gke": true,
		"aks": true,
	}
)

func runDriver(driverName string, addrChan chan string) error {
	var driver rpcDriver.Driver
	switch driverName {
	case "gke":
		driver = gke.NewDriver()
	default:
		addrChan <- ""
	}
	if builtInDrivers[driverName] {
		go startRPCServer(rpcDriver.NewServer(driver, addrChan))
	}
	return nil
}

func startRPCServer(server rpcDriver.RPCServer) {
	server.Serve()
}
