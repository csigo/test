package test

import (
	docker "github.com/fsouza/go-dockerclient"
	gnatsd "github.com/nats-io/nats-server/v2/server"
	gnatsdtest "github.com/nats-io/nats-server/v2/test"
)

func init() {
	RegisterService(Gnatsd, func() Service {
		return &gnatsdService{}
	})
}

type gnatsdService struct {
	port      int
	workDir   string
	gnatsd    *gnatsd.Server
	container *docker.Container
}

func (s *gnatsdService) Start() (string, error) {
	// perform default check
	opts := gnatsdtest.DefaultTestOptions
	opts.Port = gnatsd.RANDOM_PORT
	s.gnatsd = gnatsdtest.RunServer(&opts)
	addr := s.gnatsd.Addr()

	return addr.String(), nil
}

func (s *gnatsdService) Stop() error {
	// close process
	s.gnatsd.Shutdown()
	return nil
}

// StartDocker start the service via docker
func (s *gnatsdService) StartDocker(cl *docker.Client) (ipport string, err error) {
	s.container, ipport, err = StartContainer(
		cl,
		SetImage("nats"),
		SetExposedPorts([]string{"4222/tcp"}),
	)
	return ipport, err
}

// StopDocker stops the service via docker
func (s *gnatsdService) StopDocker(cl *docker.Client) error {
	return RemoveContainer(cl, s.container)
}
