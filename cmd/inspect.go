package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rancher/kontainer-engine/store"
	"github.com/urfave/cli"
	"os"
)

func InspectCommand() cli.Command {
	return cli.Command{
		Name:   "inspect",
		Usage:  "inspect kubernetes clusters",
		Action: inspectCluster,
		Flags:  []cli.Flag{},
	}
}

func inspectCluster(ctx *cli.Context) error {
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
	cluster.ClientKey = "Redacted"
	cluster.ClientCertificate = "Redacted"
	cluster.RootCACert = "Redacted"
	data, err := json.MarshalIndent(cluster, "", "\t")
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(data)
	return err
}
