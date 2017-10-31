package cmd

import (
	"errors"
	"fmt"

	"os"

	generic "github.com/rancher/netes-machine/driver"
	"github.com/rancher/netes-machine/store"
	"github.com/urfave/cli"
)

func UpdateCommand() cli.Command {
	return cli.Command{
		Name:            "update",
		Usage:           "update kubernetes clusters",
		Action:          updateWrapper,
		SkipFlagParsing: true,
	}
}

func updateWrapper(ctx *cli.Context) error {
	name := ctx.Args().Get(len(ctx.Args()) - 1)
	if name == "" || name == "--help" {
		return cli.ShowCommandHelp(ctx, "update")
	}
	clusters, err := store.GetAllClusterFromStore()
	if err != nil {
		return err
	}
	cluster, ok := clusters[name]
	if !ok {
		return fmt.Errorf("cluster %v can't be found", name)
	}
	runDriver(cluster.DriverName)

	rpcClient, err := generic.NewRPCClient(cluster.DriverName)
	if err != nil {
		return err
	}
	driverFlags, err := rpcClient.GetDriverUpdateOptions()
	if err != nil {
		return err
	}

	flags := getDriverFlags(driverFlags)
	for i, command := range ctx.App.Commands {
		if command.Name == "update" {
			updateeCmd := &ctx.App.Commands[i]
			updateeCmd.SkipFlagParsing = false
			updateeCmd.Flags = append(GlobalFlag, append(updateeCmd.Flags, flags...)...)
			updateeCmd.Action = updateCluster
		}
	}
	return ctx.App.Run(os.Args)
}

func updateCluster(ctx *cli.Context) error {
	name := ctx.Args().Get(0)
	if name == "" {
		return errors.New("name is required when inspecting cluster")
	}
	clusters, err := store.GetAllClusterFromStore()
	if err != nil {
		return err
	}
	cluster, ok := clusters[name]
	if !ok {
		return fmt.Errorf("cluster %v can't be found", name)
	}
	rpcClient, err := generic.NewRPCClient(cluster.DriverName)
	if err != nil {
		return err
	}
	cluster.Ctx = ctx
	cluster.Driver = rpcClient
	return cluster.Update()
}
