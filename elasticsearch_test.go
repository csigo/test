package test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/olivere/elastic"
	"github.com/stretchr/testify/suite"
)

func TestElasticSearchSuite(t *testing.T) {
	t.Skip("skip elastic search test")
	suite.Run(t, new(elasticSearchSuite))
}

type elasticSearchSuite struct {
	suite.Suite
}

func (s *elasticSearchSuite) TestSerivce() {
	service := &esService{}

	// start server
	port, err := service.Start()
	s.NoError(err, "start service error")
	_, err = net.Listen("tcp", fmt.Sprintf(":%s", port))
	s.Error(err, "port is not listening")
	defer service.Stop()

	// do db operation
	url := fmt.Sprintf("http://localhost:%s", port)
	client, err := elastic.NewClient(elastic.SetURL(url), elastic.SetSniff(false))
	s.NoError(err)

	index := "csi.common.test"
	ctx := context.Background()

	// clear index before test
	_, err = client.DeleteIndex(index).Do(ctx)
	s.NoError(err)
	// create index
	createResp, err := client.CreateIndex(index).Do(ctx)
	s.NoError(err)
	s.True(createResp.Acknowledged)
	// clear index after test
	deleteResp, err := client.DeleteIndex(index).Do(ctx)
	s.NoError(err)
	s.True(deleteResp.Acknowledged)

	// stop server
	service.Stop()
	time.Sleep(3 * time.Second)
	ln, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	s.NoError(err, "port is listening")
	ln.Close()
}
