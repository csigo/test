package test

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/fsouza/go-dockerclient"
)

const (
	redisChkTimes = 10
	redisChkDelay = 1 * time.Second
	maxMemory     = "64mb"
)

func init() {
	RegisterService(Redis, func() Service {
		return &redisService{}
	})
}

type redisService struct {
	port      int
	workDir   string
	auth      string
	container *docker.Container
}

func (s *redisService) Start() (string, error) {
	// perform default check
	if err := CheckExecutable("redis-server", "redis-cli"); err != nil {
		return "", err
	}

	// booking 1 ports
	ports, err := BookPorts(1)
	if err != nil {
		return "", fmt.Errorf("fail to book ports, err:%v", err)
	}
	s.port = ports[0]

	// prepare tmp dir
	s.workDir, err = ioutil.TempDir("", "redis-test")
	if err != nil {
		return "", fmt.Errorf("fail to prepare tmp dir, err:%v", err)
	}

	pidFile := filepath.Join(s.workDir, "redis.pid")
	logFile := filepath.Join(s.workDir, "redis.log")

	cmds := []interface{}{
		"--daemonize", "yes",
		"--port", s.port,
		"--pidfile", pidFile,
		"--logfile", logFile,
		"--dir", s.workDir,
		"--maxmemory", maxMemory,
	}

	if s.auth != "" {
		cmds = append(cmds, "--requirepass", s.auth)
	}

	if err := Exec(s.workDir, nil, nil, "redis-server", cmds...); err != nil {
		return "", fmt.Errorf("fail to start redis server, err:%v", err)
	}

	for i := 0; i < redisChkTimes; i++ {
		time.Sleep(redisChkDelay)
		if CheckListening(s.port) {
			return fmt.Sprintf("localhost:%d", s.port), nil
		}
	}
	// only need region server thrift port
	return "", fmt.Errorf("fail to start redis")
}

func (s *redisService) Stop() error {
	// close process
	return Exec(
		s.workDir, nil, nil,
		"redis-cli",
		"-h", "localhost",
		"-p", s.port,
		"shutdown")
}

// StartDocker start the service via docker
func (s *redisService) StartDocker(cl *docker.Client) (ipport string, err error) {
	s.container, ipport, err = StartContainer(
		cl,
		SetImage("redis"),
		SetExposedPorts([]string{"6379/tcp"}),
	)
	return ipport, err
}

// StopDocker stops the service via docker
func (s *redisService) StopDocker(cl *docker.Client) error {
	return RemoveContainer(cl, s.container)
}

func RedisAuth(password string) ServiceOption {
	return func(s Service) error {
		rs, ok := s.(*redisService)
		if !ok {
			return fmt.Errorf("can't set redis auth with service %v", s)
		}
		rs.auth = password
		return nil
	}
}
