package radware

import (
	"github.com/zdnscloud/elb-controller/driver/radware/client"
)

func (c radwareConfig) delete(cli *client.Client) error {
	if err := cli.VirtualServer().Delete(c.VsID); err != nil {
		return err
	}

	if err := cli.ServerGroup().Delete(c.VsID); err != nil {
		return err
	}

	for rs := range c.RealServers {
		if err := cli.RealServer().Delete(rs); err != nil {
			return err
		}
	}
	return nil
}

func (c radwareConfig) create(cli *client.Client) error {
	if err := cli.ServerGroup().Reconcile(c.VsID, c.ServerGroup); err != nil {
		return err
	}

	for k, v := range c.RealServers {
		if err := cli.RealServer().Reconcile(k, v); err != nil {
			return err
		}
		if err := cli.RealServerPort().Reconcile(k, c.RealServerPort); err != nil {
			return err
		}
		if err := cli.ServerGroup().ReconcileServer(c.VsID, k); err != nil {
			return err
		}
	}

	if err := cli.VirtualServer().Reconcile(c.VsID, c.VirtualServer); err != nil {
		return err
	}
	return cli.VirtualService().Reconcile(c.VsID, c.VirtualService)
}

func (c updateRadwareConfig) update(cli *client.Client) error {
	if err := cli.ServerGroup().Reconcile(c.new.VsID, c.new.ServerGroup); err != nil {
		return err
	}

	if err := cli.VirtualService().Reconcile(c.new.VsID, c.new.VirtualService); err != nil {
		return err
	}

	for toDeleteRs := range getToDeleteRsmap(c.old, c.new) {
		if err := cli.RealServer().Delete(toDeleteRs); err != nil {
			return err
		}
	}

	for toAddRsID, toAddRs := range getToAddRsmap(c.old, c.new) {
		if err := cli.RealServer().Reconcile(toAddRsID, toAddRs); err != nil {
			return err
		}
		if err := cli.RealServerPort().Reconcile(toAddRsID, c.new.RealServerPort); err != nil {
			return err
		}
		if err := cli.ServerGroup().ReconcileServer(c.new.VsID, toAddRsID); err != nil {
			return err
		}
	}
	return nil
}
