package cmd

import (
	"os"
	"strings"

	"github.com/rancher/netes-machine/cluster"
	generic "github.com/rancher/netes-machine/driver"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var GlobalFlag = []cli.Flag{
	cli.BoolFlag{
		Name:  "debug",
		Usage: "Enable verbose logging",
	},
}

func CreateCommand() cli.Command {
	return cli.Command{
		Name:            "create",
		Usage:           "Create a kubernetes cluster",
		Action:          createWapper,
		SkipFlagParsing: true,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "driver",
				Usage: "Driver to create kubernetes clusters",
			},
		},
	}
}

func createWapper(ctx *cli.Context) error {
	debug := lookUpDebugFlag()
	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	driverName := flagHackLookup("--driver")
	if driverName == "" {
		logrus.Error("Driver name is required")
		return cli.ShowCommandHelp(ctx, "create")
	}
	runDriver(driverName)

	rpcClient, err := generic.NewRPCClient(driverName)
	if err != nil {
		return err
	}
	driverFlags, err := rpcClient.GetDriverCreateOptions()
	if err != nil {
		return err
	}
	flags := getDriverFlags(driverFlags)
	for i, command := range ctx.App.Commands {
		if command.Name == "create" {
			createCmd := &ctx.App.Commands[i]
			createCmd.SkipFlagParsing = false
			createCmd.Flags = append(GlobalFlag, append(createCmd.Flags, flags...)...)
			createCmd.Action = create
		}
	}
	return ctx.App.Run(os.Args)
}

func flagHackLookup(flagName string) string {
	// e.g. "-d" for "--driver"
	flagPrefix := flagName[1:3]

	// TODO: Should we support -flag-name (single hyphen) syntax as well?
	for i, arg := range os.Args {
		if strings.Contains(arg, flagPrefix) {
			// format '--driver foo' or '-d foo'
			if arg == flagPrefix || arg == flagName {
				if i+1 < len(os.Args) {
					return os.Args[i+1]
				}
			}

			// format '--driver=foo' or '-d=foo'
			if strings.HasPrefix(arg, flagPrefix+"=") || strings.HasPrefix(arg, flagName+"=") {
				return strings.Split(arg, "=")[1]
			}
		}
	}

	return ""
}

func create(ctx *cli.Context) error {
	driverName := ctx.String("driver")
	if driverName == "" {
		logrus.Error("Driver name is required")
		return cli.ShowCommandHelp(ctx, "create")
	}
	cls, err := cluster.NewCluster(driverName, ctx)
	if err != nil {
		return err
	}
	if cls.Name == "" {
		logrus.Error("Cluster name is required")
		return cli.ShowCommandHelp(ctx, "create")
	}
	return cls.Create()
}

func lookUpDebugFlag() bool {
	for _, arg := range os.Args {
		if arg == "--debug" {
			return true
		}
	}
	return false
}

func getDriverFlags(opts generic.DriverFlags) []cli.Flag {
	flags := []cli.Flag{}
	for k, v := range opts.Options {
		switch v.Type {
		case "int":
			flags = append(flags, cli.Int64Flag{
				Name:  k,
				Usage: v.Usage,
			})
		case "string":
			flags = append(flags, cli.StringFlag{
				Name:  k,
				Usage: v.Usage,
			})
		case "stringSlice":
			flags = append(flags, cli.StringSliceFlag{
				Name:  k,
				Usage: v.Usage,
			})
		case "bool":
			flags = append(flags, cli.BoolFlag{
				Name:  k,
				Usage: v.Usage,
			})
		}
	}
	return flags
}
