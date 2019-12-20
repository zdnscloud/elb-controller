package radware

import (
	"github.com/zdnscloud/elb-controller/driver"
	"github.com/zdnscloud/elb-controller/driver/radware/client"
)

const (
	version = "radware lb driver v0.0.1"
)

type RadwareDriver struct {
	client *client.Client
}

func New(serverAddr, user, password string) *RadwareDriver {
	return &RadwareDriver{
		client: client.New(user, password, serverAddr),
	}
}

func (d *RadwareDriver) Create(c driver.Config) error {
	if err := validateConfig(c); err != nil {
		return err
	}
	for _, config := range getRadwareConfigs(c) {
		if err := config.create(d.client); err != nil {
			return err
		}
	}
	return d.client.ApplyAndSave()
}

func (d *RadwareDriver) Update(old, new driver.Config) error {
	if err := validateConfig(old); err != nil {
		return err
	}
	if err := validateConfig(new); err != nil {
		return err
	}

	olds := getRadwareConfigs(old)
	news := getRadwareConfigs(new)
	for _, toD := range getToDeleteRdConfigs(olds, news) {
		if err := toD.delete(d.client); err != nil {
			return err
		}
	}

	for _, toA := range getToAddRdConfigs(olds, news) {
		if err := toA.create(d.client); err != nil {
			return err
		}
	}

	for _, toU := range getUpdateRdConfigs(olds, news) {
		if err := toU.update(d.client); err != nil {
			return err
		}
	}
	return d.client.ApplyAndSave()
}

func (d *RadwareDriver) Delete(c driver.Config) error {
	if err := validateConfig(c); err != nil {
		return err
	}

	for _, config := range getRadwareConfigs(c) {
		if err := config.delete(d.client); err != nil {
			return err
		}
	}
	return d.client.ApplyAndSave()
}

func (d *RadwareDriver) Version() string {
	return version
}

var _ driver.Driver = &RadwareDriver{}
