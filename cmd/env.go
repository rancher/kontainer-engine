package cmd

import (
	"fmt"

	"github.com/rancher/kontainer-engine/store"
	"github.com/rancher/kontainer-engine/utils"
	"github.com/urfave/cli"
)

// EnvCommand defines the env command
func EnvCommand() cli.Command {
	return cli.Command{
		Name:   "env",
		Usage:  "Set cluster as current context",
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
	_, ok := clusters[name]
	if !ok {
		return fmt.Errorf("cluster %v can't be found", name)
	}
	config, err := getConfigFromFile()
	if err != nil {
		return err
	}
	config.CurrentContext = name
	if err := setConfigToFile(config); err != nil {
		return err
	}

	configFile := utils.KubeConfigFilePath()
	fmt.Printf("Current context is set to %s\n", name)
	fmt.Printf("run `export KUBECONFIG=%v` or `--kubeconfig %s` to use the config file\n", configFile, configFile)
	return nil
}
