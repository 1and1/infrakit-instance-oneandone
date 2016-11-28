package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/1and1/oneandone-cloudserver-sdk-go"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/infrakit/pkg/spi/instance"
	"github.com/spf13/afero"
)

// Provisioner is instance provisioner type for 1&1 Cloud Server.
type Provisioner struct {
	client *oneandone.API
	prop   properties
	Dir    string
	fs     afero.Fs
	wait   bool
}

type properties struct {
	apiKey          string
	Datacenter      string
	Appliance       string
	FixedServerSize string
	Password        string
	SSHKey          string
	SSHKeyPath      string
	FirewallID      string
	LoadBalancerID  string
	MonitorPolicyID string
}

// NewInstancePlugin creates a new plugin that creates 1&1 Cloud Server instances.
func NewInstancePlugin(apiKey string, dir string) instance.Plugin {
	log.Infof("oneandone instance plugin. dir=%s", dir)
	return &Provisioner{
		client: oneandone.New(apiKey, oneandone.BaseUrl),
		prop: properties{
			apiKey: apiKey,
		},
		Dir:  dir,
		fs:   afero.NewOsFs(),
		wait: true,
	}
}

func (p *Provisioner) validateAppliance(prop properties) error {
	serverApp := strings.ToLower(prop.Appliance)
	if serverApp == "" {
		serverApp = "ubuntu1404-64std" // Default server appliance
	}

	appliances, err := p.client.ListServerAppliances(0, 0, "", serverApp, "")
	if err != nil {
		return fmt.Errorf("Validating server appliance '%s' failed. Error: %s", serverApp, err.Error())
	}

	for _, sa := range appliances {
		if serverApp == strings.ToLower(sa.Name) || serverApp == strings.ToLower(sa.Id) {
			p.prop.Appliance = sa.Id
		}
	}

	if p.prop.Appliance == "" {
		return fmt.Errorf("Server appliance '%s' could not be found.", serverApp)
	}
	return nil
}

func (p *Provisioner) validateDatacenter(prop properties) error {
	dc := strings.ToUpper(prop.Datacenter)
	if dc == "" {
		dc = "US" // Default data center
	}

	datacenters, err := p.client.ListDatacenters()
	if err != nil {
		return fmt.Errorf("Validating data center '%s' failed. Error: %s", dc, err.Error())
	}

	for _, loc := range datacenters {
		if dc == strings.ToUpper(loc.CountryCode) || dc == strings.ToUpper(loc.Id) {
			p.prop.Datacenter = loc.Id
		}
	}

	if p.prop.Datacenter == "" {
		return fmt.Errorf("Data center '%s' could not be found.", dc)
	}
	return nil
}

func (p *Provisioner) validateServerSize(prop properties) error {
	size := strings.ToUpper(prop.FixedServerSize)
	if size == "" {
		size = "M" // Default server size
	}

	fixSizes, err := p.client.ListFixedInstanceSizes()
	if err != nil {
		return fmt.Errorf("Validating fixed-instance size '%s' failed. Error: %s", size, err.Error())
	}

	for _, s := range fixSizes {
		if size == strings.ToUpper(s.Name) || size == strings.ToUpper(s.Id) {
			p.prop.FixedServerSize = s.Id
		}
	}

	if p.prop.FixedServerSize == "" {
		return fmt.Errorf("Fixed-instance size '%s' could not be found.", size)
	}
	return nil
}

// Validate the provisioner and 1&1 Cloud Server properties
func (p *Provisioner) Validate(req json.RawMessage) error {
	if p.client == nil {
		return fmt.Errorf("1&1 API client is not created. Please use the appropriate methods to instantiate the provisioner.")
	}
	if p.prop.apiKey == "" {
		return fmt.Errorf("1&1 API key could not be found.")
	}

	log.Debugf("Validate : %s", string(req))

	prop := properties{}
	err := json.Unmarshal([]byte(req), &prop)

	if err != nil {
		return fmt.Errorf("Invalid instance properties: %s", err)
	}

	if err = p.validateAppliance(prop); err != nil {
		return err
	}
	if err = p.validateDatacenter(prop); err != nil {
		return err
	}
	if err = p.validateServerSize(prop); err != nil {
		return err
	}

	if prop.SSHKey != "" {
		p.prop.SSHKey = prop.SSHKey
	} else if prop.SSHKeyPath != "" {
		key, err := ioutil.ReadFile(prop.SSHKeyPath)

		if err != nil {
			log.Errorf("Cannot read SSH key from file '%s'. Error: %s", prop.SSHKeyPath, err.Error())
		} else {
			p.prop.SSHKey = string(key)
		}
	}

	log.Debugln("Validated:", prop)

	p.prop.Password = prop.Password
	p.prop.FirewallID = prop.FirewallID
	p.prop.LoadBalancerID = prop.LoadBalancerID
	p.prop.MonitorPolicyID = prop.MonitorPolicyID

	return nil
}

// Provision creates a new instance.
func (p *Provisioner) Provision(spec instance.Spec) (*instance.ID, error) {
	rand.Seed(time.Now().UTC().UnixNano())
	name := fmt.Sprintf("instance-%d", rand.Int63())

	req := oneandone.ServerRequest{
		Name:               name,
		ApplianceId:        p.prop.Appliance,
		DatacenterId:       p.prop.Datacenter,
		Password:           p.prop.Password,
		SSHKey:             p.prop.SSHKey,
		PowerOn:            true,
		FirewallPolicyId:   p.prop.FirewallID,
		LoadBalancerId:     p.prop.LoadBalancerID,
		MonitoringPolicyId: p.prop.MonitorPolicyID,
		Hardware: oneandone.Hardware{
			FixedInsSizeId: p.prop.FixedServerSize,
		},
	}

	_, server, err := p.client.CreateServer(&req)
	if err != nil {
		log.Errorln("Creating server failed")
		return nil, err
	}

	if p.wait {
		p.client.WaitForState(server, "POWERED_ON", 10, 360)

		server, err = p.client.GetServer(server.Id)
		if err != nil {
			return nil, err
		}
	}

	ip := ""
	if server.Ips != nil {
		ip = server.Ips[0].Ip
	}

	id := instance.ID(name)
	logicalID := instance.LogicalID(ip)

	description := instance.Description{
		Tags:      spec.Tags,
		ID:        id,
		LogicalID: &logicalID,
	}

	description.Tags["serverID"] = server.Id

	buff, err := json.MarshalIndent(description, "  ", "  ")
	if err != nil {
		return nil, err
	}

	return &id, afero.WriteFile(p.fs, filepath.Join(p.Dir, string(id)+".1and1.spec"), buff, 0644)
}

// Destroy terminates an existing instance.
func (p *Provisioner) Destroy(id instance.ID) error {
	servers, err := p.client.ListServers(0, 0, "", string(id), "")

	if err != nil {
		log.Warningf("Server '%s' could not be found, assuming it is already deleted", string(id))
	} else {
		for _, s := range servers {
			if s.Name == string(id) {
				_, err = p.client.DeleteServer(s.Id, false)

				if err != nil {
					log.Errorf("Cannot delete server '%s'", string(id))
					return err
				}
				break
			}
		}
	}

	fp := filepath.Join(p.Dir, string(id)+".1and1.spec")
	return p.fs.Remove(fp)
}

// DescribeInstances returns descriptions of all instances matching all of the provided tags.
func (p *Provisioner) DescribeInstances(tags map[string]string) ([]instance.Description, error) {
	log.Debugln("describe-instances", tags)

	result := []instance.Description{}
	re := regexp.MustCompile("(^instance-[0-9]+)(.1and1.spec)")

	fs := &afero.Afero{Fs: p.fs}
	err := fs.Walk(p.Dir, func(path string, info os.FileInfo, err error) error {
		matches := re.FindStringSubmatch(info.Name())

		if len(matches) == 3 {
			fName := filepath.Join(p.Dir, info.Name())
			buff, err := ioutil.ReadFile(fName)

			if err != nil {
				log.Errorf("Cannot read file '%s'.", fName)
				return err
			}

			description := instance.Description{}
			err = json.Unmarshal(buff, &description)
			if err != nil {
				log.Errorln("Instance description unmarshal error")
				return err
			}

			result = append(result, description)
		}
		return nil
	})

	return result, err
}
