package test

// This file handles the disque service.

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/fsouza/go-dockerclient"
)

const (
	disqueChkTimes = 40                    // disqueChkTimes is the number of times to retry.
	disqueChkDelay = 50 * time.Millisecond // disqueChkDelay is the waiting time for next retry.
)

func init() {
	RegisterService(Disque, func() Service {
		return &disqueService{}
	})
}

// disqueService is the disque service.
type disqueService struct {
	// port is the port for disque server.
	port int
}

// Start runs the disque service and returns its port.
func (s *disqueService) Start() (ipport string, err error) {
	if err := CheckExecutable("disque-server", "disque"); err != nil {
		return "", fmt.Errorf("Disque is not installed: %v\n", err)
	}

	s.port, err = genDisquePort(minPort, maxPort)
	if err != nil {
		return "", fmt.Errorf("Fail to get port for disque: %v\n", err)
	}

	// Starts disque server.
	cmd := exec.Command(
		"disque-server",
		"--port", fmt.Sprintf("%d", s.port),
		"--daemonize", "yes",
	)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("Fail to start disque server: %v\n", err)
	}

	// Make sure that the server is running.
	for i := 0; i < disqueChkTimes; i++ {
		time.Sleep(disqueChkDelay)
		if CheckListening(s.port) {
			return fmt.Sprintf("localhost:%d", s.port), nil
		}
	}
	return "", fmt.Errorf("Fail to start disque server for port: %v", s.port)
}

// genDisquePort returns the port for disque server.
func genDisquePort(min, max int) (int, error) {
	// Cluster communication port is 10,000 port numbers higher than your Disque node port.
	// By noting that BookPorts() returns available ports from max,
	// we first get the cluster communication port since it is the larger one.
	if max-min < 10000 {
		return 0, fmt.Errorf("Min(%v) and Max(%v) should be separated by at least 10,000.", min, max)
	}
	portCLMin := min + 10000
	ports, err := BookPorts(1)
	if err != nil {
		return 0, fmt.Errorf("Fail to book port: %v", err)
	}
	if len(ports) == 0 {
		return 0, fmt.Errorf("Cannot get available ports.")
	}
	portCL := ports[0]
	if portCL < portCLMin {
		return 0, fmt.Errorf("Port booking(%v) cannot be lower than min(%v)", portCL, portCLMin)
	}
	portSvc := portCL - 10000
	if !portAvailable(int32(portSvc)) {
		return 0, fmt.Errorf("Port(%v) is not available for service.", portSvc)
	}
	return portSvc, nil
}

// Stop stops the disque service.
func (s *disqueService) Stop() error {
	cmd := exec.Command(
		"disque",
		"-p", fmt.Sprintf("%d", s.port),
		"SHUTDOWN",
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Fail to stop the service: %v", err)
	}
	return nil
}

// StartDocker start the service via docker
func (s *disqueService) StartDocker(cl *docker.Client) (string, error) {
	return "", fmt.Errorf("implmenet this")
}

// StopDocker stops the service via docker
func (s *disqueService) StopDocker(cl *docker.Client) error {
	return fmt.Errorf("implmenet this")
}
