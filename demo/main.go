package main

import (
	"fmt"

	"github.com/17media/test"
)

func run() {
	sl := test.NewServiceLauncher()
	defer sl.StopAll()

	zkPort, _, _ := sl.Start(test.ZooKeeper)
	fmt.Println(zkPort)

	redisPort, _, _ := sl.Start(test.Redis)
	fmt.Println(redisPort)

	etcdPort, _, _ := sl.Start(test.Etcd)
	fmt.Println(etcdPort)

	hbasePort, _, _ := sl.Start(test.HBase)
	fmt.Println(hbasePort)

	s := sl.Get(hbasePort)
	err := s.(test.HbaseService).RunScript(`list`)
	fmt.Println(err)
	err = s.(test.HbaseService).RunScriptFromFile("schema.hbase")
	fmt.Println(err)
}

func main() {
	run()
}
