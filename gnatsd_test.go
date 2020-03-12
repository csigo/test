package test

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/suite"
)

func TestGnatsdSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skip gnatsd test")
		return
	}
	suite.Run(t, new(gnatsdSuite))
}

type gnatsdSuite struct {
	suite.Suite
}

func (s *gnatsdSuite) aTestSerivce() {
	service := &gnatsdService{}

	port, err := service.Start()
	s.NoError(err, "start service error")
	defer service.Stop()

	nc, err := nats.Connect(fmt.Sprintf("nats://localhost:%s", port))
	s.NoError(err, "create conn error")
	c, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
	s.NoError(err, "create encoded conn error")
	defer c.Close()

	testText := "Hello World"
	testTopic := "foo"
	received := false

	// Simple Async Subscriber
	c.Subscribe(testTopic, func(reply string) {
		s.Equal(testText, reply, "reply inconsistent")
		received = true
	})

	// Simple Publisher
	err = c.Publish(testTopic, testText)
	s.NoError(err, "publish msg error")

	time.Sleep(3 * time.Second)
	s.True(received, "not recevied any msgs")
}

func (s *gnatsdSuite) TestStop() {
	service := &gnatsdService{}
	defer service.Stop()

	port, err := service.Start()
	s.NoError(err, "start service error")

	_, err = net.Listen("tcp", fmt.Sprintf("127.0.0.1:%s", port))
	s.Error(err, "port is not listenering")
	service.Stop()

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%s", port))
	s.NoError(err, "port is listenering")
	ln.Close()
}
