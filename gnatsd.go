package test

import (
	gnatsd "github.com/nats-io/gnatsd/server"
	gnatsdtest "github.com/nats-io/gnatsd/test"
)

func init() {
	RegisterService(Gnatsd, func() Service {
		return &gnatsdService{}
	})
}

type gnatsdService struct {
	port    int
	workDir string
	gnatsd  *gnatsd.Server
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
