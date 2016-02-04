package test

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

func TestSrvLauncherSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skip service launcher test")
		return
	}
	suite.Run(t, new(srvLauncherSuite))
}

type srvLauncherSuite struct {
	suite.Suite
}

func (s *srvLauncherSuite) TestAll() {
	sl := NewServiceLauncher()

	var err error
	var ports = make([]int, 5)

	ports[0], _, err = sl.Start(ZooKeeper)
	s.NoError(err)

	ports[1], _, err = sl.Start(Redis)
	s.NoError(err)

	ports[2], _, err = sl.Start(Etcd)
	s.NoError(err)

	ports[3], _, err = sl.Start(HBase)
	s.NoError(err)

	ports[4], _, err = sl.Start(Gnatsd)
	s.NoError(err)

	for _, port := range ports {
		_, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), time.Second)
		s.NoError(err)
	}

	s.NoError(sl.StopAll())

	time.Sleep(5 * time.Second)

	for _, port := range ports {
		_, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), time.Second)
		s.Error(err)
	}
}

func (s *srvLauncherSuite) TestDoubleStop() {
	sl := NewServiceLauncher()
	defer sl.StopAll()

	_, stop, err := sl.Start(Gnatsd)
	s.NoError(err, "fail to create service")
	s.NoError(stop(), "fail to stop service")
	s.Error(stop(), "fail to inform double stop")
}

func (s *srvLauncherSuite) TestGet() {
	sl := NewServiceLauncher()
	defer sl.StopAll()

	port, _, err := sl.Start(Gnatsd)
	s.NoError(err, "fail to create service")
	_, ok := sl.Get(port).(*gnatsdService)
	s.True(ok, "service is not gnatsd service")
}
