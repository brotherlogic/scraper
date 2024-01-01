package scraper_client

import (
	"context"
	"io/ioutil"
	"os"
	"strings"

	pb "github.com/brotherlogic/scraper/proto"
)

type TestClient struct {
}

func GetTestClient() ScraperClient {
	return &TestClient{}
}

func (c *TestClient) Scrape(ctx context.Context, req *pb.ScrapeRequest) (*pb.ScrapeResponse, error) {

	testFile := strings.Replace(strings.Replace(url[23:], "?", "_", -1), "&", "_", -1)

	stat, err := os.Stat("testdata" + testFile)
	if err != nil {
		return nil, err
	}

	adder := ""
	if stat.IsDir() {
		adder = "/FILE"
	}

	blah, err := ioutil.ReadFile("testdata" + testFile + adder)
	return &pb.ScrapeResponse{Body: string(blah)}, err
}
