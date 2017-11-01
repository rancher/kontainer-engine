package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rancher/kontainer-engine/store"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func RmCommand() cli.Command {
	return cli.Command{
		Name:      "remove",
		ShortName: "rm",
		Usage:     "Remove kubernetes clusters",
		Action:    rmCluster,
	}
}

func rmCluster(ctx *cli.Context) error {
	name := ctx.Args().Get(0)
	if name == "" || name == "--help" {
		return cli.ShowCommandHelp(ctx, "remove")
	}
	clusters, err := store.GetAllClusterFromStore()
	if err != nil {
		return err
	}
	cluster, ok := clusters[name]
	if !ok {
		return fmt.Errorf("cluster %v can't be found", name)
	}
	rpcClient, _, err := runRPCDriver(cluster.DriverName)
	if err != nil {
		return err
	}
	cluster.Driver = rpcClient
	if err := cluster.Remove(); err != nil {
		return err
	}
	// todo: interface the storage level
	clusterFilePath := filepath.Join(cluster.GetFileDir(), cluster.Name)
	logrus.Debugf("Deleting cluster storage path %v", clusterFilePath)
	if err := os.RemoveAll(clusterFilePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	fmt.Println(cluster.Name)
	return nil
}
