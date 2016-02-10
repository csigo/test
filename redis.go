package test

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"
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
	port    int
	workDir string
	auth    string
}

func (s *redisService) Start() (int, error) {
	// perform default check
	if err := CheckExecutable("redis-server", "redis-cli"); err != nil {
		return 0, err
	}

	// booking 1 ports
	ports, err := BookPorts(1)
	if err != nil {
		return 0, fmt.Errorf("fail to book ports, err:%v", err)
	}
	s.port = ports[0]

	// prepare tmp dir
	s.workDir, err = ioutil.TempDir("", "redis-test")
	if err != nil {
		return 0, fmt.Errorf("fail to prepare tmp dir, err:%v", err)
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
		return 0, fmt.Errorf("fail to start redis server, err:%v", err)
	}

	for i := 0; i < redisChkTimes; i++ {
		time.Sleep(redisChkDelay)
		if CheckListening(s.port) {
			return s.port, nil
		}
	}
	// only need region server thrift port
	return 0, fmt.Errorf("fail to start redis")
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
