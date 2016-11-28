# InfraKit Instance Plugin for 1&amp;1 Cloud Server

[![Build Status](https://travis-ci.org/1and1/infrakit-instance-oneandone.svg?branch=master)](https://travis-ci.org/1and1/infrakit-instance-oneandone)

This is an [InfraKit](https://github.com/docker/infrakit) instance plugin for creating and managing 1&amp;1 Cloud servers.
The plugin development is ongoing and attempts to follow changes and new features in the InfraKit itself.

## Usage

Before you start, make sure you have installed [Go](https://golang.org/). Follow the [steps](https://github.com/docker/infrakit#building) to obtain InfraKit code and make [binaries](https://github.com/docker/infrakit/blob/master/README.md#binaries).

Currently, you can use this plugin with a plain [vanilla flavor plugin](https://github.com/docker/infrakit/tree/master/pkg/example/flavor/vanilla) and the [default group plugin](https://github.com/docker/infrakit/blob/master/cmd/group/README.md).

Get the plugin source code:

```
go get github.com/1and1/infrakit-instance-oneandone
```

To build the 1&amp;1 Instance plugin, run `make binary` in the plugin repository root.  The binary will be located at `./build/infrakit-instance-oneandone`.

Use the help command to list the command line options available with the plugin.

```shell
$ build/infrakit-instance-oneandone --help
1&1 Cloud Server instance plugin

Usage:
  infrakit-instance-oneandone [flags]
  infrakit-instance-oneandone [command]

Available Commands:
  version     print build version information

Flags:
      --api-key string   1&1 API access key
      --dir string       Existing directory for storing the plugin files (default "/tmp")
      --log int          Logging level. 0 is least verbose. Max is 5 (default 4)
      --name string      Plugin name to advertise for discovery (default "instance-1and1")

Use "infrakit-instance-oneandone [command] --help" for more information about a command.
```

Run the plugin:

```shell
$ build/infrakit-instance-oneandone --dir ./
INFO[0000] oneandone instance plugin. dir=./
INFO[0000] Listening at: /home/nb/.infrakit/plugins/instance-1and1
```

Note that `--api-key` is required, if you do not provide the key with `ONEANDONE_API_KEY` environment variable.

From the InfraKit build directory run:

```shell
$ build/infrakit-group-default
```

```shell
$ build/infrakit-flavor-vanilla
```

Use the provided configuration example [example1and1.json](./example1and1.json) as a reference and feel free to change 
the values of the properties.

```shell
$ cat << EOF > 1and1.json
{
	"ID": "myGroup",
	"Properties": {
		"Allocation": {
			"Size": 2
		},
		"Instance": {
			"Plugin": "instance-1and1",
			"Properties": {
				"Appliance": "centos7-64std",
				"Datacenter": "GB",
				"FixedServerSize": "L",
				"Password": "",
				"SSHKey": "",
				"SSHKeyPath": "/home/nb/.ssh/id_rsa.pub",
				"FirewallID": "",
				"LoadBalancerID": "",
				"MonitorPolicyID": ""
			}
		},
		"Flavor": {
			"Plugin": "flavor-vanilla",
			"Properties": {
				"Tags": {
					"project": "test"
				}
			}
		}
	}
}
EOF
```

Commit the configuration by running the InfraKit command:

```shell
$ build/infrakit group commit 1and1.json
Committed myGroup: Managing 2 instances
```

### Managing groups

You can use the set of InfraKit commands to manage the groups being monitored.

```
$ build/infrakit group --help
Access group plugin

Usage:
  build/infrakit group [command]

Available Commands:
  commit      commit a group configuration
  describe    describe the live instances that make up a group
  destroy     destroy a group
  free        free a group from active monitoring, nondestructive
  inspect     return the raw configuration associated with a group
  ls          list groups

Flags:
      --name string   Name of plugin (default "group")

Global Flags:
      --log int   Logging level. 0 is least verbose. Max is 5 (default 4)

Use "build/infrakit group [command] --help" for more information about a command.
```

Describe group command displays info about the instances in a tabular form.

```
$ build/infrakit group describe myGroup 
ID                            	LOGICAL                       	TAGS
instance-3596758581874575275  	77.68.13.219                  	infrakit.config_sha=Cf4KpiVHyBBRTVR9XqOubrKB6kU=,infrakit.group=myGroup,project=test,serverID=FD7697A0FCE6E92DA9066BF8A53B1B12
instance-4696994370972658643  	77.68.15.106                  	infrakit.config_sha=Cf4KpiVHyBBRTVR9XqOubrKB6kU=,infrakit.group=myGroup,project=test,serverID=0ACC0C790843C1213F8B528A5985710C
```

If you would like to increase or decrease the number of instances in the group, modify the allocation `Size` property and commit the modified configuration.

```
$ build/infrakit group commit 1and1.json 
Committed myGroup: Adding 1 instances to increase the group size to 3
```
```
$ build/infrakit group commit 1and1.json 
Committed myGroup: Terminating 2 instances to reduce the group size to 1
```

Run destroy command to terminate a group monitoring and delete all instances, i.e., servers in the group.

```
$ build/infrakit group destroy myGroup
destroy myGroup initiated
```

## Configuration parameters

* **Appliance** -  1&amp;1 server appliance ID or name.
* **Datacenter** - 1&amp;1 data center ID or country code.
* **FixedServerSize** - 1&amp;1 fixed-instance size used for the server.
* **Password** - Password for the server.
* **SSHKey** - SSH public key.
* **SSHKeyPath** - Path to public SSH key. 
* **FirewallID** - 1&amp;1 firewall policy ID.
* **LoadBalancerID** - 1&amp;1 load balancer ID.
* **MonitorPolicyID** - 1&amp;1 monitoring policy ID.

The required parameters are `Appliance` and `FixedServerSize`.

## Design notes

The plugin stores a basic info about the instances onto the provided location (`--dir`). You can stop and start a group monitoring without redeploying the servers.

The instance file names consist of the instance (server) name and `.1and1.spec` extension.
The numeric part if the file name (also server name) is generated randomly to avoid naming conflict when a group is scaled automatically.

Default values are provided for the following instance properties:

* **Appliance** => `ubuntu1404-64std`
* **Datacenter** => `US`
* **FixedServerSize** => `M`

Either ID or name can be specified for `Appliance` and `FixedServerSize`, and ID or country code for `Datacenter`.

If `SSHKey` is provided, `SSHKeyPath` will be ignored.
