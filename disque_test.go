package test

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

const (
	envProject       = "PROJECT"
	projDeepLearning = "DL"
)

// DisqueSuite is the suite for disque service.
type DisqueSuite struct {
	suite.Suite

	// s is the service to test.
	service *disqueService
}

// SetupSuite runs before all the tests.
func (s *DisqueSuite) SetupSuite() {
	s.service = &disqueService{}
}

// TestDisqueSuite runs all the tests.
func TestDisqueSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip disque long test")
		return
	}
	if os.Getenv(envProject) != projDeepLearning {
		t.Skip("Skip disque test for non-deep learning project.")
		return
	}
	suite.Run(t, new(DisqueSuite))
}

// TestStart tests Start() and Stop() through
func (s *DisqueSuite) TestStartAndStop() {
	// Test the case that the disque service starts.
	ipport, err := s.service.Start()
	s.NoError(err, "No error is expected")
	_, strPort, err := net.SplitHostPort(ipport)
	s.NoError(err, "No error is expected")
	port, _ := strconv.Atoi(strPort)
	isListening := CheckListening(port)
	s.True(isListening, "Should be listening")

	// Test the case that the disque service stops.
	err = s.service.Stop()
	s.NoError(err, "No error is expected")
	isListening = CheckListening(port)
	s.False(isListening, "Should not be listening")
}

// TestDisqueFunction tests whether the disque server works correctly.
func (s *DisqueSuite) TestDisqueFunction() {
	// Start a disque server.
	port, err := s.service.Start()
	s.NoError(err, "No error is expected")
	defer func() {
		err := s.service.Stop()
		s.NoError(err, "No error is expected")
	}()

	// Test enqueue a job.
	queueName := "test_queue"
	job := "test_job"
	cmd := exec.Command(
		"disque",
		"-p", fmt.Sprintf("%v", port),
		"ADDJOB", queueName, job, fmt.Sprintf("0"),
	)
	// Output is of the form: "[job id]\n"
	b, err := cmd.Output()
	s.NoError(err, "No error is expected")
	out := strings.Split(string(b), "\n")
	s.Equal(len(out), 2, "Should be of length 2")
	id := out[0]

	// Test dequeue the job.
	cmd = exec.Command(
		"disque",
		"-p", fmt.Sprintf("%v", port),
		"GETJOB", "FROM", queueName,
	)
	// Output is of the form: "[queue name]\n[job id]\n[job body]\n"
	b, err = cmd.Output()
	s.NoError(err, "No error is expected")
	out = strings.Split(string(b), "\n")
	s.Equal(len(out), 4, "Should be of length 4")
	s.Equal(out[0], queueName, "Should be queue name.")
	s.Equal(out[1], id, "Should be job id.")
	s.Equal(out[2], job, "Should be job body.")
}
