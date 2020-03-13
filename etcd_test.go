package test

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/coreos/go-etcd/etcd"
	"github.com/stretchr/testify/suite"
)

func TestEtcdSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skip etcd test")
		return
	}
	suite.Run(t, new(etcdSuite))
}

type etcdSuite struct {
	suite.Suite
}

func (s *etcdSuite) TestSerivce() {
	service := &etcdService{}

	ipport, err := service.Start()
	s.NoError(err, "start service error")
	defer service.Stop()

	client := etcd.NewClient([]string{fmt.Sprintf("http://%s", ipport)})
	defer client.Close()

	testNode := "aaa"
	testValue := "ccc"

	s.NoError(err, "create path failed")

	_, err = client.Create(testNode, testValue, 0)
	s.NoError(err, "create node failed")

	resp, err := client.Get(testNode, false, false)
	s.NoError(err, "get path failed")
	s.Equal(testValue, resp.Node.Value)
}

func (s *etcdSuite) TestStop() {
	service := &etcdService{}
	defer service.Stop()

	port, err := service.Start()
	s.NoError(err, "start service error")

	_, err = net.Listen("tcp", fmt.Sprintf(":%s", port))
	s.Error(err, "port is not listenering")
	service.Stop()

	time.Sleep(3 * time.Second)

	ln, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	s.NoError(err, "port is listenering")
	ln.Close()
}
