package cmd

import (
	generic "github.com/rancher/kontainer-engine/driver"
	"github.com/rancher/kontainer-engine/driver/gke"
)

func runDriver(driverName string, addrChan chan string) error {
	switch driverName {
	case "gke":
		gkeDriver := gke.NewDriver()
		go startRPCServer(gke.NewGkeRPCServer(gkeDriver, addrChan))
	default:
		addrChan <- ""
	}
	return nil
}

func startRPCServer(server generic.RPCServer) {
	server.Serve()
}
