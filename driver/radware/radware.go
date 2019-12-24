package radware

import (
	"github.com/zdnscloud/elb-controller/driver"
	"github.com/zdnscloud/elb-controller/driver/radware/client"
)

const (
	version = "radware lb driver v0.0.1"
)

type RadwareDriver struct {
	primary   *client.Client
	secondary *client.Client
}

func New(masterServer, backupServer, user, password string) *RadwareDriver {
	if backupServer != "" {
		return &RadwareDriver{
			primary:   client.New(user, password, masterServer),
			secondary: client.New(user, password, backupServer),
		}
	}
	return &RadwareDriver{
		primary: client.New(user, password, masterServer),
	}
}

func (d *RadwareDriver) client() *client.Client {
	if d.secondary == nil {
		return d.primary
	}

	m, err := d.primary.IsMaster()
	if m && err == nil {
		return d.primary
	}

	b, err := d.secondary.IsMaster()
	if b && err == nil {
		return d.secondary
	}
	return d.primary
}

func (d *RadwareDriver) Create(c driver.Config) error {
	client := d.client()
	if err := validateConfig(c); err != nil {
		return err
	}
	for _, config := range getRadwareConfigs(c) {
		if err := config.create(client); err != nil {
			return err
		}
	}
	return client.ApplyAndSave()
}

func (d *RadwareDriver) Update(old, new driver.Config) error {
	client := d.client()
	if err := validateConfig(old); err != nil {
		return err
	}
	if err := validateConfig(new); err != nil {
		return err
	}

	olds := getRadwareConfigs(old)
	news := getRadwareConfigs(new)
	for _, toD := range getToDeleteRdConfigs(olds, news) {
		if err := toD.delete(client); err != nil {
			return err
		}
	}

	for _, toA := range getToAddRdConfigs(olds, news) {
		if err := toA.create(client); err != nil {
			return err
		}
	}

	for _, toU := range getUpdateRdConfigs(olds, news) {
		if err := toU.update(client); err != nil {
			return err
		}
	}
	return d.client().ApplyAndSave()
}

func (d *RadwareDriver) Delete(c driver.Config) error {
	client := d.client()
	if err := validateConfig(c); err != nil {
		return err
	}

	for _, config := range getRadwareConfigs(c) {
		if err := config.delete(client); err != nil {
			return err
		}
	}
	return client.ApplyAndSave()
}

func (d *RadwareDriver) Version() string {
	return version
}

var _ driver.Driver = &RadwareDriver{}
