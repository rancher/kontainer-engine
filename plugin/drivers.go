package plugin

import (
	rpcDriver "github.com/rancher/kontainer-engine/driver"
	"github.com/rancher/kontainer-engine/driver/gke"
)

var (
	BuiltInDrivers = map[string]bool{
		"gke": true,
		"aks": true,
	}
)

func Run(driverName string, addrChan chan string) error {
	var driver rpcDriver.Driver
	switch driverName {
	case "gke":
		driver = gke.NewDriver()
	default:
		addrChan <- ""
	}
	if BuiltInDrivers[driverName] {
		go startRPCServer(rpcDriver.NewServer(driver, addrChan))
	}
	return nil
}

func startRPCServer(server rpcDriver.RPCServer) {
	server.Serve()
}
