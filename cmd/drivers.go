package cmd

import (
	rpcDriver "github.com/rancher/kontainer-engine/driver"
	"github.com/rancher/kontainer-engine/driver/gke"
)

func runDriver(driverName string, addrChan chan string) error {
	switch driverName {
	case "gke":
		gkeDriver := gke.NewDriver()
		go startRPCServer(rpcDriver.NewServer(gkeDriver, addrChan))
	default:
		addrChan <- ""
	}
	return nil
}

func startRPCServer(server rpcDriver.RPCServer) {
	server.Serve()
}
