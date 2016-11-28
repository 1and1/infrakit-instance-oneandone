package main

import (
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/infrakit/pkg/spi/instance"
	"github.com/spf13/pflag"
)

type options struct {
	apiKey     string
	dir        string
	datacenter string
	appliance  string
	fixedSize  string
}

// Builder is a ProvisionerBuilder that creates an 1&1 Cloud instance provisioner.
type Builder struct {
	options options
}

// Flags returns the flags required.
func (b *Builder) Flags() *pflag.FlagSet {
	flags := pflag.NewFlagSet("oneandone", pflag.PanicOnError)
	flags.StringVar(&b.options.apiKey, "api-key", "", "1&1 API access key")
	flags.StringVar(&b.options.dir, "dir", os.TempDir(), "Existing directory for storing the plugin files")
	return flags
}

// BuildInstancePlugin creates an instance Provisioner configured with the Flags.
func (b *Builder) BuildInstancePlugin() (instance.Plugin, error) {
	apiKey := b.options.apiKey
	var ok bool

	if len(apiKey) == 0 {
		if apiKey, ok = os.LookupEnv("ONEANDONE_API_KEY"); !ok {
			log.Fatal("1&1 API access key not found. Please set api-key CLI option or ONEANDONE_API_KEY environment variable.")
		}
	}

	return NewInstancePlugin(apiKey, b.options.dir), nil
}
