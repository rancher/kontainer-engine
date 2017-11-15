package cmd

import (
	"errors"
	"fmt"

	"os"

	generic "github.com/rancher/kontainer-engine/driver"
	"github.com/rancher/kontainer-engine/store"
	"github.com/urfave/cli"
)

// UpdateCommand defines the update command
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
	rpcClient, addr, err := runRPCDriver(cluster.DriverName)
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
			updateeCmd.Flags = append(globalFlag, append(updateeCmd.Flags, flags...)...)
			updateeCmd.Action = updateCluster
		}
	}
	if len(os.Args) > 1 && addr != "" {
		args := append(os.Args[0:len(os.Args)-1], "--plugin-listen-addr", addr, os.Args[len(os.Args)-1])
		return ctx.App.Run(args)
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
	addr := ctx.String("plugin-listen-addr")
	rpcClient, err := generic.NewClient(cluster.DriverName, addr)
	if err != nil {
		return err
	}
	configGetter := cliConfigGetter{
		name: name,
		ctx:  ctx,
	}
	cluster.ConfigGetter = configGetter
	cluster.PersistStore = cliPersistStore{}
	cluster.Driver = rpcClient
	return cluster.Update()
}
