package test

// This file handles the consul service.
// 1) We generate a config file with the port generated dynamically with BookPorts().
//    The reason is that the ports cannot be setup through the command line only.
//    It needs to be setup with a config file.
// 2) It guarantees that both the service port is listening and the leader is elected.

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	consul "github.com/hashicorp/consul/api"
)

const (
	consulChkTimesListen = 20                    // consulChkTimesListen is the number of times to retry for port listening.
	consulChkDelayListen = 50 * time.Millisecond // consulChkDelayListen is the waiting time for next retry for port listening.

	// Constants to check leader election mechanism, which usually takes 1-2 seconds.
	// Thus, we use 5 seconds as the time limit for the checking.
	consulChkTimesLeader = 10                     // consulChkTimesLeader is the number of times to retry for leader election.
	consulChkDelayLeader = 500 * time.Millisecond // consulChkDelayLeader is the waiting time for next retry for leader election.
)

func init() {
	RegisterService(Consul, func() Service {
		return &consulService{}
	})
}

// consulConfig is the config for consul service.
// It is basically a clone of the original config.
// Ref: https://github.com/hashicorp/consul/blob/master/command/agent/config.go#L96
//      https://www.consul.io/docs/agent/options.html
type consulConfig struct {
	BootstrapExpect int                `json:"bootstrap_expect"`
	Server          bool               `json:"server"`
	DataDir         string             `json:"data_dir"`
	Ports           *consulPortsConfig `json:"ports"`
}

// consulPortsConfig is the config for ports of consul service.
// It is basically a clone of the original Ports config.
// Ref: https://github.com/hashicorp/consul/blob/master/command/agent/config.go#L23
//      https://www.consul.io/docs/agent/options.html#ports
type consulPortsConfig struct {
	DNS     int `json:"dns"`      // DNS Query interface
	HTTP    int `json:"http"`     // HTTP API
	HTTPS   int `json:"https"`    // HTTPS API
	RPC     int `json:"rpc" `     // CLI RPC
	SerfLan int `json:"serf_lan"` // LAN gossip (Client + Server)
	SerfWan int `json:"serf_wan"` // WAN gossip (Server only)
	Server  int `json:"server"`   // Server internal RPC
}

// consulService is the consul service.
type consulService struct {
	// cmd is the command to run consul.
	cmd *exec.Cmd

	// port is the http port for consul service.
	port int
}

// Start runs the consul service and returns its port.
func (s *consulService) Start() (port int, err error) {
	if err := CheckExecutable("consul"); err != nil {
		return 0, fmt.Errorf("Consul is not installed: %v", err)
	}
	workDir, err := ioutil.TempDir("", "consul")
	if err != nil {
		return 0, fmt.Errorf("Fail to generate work dir: %v", err)
	}
	ports, err := BookPorts(5)
	if err != nil {
		return 0, fmt.Errorf("Fail to book ports for consul: %v", err)
	}
	config := &consulConfig{
		BootstrapExpect: 1,
		Server:          true,
		DataDir:         filepath.Join(workDir, "data"),
		Ports: &consulPortsConfig{
			DNS:     -1,
			HTTP:    ports[0],
			HTTPS:   -1,
			RPC:     ports[1],
			SerfLan: ports[2],
			SerfWan: ports[3],
			Server:  ports[4],
		},
	}
	b, err := json.Marshal(config)
	if err != nil {
		return 0, fmt.Errorf("Fail to json marshal consul config: %v", err)
	}
	configFile := filepath.Join(workDir, "consul.conf")
	if err := ioutil.WriteFile(configFile, b, os.ModePerm); err != nil {
		return 0, fmt.Errorf("Fail to write config file(path: %v): %v", configFile, err)
	}
	s.cmd = exec.Command(
		"consul", "agent", "-config-file", configFile,
	)
	if err := s.cmd.Start(); err != nil {
		return 0, fmt.Errorf("Fail to start consul: %v", err)
	}
	s.port = config.Ports.HTTP

	// Make sure that the server is running.
	if err := checkPortListening(s.port, consulChkTimesListen, consulChkDelayListen); err != nil {
		return 0, fmt.Errorf("Port is not listening: %v", err)
	}
	if err := checkLeaderElected(s.port, consulChkTimesLeader, consulChkDelayLeader); err != nil {
		return 0, fmt.Errorf("Leader election mechanism fails: %v", err)
	}
	return s.port, nil
}

// checkPortListening checks whether the port is listening or not.
// - port : The port to check.
// - times: Total number of checks.
// - delay: Time delay between checks.
func checkPortListening(port, times int, delay time.Duration) error {
	for i := 0; i < times; i++ {
		time.Sleep(delay)
		if CheckListening(port) {
			return nil
		}
	}
	return fmt.Errorf("Port(%v) is not listening within time: %v",
		port, time.Duration(int64(times))*delay)
}

// checkLeaderElected checks whether the leader is elected for the consul service or not.
// - port : The port to check.
// - times: Total number of checks.
// - delay: Time delay between checks.
func checkLeaderElected(port, times int, delay time.Duration) error {
	config := consul.DefaultConfig()
	config.Address = fmt.Sprintf("127.0.0.1:%d", port)
	client, err := consul.NewClient(config)
	if err != nil {
		return fmt.Errorf("Fail to build connection with consul agent: %v", err)
	}
	for i := 0; i < times; i++ {
		time.Sleep(delay)
		if leader, _ := client.Status().Leader(); leader != "" {
			return nil
		}
	}
	return fmt.Errorf("Fail to find the leader within time: %v", time.Duration(int64(times))*delay)
}

// Stop stops the consul service.
func (s *consulService) Stop() error {
	if err := s.cmd.Process.Signal(os.Interrupt); err != nil {
		return fmt.Errorf("Fail to stop consul service with INT: %v", err)
	}
	for i := 0; i < consulChkTimesListen; i++ {
		time.Sleep(consulChkDelayListen)
		if !CheckListening(s.port) {
			return nil
		}
	}
	return fmt.Errorf("Fail to stop consul service within time limit: %v",
		consulChkTimesListen*consulChkDelayListen)
}
