package test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/fsouza/go-dockerclient"
)

const (
	elasticSearchChkTimes       = 10
	elasticSearchChkDelay       = 1 * time.Second
	elasticSearchAvailableDelay = 50
)

func init() {
	RegisterService(ElasticSearch, func() Service {
		return &esService{}
	})
}

type esService struct {
	port    int
	workDir string
}

func (s *esService) Start() (string, error) {
	// perform default check
	if err := CheckExecutable("elasticsearch"); err != nil {
		return "", err
	}

	// booking 1 ports
	ports, err := BookPorts(1)
	if err != nil {
		return "", fmt.Errorf("fail to book ports, err:%v", err)
	}
	s.port = ports[0]

	// prepare tmp dir
	s.workDir, err = ioutil.TempDir("", "elasticsearch-test")
	if err != nil {
		return "", fmt.Errorf("fail to prepare tmp dir, err:%v", err)
	}

	pidFile := filepath.Join(s.workDir, "elasticsearch.pid")
	dataDir := filepath.Join(s.workDir, "data")
	logsDir := filepath.Join(s.workDir, "logs")

	host, _ := os.Hostname()
	if err := Exec(
		s.workDir, nil, nil, "elasticsearch",
		fmt.Sprintf("-Des.http.port=%d", s.port),
		fmt.Sprintf("-Des.cluster.name=elasticsearch-csi-test-%s-%d", host, os.Getpid()),
		"-Des.script.default_lang=groovy",
		"-Des.script.disable_dynamic=false",
		"-Des.node.local=true",
		"-Des.index.number_of_shards=1",
		"-Des.index.number_of_replicas=0",
		"-d", "-p", pidFile,
		"-Des.path.data="+dataDir,
		"-Des.path.logs="+logsDir); err != nil {
		return "", fmt.Errorf("fail to start start elastic server, err:%v", err)
	}

	// check if remote port is listening
	for i := 0; i < elasticSearchChkTimes; i++ {
		time.Sleep(elasticSearchChkDelay)
		if CheckListening(s.port) {
			// check if server is ready
			if !s.isServerAvailable() {
				s.Stop()
				return "", fmt.Errorf("Elastic Search start time out")
			}
			return fmt.Sprintf("localhost:%d", s.port), nil
		}
	}
	return "", fmt.Errorf("fail to start elastic search")
}

func (s *esService) Stop() error {
	pidFile := filepath.Join(s.workDir, "elasticsearch.pid")
	bytes, err := ioutil.ReadFile(pidFile)
	if err != nil {
		return fmt.Errorf("Read pid file error: %s", err)
	}
	pid, err := strconv.Atoi(string(bytes))
	if err != nil {
		return fmt.Errorf("Parse pid error: %s", err)
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("Get process %d error: %s", pid, err)
	}
	if err = process.Kill(); err != nil {
		return fmt.Errorf("Kill process %d error: %s", pid, err)
	}
	return nil
}

// StartDocker start the service via docker
func (s *esService) StartDocker(cl *docker.Client) (string, error) {
	return "", fmt.Errorf("implmenet this")
}

// StopDocker stops the service via docker
func (s *esService) StopDocker(cl *docker.Client) error {
	return fmt.Errorf("implmenet this")
}

// Wait until Easltic Search cluster status is good enough for operations
func (s *esService) isServerAvailable() bool {
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/_cluster/health?wait_for_status=yellow&timeout=%ds", s.port, elasticSearchAvailableDelay))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false
	}
	var v map[string]interface{}
	err = json.Unmarshal(bytes, &v)
	if err != nil {
		return false
	}
	return !v["timed_out"].(bool)
}
