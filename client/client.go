package scraper_client

import (
	"context"

	pb "github.com/brotherlogic/scraper/proto"

	"google.golang.org/grpc"
)

type ScraperClient interface {
	Scrape(ctx context.Context, req *pb.ScrapeRequest) (*pb.ScrapeResponse, error)
}

type sClient struct {
	gClient pb.ScraperServiceClient
}

func GetClient() (ScraperClient, error) {
	conn, err := grpc.Dial("scraper.scraper:8080", grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return &sClient{gClient: pb.NewScraperServiceClient(conn)}, nil
}

func (c *sClient) Scrape(ctx context.Context, req *pb.ScrapeRequest) (*pb.ScrapeResponse, error) {
	return c.gClient.Scrape(ctx, req)
}
