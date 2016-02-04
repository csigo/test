package test

import (
	"fmt"
	"os"
	"testing"

	consul "github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/suite"
)

// ConsulSuite is the test suite for consul.
type ConsulSuite struct {
	suite.Suite

	// service is the consul service to test.
	service *consulService
}

// SetupSuite runs before all the tests.
func (s *ConsulSuite) SetupSuite() {
	s.service = &consulService{}
}

// TestConsulSuite runs all the tests in consul suite.
func TestConsulSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip consul long test")
		return
	}
	if os.Getenv(envProject) != projDeepLearning {
		t.Skip("Skip consul test for non-deep learning project.")
		return
	}
	suite.Run(t, new(ConsulSuite))
}

// TestStartAndStop tests Start() and Stop()
func (s *ConsulSuite) TestStartAndStop() {
	port, err := s.service.Start()
	s.NoError(err, "No error is expected")
	isListening := CheckListening(port)
	s.True(isListening, "Should be listening")

	err = s.service.Stop()
	s.NoError(err, "No error is expected")
	isListening = CheckListening(port)
	s.False(isListening, "Should not be listening")
}

// TestRegisterDeregister tests Register() and Deregister() for consul.
func (s *ConsulSuite) TestRegisterDeregister() {
	port, err := s.service.Start()
	s.NoError(err, "No error is expected")
	isListening := CheckListening(port)
	s.True(isListening, "Should be listening")

	config := consul.DefaultConfig()
	config.Address = fmt.Sprintf("127.0.0.1:%d", port)
	client, err := consul.NewClient(config)
	s.NoError(err, "No error is expected")

	expect := map[string]*consul.AgentService{
		"consul": &consul.AgentService{},
	}
	svcs, err := client.Agent().Services()
	s.assertServices(svcs, expect)

	// Test the case that a service is registered.
	testID := "test_id"
	testName := "test_name"
	expect[testID] = &consul.AgentService{
		ID:      testID,
		Service: testName,
	}
	reg := &consul.AgentServiceRegistration{
		ID:   testID,
		Name: testName,
	}
	err = client.Agent().ServiceRegister(reg)
	s.NoError(err, "No error is expected")
	svcs, err = client.Agent().Services()
	s.NoError(err, "No error is expected")
	s.assertServices(svcs, expect)

	// Test the case that a service is deregistered.
	delete(expect, testID)
	err = client.Agent().ServiceDeregister(testID)
	s.NoError(err, "No error is expected")
	svcs, err = client.Agent().Services()
	s.NoError(err, "No error is expected")
	s.assertServices(svcs, expect)

	err = s.service.Stop()
	s.NoError(err, "No error is expected")
	isListening = CheckListening(port)
	s.False(isListening, "Should not be listening")
}

// assertServices checks whether the services input are equal to each other or not.
func (s *ConsulSuite) assertServices(actual map[string]*consul.AgentService, expect map[string]*consul.AgentService) {
	s.Equal(len(actual), len(expect), "Should be of same length")
	for k, v := range expect {
		result, ok := actual[k]
		s.True(ok, "Should have service consul")
		if k == "consul" {
			// Note that the service consul in AgentService contains Server port only,
			// which is different from the HTTP port returned and used by client.
			// Thus, we do not check the whole structure but the id only.
			s.Equal(result.ID, "consul", "ID should be consul")
			continue
		}
		s.Equal(result, v, "Should be as expected")
	}
}
