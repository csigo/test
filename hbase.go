package test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"
)

const (
	hbaseChkTimes = 20
	hbaseChkDelay = time.Second
	// config file name and template
	hbaseCfgFileName = "hbase-site.xml"
	hbaseCfgTpl      = `
<configuration>
  <property>
    <name>hbase.zookeeper.property.clientPort</name>
    <value>{{.ZK_PORT}}</value>
  </property>
  <property>
    <name>hbase.master.info.port</name>
    <value>{{.HBASE_MASTER_PORT}}</value>
  </property>
  <property>
    <name>hbase.rootdir</name>
    <value>{{.HBASE_ROOTDIR}}</value>
  </property>
  <property>
    <name>hbase.zookeeper.property.dataDir</name>
    <value>{{.HBASE_ROOTDIR}}/zookeeper</value>
  </property>
  <property>
    <name>hbase.thrift.info.port</name>
    <value>{{.HBASE_THRIFT_PORT}}</value>
  </property>
  <property>
    <name>hbase.regionserver.thrift.port</name>
    <value>{{.HBASE_REG_THRIFT_PORT}}</value>
  </property>
  <property>
    <name>hbase.zookeeper.property.maxClientCnxns</name>
    <value>1000</value>
  </property>
  <property>
    <name>hbase.thrift.minWorkerThreads</name>
    <value>1000</value>
  </property>
  <property>
    <name>hbase.regionserver.thrift.framed</name>
    <value>true</value>
  </property>
  <property>
    <name>hbase.regionserver.thrift.framed.max_frame_size_in_mb</name>
    <value>16</value>
  </property>
</configuration>
`
)

// HbaseService represents hbase service
type HbaseService interface {
	// RunScript runs the hbase script directly
	RunScript(script string) error
	// RunScript runs the hbase script file
	RunScriptFromFile(file string) error
}

func init() {
	RegisterService(HBase, func() Service {
		return &hbaseService{}
	})
}

type hbaseService struct {
	ports   []int
	envs    []string
	workDir string
}

func (s *hbaseService) Start() (int, error) {
	// perform default check
	if err := CheckExecutable("java", "hbase", "hbase-daemon.sh"); err != nil {
		return 0, err
	}

	// booking 4 ports
	var err error
	s.ports, err = BookPorts(4)
	if err != nil {
		return 0, fmt.Errorf("fail to book ports, err:%v", err)
	}

	// prepare tmp dir
	s.workDir, err = ioutil.TempDir("", "hbase-test")
	if err != nil {
		return 0, fmt.Errorf("fail to prepare tmp dir, err:%v", err)
	}

	// prepare cfg
	if err = ApplyTemplate(
		filepath.Join(s.workDir, hbaseCfgFileName),
		hbaseCfgTpl,
		map[string]interface{}{
			"HBASE_REG_THRIFT_PORT": s.ports[0],
			"HBASE_THRIFT_PORT":     s.ports[1],
			"HBASE_MASTER_PORT":     s.ports[2],
			"ZK_PORT":               s.ports[3],
			"HBASE_ROOTDIR":         s.workDir,
		}); err != nil {
		return 0, fmt.Errorf("fail to prepare cfg file, err:%v", err)
	}

	// prepare env variables
	s.envs = []string{
		fmt.Sprintf("HBASE_CONF_DIR=%s", s.workDir),
		fmt.Sprintf("HBASE_LOG_DIR=%s", s.workDir),
		fmt.Sprintf("HBASE_PID_DIR=%s", s.workDir),
	}

	if err := Exec(s.workDir, s.envs, nil, "hbase-daemon.sh", "start", "master"); err != nil {
		return 0, fmt.Errorf("fail to start hbase master, err:%v", err)
	}
	if err := Exec(s.workDir, s.envs, nil, "hbase-daemon.sh", "start", "thrift"); err != nil {
		return 0, fmt.Errorf("fail to start hbase thrift, err:%v", err)
	}

	for i := 0; i < hbaseChkTimes; i++ {
		time.Sleep(hbaseChkDelay)
		if s.check() == nil {
			return s.ports[0], nil
		}
	}
	// only need region server thrift port
	return 0, fmt.Errorf("fail to start hbase")
}

func (s *hbaseService) Stop() error {
	return CombineError(
		Exec(s.workDir, s.envs, nil, "hbase-daemon.sh", "stop", "thrift"),
		Exec(s.workDir, s.envs, nil, "hbase-daemon.sh", "stop", "master"),
	)
}

func (s *hbaseService) RunScript(script string) error {
	in := bytes.NewReader([]byte(script))
	return Exec(s.workDir, s.envs, in, "hbase", "shell")
}

func (s *hbaseService) RunScriptFromFile(file string) error {
	return Exec(s.workDir, s.envs, nil, "hbase", "shell", file)
}

func (s *hbaseService) check() error {
	if !CheckListening(s.ports[2], s.ports[1]) {
		return fmt.Errorf("not listening")
	}
	buf := bytes.NewBuffer(nil)
	buf.Write([]byte("list"))
	return Exec(s.workDir, s.envs, buf, "hbase", "shell")
}
