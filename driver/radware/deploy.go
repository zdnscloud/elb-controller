package radware

import (
	"github.com/zdnscloud/elb-controller/driver/radware/client"
	"github.com/zdnscloud/elb-controller/driver/radware/types"
)

func (c radwareConfig) delete(cli *client.Client) error {
	if err := cli.VServer().Delete(c.VsID); err != nil {
		return err
	}

	if err := cli.Sg().Delete(c.VsID); err != nil {
		return err
	}

	for rs := range c.RealServers {
		if err := cli.Rs().Delete(rs); err != nil {
			return err
		}
	}
	return nil
}

func (c radwareConfig) create(cli *client.Client) error {
	if err := cli.Sg().Create(c.VsID, c.ServerGroup); err != nil {
		return err
	}

	for k, v := range c.RealServers {
		if err := cli.Rs().Create(k, v); err != nil {
			return err
		}
		if err := cli.Rsp().Create(k, c.RealServerPort); err != nil {
			return err
		}
		if err := cli.Sg().Update(c.VsID, types.NewAddServerServerGroup(k)); err != nil {
			return err
		}
	}

	if err := cli.VServer().Create(c.VsID, c.VirtualServer); err != nil {
		return err
	}
	return cli.VService().Create(c.VsID, c.VirtualService)
}

func (c updateRadwareConfig) update(cli *client.Client) error {
	if err := cli.Sg().Update(c.new.VsID, c.new.ServerGroup); err != nil {
		return err
	}

	if err := cli.VService().Update(c.new.VsID, c.new.VirtualService); err != nil {
		return err
	}

	for toDeleteRs := range getToDeleteRsmap(c.old, c.new) {
		if err := cli.Rs().Delete(toDeleteRs); err != nil {
			return err
		}
	}

	for toAddRsID, toAddRs := range getToAddRsmap(c.old, c.new) {
		if err := cli.Rs().Create(toAddRsID, toAddRs); err != nil {
			return err
		}
		if err := cli.Rsp().Create(toAddRsID, c.new.RealServerPort); err != nil {
			return err
		}
	}

	if !isSameRsport(c) {
		for id, _ := range c.new.RealServers {
			if err := cli.Rsp().Update(id, c.new.RealServerPort); err != nil {
				return err
			}
		}
	}
	return nil
}
