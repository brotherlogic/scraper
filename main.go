package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/chromedp"
	"github.com/go-rod/stealth"
	"google.golang.org/grpc"

	cu "github.com/Davincible/chromedp-undetected"

	pb "github.com/brotherlogic/scraper/proto"
)

var (
	port        = flag.Int("port", 8080, "The server port.")
	metricsPort = flag.Int("metrics_port", 8081, "Metrics port")
)

type Server struct{}

func (s *Server) Scrape(ctx context.Context, req *pb.ScrapeRequest) (*pb.ScrapeResponse, error) {
	ctx, cancel, err := cu.New(cu.NewConfig(
		cu.WithHeadless(),
		cu.WithTimeout(60*time.Second),
	))
	if err != nil {
		return nil, fmt.Errorf("error building chrome headless: %w", err)
	}
	defer cancel()

	html := ""

	err = chromedp.Run(ctx,
		chromedp.Evaluate(stealth.JS, nil),
		chromedp.Navigate(req.GetUrl()),
		chromedp.Sleep(2000*time.Millisecond),
		chromedp.ActionFunc(func(ctx context.Context) error {
			rootNode, err := dom.GetDocument().Do(ctx)
			if err != nil {
				return err
			}
			html, err = dom.GetOuterHTML().WithNodeID(rootNode.NodeID).Do(ctx)
			return err
		}),
	)
	if err != nil {
		return nil, err
	}

	return &pb.ScrapeResponse{Body: html}, nil
}

func main() {
	flag.Parse()

	s := Server{}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen on port %v: %v", *port, err)
	}
	gs := grpc.NewServer()
	pb.RegisterScraperServiceServer(gs, &s)

	if err := gs.Serve(lis); err != nil {
		log.Fatalf("failed to serve grpc: %v", err)
	}
}
