package main

import (
	"encoding/json"
	"fmt"
	"os"

	log "github.com/Sirupsen/logrus"
	//	"github.com/StackPointCloud/infrakit-instance-oneandone"
	"github.com/docker/infrakit/pkg/cli"
	instance_plugin "github.com/docker/infrakit/pkg/rpc/instance"
	"github.com/spf13/cobra"
)

var (
	// Version is the build release identifier.
	Version = "Unspecified"

	// Revision is the build source control revision.
	Revision = "Unspecified"
)

func main() {
	builder := &Builder{}

	var logLevel int
	var name string
	cmd := &cobra.Command{
		Use:   os.Args[0],
		Short: "1&1 Cloud Server instance plugin",
		Run: func(c *cobra.Command, args []string) {

			instancePlugin, err := builder.BuildInstancePlugin()
			if err != nil {
				log.Error(err)
				os.Exit(1)
			}

			cli.SetLogLevel(logLevel)
			cli.RunPlugin(name, instance_plugin.PluginServer(instancePlugin))
		},
	}

	cmd.Flags().IntVar(&logLevel, "log", cli.DefaultLogLevel, "Logging level. 0 is least verbose. Max is 5")
	cmd.Flags().StringVar(&name, "name", "instance-1and1", "Plugin name to advertise for discovery")

	cmd.Flags().AddFlagSet(builder.Flags())

	cmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "print build version information",
		RunE: func(c *cobra.Command, args []string) error {
			buff, err := json.MarshalIndent(map[string]interface{}{
				"name":     name,
				"version":  Version,
				"revision": Revision,
			}, "  ", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(buff))
			return nil
		},
	})

	err := cmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
