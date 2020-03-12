package test

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/samuel/go-zookeeper/zk"
	"github.com/stretchr/testify/suite"
)

func TestZkSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skip zk test")
		return
	}
	suite.Run(t, new(zkSuite))
}

type zkSuite struct {
	suite.Suite
}

func (s *zkSuite) TestSerivce() {
	service := &zkService{}

	port, err := service.Start()
	s.NoError(err, "start service error")
	defer service.Stop()

	conn, _, err := zk.Connect([]string{fmt.Sprintf("localhost:%s", port)}, 3*time.Second)
	s.NoError(err, "get conn error")

	_, err = conn.Create("/testhome", nil, 0, zk.WorldACL(zk.PermAll))
	s.NoError(err, "create node error")

	ok, _, err := conn.Exists("/testhome")
	s.NoError(err, "get node error")
	s.True(ok, "no such node")
}

func (s *zkSuite) TestStop() {
	service := &zkService{}
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
