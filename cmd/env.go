package cmd

import (
	"fmt"

	"github.com/rancher/kontainer-engine/store"
	"github.com/urfave/cli"
)

func EnvCommand() cli.Command {
	return cli.Command{
		Name:   "env",
		Usage:  "Show the environment variable(KUBECONFIG) to export",
		Action: env,
	}
}

func env(ctx *cli.Context) error {
	name := ctx.Args().Get(0)
	if name == "" || name == "--help" {
		return cli.ShowCommandHelp(ctx, "env")
	}

	clusters, err := store.GetAllClusterFromStore()
	if err != nil {
		return err
	}
	cluster, ok := clusters[name]
	if !ok {
		return fmt.Errorf("cluster %v can't be found", name)
	}
	return cluster.GenerateConfig()
}
