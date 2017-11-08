package cmd

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"

	"path/filepath"

	"github.com/rancher/kontainer-engine/cluster"
	rpcDriver "github.com/rancher/kontainer-engine/driver"
	"github.com/rancher/kontainer-engine/utils"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

const (
	caPem             = "ca.pem"
	clientKey         = "key.pem"
	clientCert        = "cert.pem"
	defaultConfigName = "config.json"
)

var GlobalFlag = []cli.Flag{
	cli.BoolFlag{
		Name:  "debug",
		Usage: "Enable verbose logging",
	},
	cli.StringFlag{
		Name:  "plugin-listen-addr",
		Usage: "The listening address for rpc plugin server",
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
	rpcClient, addr, err := runRPCDriver(driverName)
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
	if len(os.Args) > 1 && addr != "" {
		args := append(os.Args[0:len(os.Args)-1], "--plugin-listen-addr", addr, os.Args[len(os.Args)-1])
		return ctx.App.Run(args)
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

type cliConfigGetter struct {
	name string
	ctx  *cli.Context
}

func (c cliConfigGetter) GetConfig() (rpcDriver.DriverOptions, error) {
	driverOpts :=  getDriverOpts(c.ctx)
	driverOpts.StringOptions["name"] = c.name
	return driverOpts, nil
}

type cliPersistStore struct{}

func (c cliPersistStore) Check(name string) (bool, error) {
	path := filepath.Join(utils.HomeDir(), "clusters", name)
	if _, err := os.Stat(filepath.Join(path, defaultConfigName)); os.IsNotExist(err) {
		return false, nil
	}
	return true, nil
}

func (c cliPersistStore) Store(cls cluster.Cluster) error {
	// store kube config file
	if err := storeConfig(cls); err != nil {
		return err
	}
	// store json config file
	fileDir := filepath.Join(utils.HomeDir(), "clusters", cls.Name)
	for k, v := range map[string]string{
		cls.RootCACert:        caPem,
		cls.ClientKey:         clientKey,
		cls.ClientCertificate: clientCert,
	} {
		data, err := base64.StdEncoding.DecodeString(k)
		if err != nil {
			return err
		}
		if err := utils.WriteToFile(data, filepath.Join(fileDir, v)); err != nil {
			return err
		}
	}
	data, err := json.Marshal(cls)
	if err != nil {
		return err
	}
	return utils.WriteToFile(data, filepath.Join(fileDir, defaultConfigName))
}

func create(ctx *cli.Context) error {
	driverName := ctx.String("driver")
	if driverName == "" {
		logrus.Error("Driver name is required")
		return cli.ShowCommandHelp(ctx, "create")
	}
	addr := ctx.String("plugin-listen-addr")
	name := ""
	if ctx.NArg() > 0 {
		name = ctx.Args().Get(0)
	}
	configGetter := cliConfigGetter{
		name: name,
		ctx:  ctx,
	}
	persistStore := cliPersistStore{}
	cls, err := cluster.NewCluster(driverName, addr, name, configGetter, persistStore)
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

func getDriverFlags(opts rpcDriver.DriverFlags) []cli.Flag {
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
