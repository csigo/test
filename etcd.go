package test

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"time"

	"github.com/fsouza/go-dockerclient"
)

const (
	etcdChkTimes = 10
	etcdChkDelay = time.Second
)

func init() {
	RegisterService(Etcd, func() Service {
		return &etcdService{}
	})
}

type etcdService struct {
	ports     []int
	workDir   string
	cmd       *exec.Cmd
	container *docker.Container
}

func (s *etcdService) Start() (string, error) {
	// perform default check
	if err := CheckExecutable("etcd"); err != nil {
		return "", err
	}

	// booking 2 ports
	var err error
	s.ports, err = BookPorts(2)
	if err != nil {
		return "", fmt.Errorf("fail to book ports, err:%v", err)
	}

	// prepare tmp dir
	s.workDir, err = ioutil.TempDir("", "etcd-test")
	if err != nil {
		return "", fmt.Errorf("fail to prepare tmp dir, err:%v", err)
	}

	s.cmd = exec.Command(
		"etcd",
		fmt.Sprintf("--listen-client-urls=http://0.0.0.0:%d", s.ports[0]),
		fmt.Sprintf("--advertise-client-urls=http://0.0.0.0:%d", s.ports[0]),
		fmt.Sprintf("-data-dir=%s", s.workDir),
		fmt.Sprintf("-name=m%d", s.ports[0]),
	)
	if err := s.cmd.Start(); err != nil {
		return "", err
	}

	for i := 0; i < etcdChkTimes; i++ {
		time.Sleep(etcdChkDelay)
		if CheckListening(s.ports[0]) {
			return fmt.Sprintf("localhost:%d", s.ports[0]), nil
		}
	}
	// only need region server thrift port
	return "", fmt.Errorf("fail to start etcd")
}

func (s *etcdService) Stop() error {
	// close process
	if err := s.cmd.Process.Kill(); err != nil {
		return err
	}
	time.Sleep(time.Second)
	return nil
}

// StartDocker start the service via docker
func (s *etcdService) StartDocker(cl *docker.Client) (ipport string, err error) {
	s.container, ipport, err = StartContainer(
		cl,
		SetImage("quay.io/coreos/etcd"),
		SetExposedPorts([]string{"2379/tcp", "2380/tcp"}),
		SetCommand([]string{
			"/usr/local/bin/etcd",
			"-advertise-client-urls=http://0.0.0.0:2379",
			"-listen-client-urls=http://0.0.0.0:2379",
		}),
	)
	return ipport, err
}

// StopDocker stops the service via docker
func (s *etcdService) StopDocker(cl *docker.Client) error {
	return RemoveContainer(cl, s.container)
}
