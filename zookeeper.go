package test

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"
)

const (
	// config file name and template
	zkCfgFileName = "zoo_test.cfg"
	zkCfgTpl      = `
tickTime=100
initLimit=20
syncLimit=20
dataDir={{.ZK_DATA_DIR}}
clientPort={{.ZK_PORT}}
`
)

func init() {
	RegisterService(ZooKeeper, func() Service {
		return &zkService{}
	})
}

type zkService struct {
	port    int
	workDir string
}

func (s *zkService) Start() (int, error) {
	// perform default check
	if err := CheckExecutable("java", "zkServer.sh"); err != nil {
		return 0, err
	}

	// booking 4 ports
	ports, err := BookPorts(1)
	if err != nil {
		return 0, fmt.Errorf("fail to book ports, err:%v", err)
	}
	s.port = ports[0]

	// prepare tmp dir
	s.workDir, err = ioutil.TempDir("", "zk-test")
	if err != nil {
		return 0, fmt.Errorf("fail to prepare tmp dir, err:%v", err)
	}

	// prepare cfg
	if err = ApplyTemplate(
		s.cfgFile(),
		zkCfgTpl,
		map[string]interface{}{
			"ZK_PORT":     s.port,
			"ZK_DATA_DIR": s.workDir,
		}); err != nil {
		return 0, fmt.Errorf("fail to prepare cfg file, err:%v", err)
	}

	// leverage zkServer.sh to start zk with config file
	if err := Exec(
		s.workDir, nil, nil,
		"zkServer.sh", "start", s.cfgFile()); err != nil {
		return 0, fmt.Errorf("fail to start hbase master, err:%v", err)
	}

	// Make sure zk really starts
	if err := WaitPortAvail(s.port, 20*time.Second); err != nil {
		return 0, fmt.Errorf("port %v isn't available after 20 sec", s.port)
	}

	// only need region server thrift port
	return s.port, nil
}

func (s *zkService) Stop() error {
	return Exec(
		s.workDir, nil, nil,
		"zkServer.sh", "stop", s.cfgFile())
}

func (s *zkService) cfgFile() string {
	return filepath.Join(s.workDir, zkCfgFileName)
}
