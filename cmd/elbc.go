package main

import (
	"flag"
	"fmt"

	"github.com/zdnscloud/elb-controller/driver/radware"
	"github.com/zdnscloud/elb-controller/lbctrl"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/signal"
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/client/config"
)

func createK8SClient() (cache.Cache, client.Client, error) {
	config, err := config.GetConfig()
	if err != nil {
		return nil, nil, err
	}

	c, err := cache.New(config, cache.Options{})
	if err != nil {
		return nil, nil, err
	}

	cli, err := client.New(config, client.Options{})
	if err != nil {
		return nil, nil, err
	}

	stop := make(chan struct{})
	go c.Start(stop)
	c.WaitForCacheSync(stop)

	return c, cli, nil
}

var (
	masterServer string
	backupServer string
	user         string
	password     string
	cluster      string
	version      string
	build        string
	showVersion  bool
)

func main() {

	flag.StringVar(&masterServer, "masterserver", "", "master external loadbalancer managerment address")
	flag.StringVar(&backupServer, "backupserver", "", "backup external loadbalancer managerment address")
	flag.StringVar(&user, "user", "admin", "external loadbalancer user")
	flag.StringVar(&password, "password", "zcloud", "external loadbalancer password")
	flag.StringVar(&cluster, "cluster", "local", "zcloud kubernetes cluster name")
	flag.BoolVar(&showVersion, "version", false, "show version")
	flag.Parse()

	if showVersion {
		fmt.Printf("Zcloud elb-controller %s (build at %s)\n", version, build)
		return
	}

	log.InitLogger("debug")

	cache, cli, err := createK8SClient()
	if err != nil {
		log.Fatalf("Create cache failed:%s", err.Error())
	}

	driver := radware.New(masterServer, backupServer, user, password)
	log.Infof("Driver info:%s", driver.Version())

	_, err = lbctrl.New(cli, cache, cluster, driver)
	if err != nil {
		log.Fatalf("new controller failed %s", err.Error())
	}
	signal.WaitForInterrupt(nil)
}
