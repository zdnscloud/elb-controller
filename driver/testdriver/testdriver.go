package testdriver

import (
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/elb-controller/driver"
)

const (
	versionInfo = "zcloud lb test driver"
)

type TestDriver struct {
	serverAddr string
	user       string
	password   string
}

func New() *TestDriver {
	return &TestDriver{}
}

func (d *TestDriver) Create(c driver.Config) error {
	log.Debugf("[TestDriver] recvice create task:%s", c.ToJson())
	return nil
}

func (d *TestDriver) Update(old, new driver.Config) error {
	log.Debugf("[TestDriver] recvice update task:%s %s", old.ToJson(), new.ToJson())
	return nil
}

func (d *TestDriver) Delete(c driver.Config) error {
	log.Debugf("[TestDriver] recvice delete task:%s", c.ToJson())
	return nil
}

func (d *TestDriver) Version() string {
	return versionInfo
}

var _ driver.Driver = &TestDriver{}
