package test

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/stretchr/testify/suite"
)

func TestRedisSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skip redis test")
		return
	}
	suite.Run(t, new(redisSuite))
}

type redisSuite struct {
	suite.Suite
}

func (s *redisSuite) TestSerivce() {
	service := &redisService{}

	port, err := service.Start()
	s.NoError(err, "start service error")
	defer service.Stop()

	conn, err := redis.Dial("tcp", fmt.Sprintf("localhost:%s", port))
	s.NoError(err, "get conn error")

	_, err = conn.Do("SET", "aaa", "bbb")
	s.NoError(err, "set data error")

	reply, err := conn.Do("GET", "aaa")
	s.NoError(err, "get data error")
	s.Equal([]byte("bbb"), reply, "data inconsistent")
}

func (s *redisSuite) TestStop() {
	service := &redisService{}
	defer service.Stop()

	port, err := service.Start()
	s.NoError(err, "start service error")

	_, err = net.Listen("tcp", fmt.Sprintf(":%s", port))
	s.Error(err, "port is not listenering")
	service.Stop()

	time.Sleep(3 * time.Second)

	ln, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	s.NoError(err, "port is listenering")
	ln.Close()
}

func (s *redisSuite) TestAuth() {
	service := NewServiceLauncher()
	auth := "password"
	port, stop, err := service.Start(Redis, RedisAuth(auth))
	s.NoError(err, "start service error")
	defer stop()

	conn, err := redis.Dial("tcp", fmt.Sprintf("localhost:%s", port))
	s.NoError(err, "get conn error")

	_, err = conn.Do("SET", "aaa", "bbb")
	s.Error(err, "set data should get error without auth")

	_, err = conn.Do("AUTH", auth)
	s.NoError(err, "auth error")

	_, err = conn.Do("SET", "aaa", "bbb")
	s.NoError(err, "set data error")
}
