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
	t := time.Now()
	defer func(t time.Time) {
		log.Printf("Scraped in %v", time.Since(t))
	}(t)

	ctx, cancel, err := cu.New(cu.NewConfig(
		cu.WithHeadless(),
		cu.WithTimeout(time.Minute),
	))
	if err != nil {
		log.Printf("Failed to build CU: %v", err)
		panic(fmt.Sprintf("error building chrome headless: %w", err))
	}
	defer cancel()

	html := ""

	err = chromedp.Run(ctx,
		chromedp.Evaluate(stealth.JS, nil),
		chromedp.Navigate(req.GetUrl()),
		chromedp.WaitReady("body"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			rootNode, err := dom.GetDocument().Do(ctx)
			if err != nil {
				return fmt.Errorf("error getting document: %w", err)
			}
			html, err = dom.GetOuterHTML().WithNodeID(rootNode.NodeID).Do(ctx)
			if err != nil {
				return fmt.Errorf("error getting html: %w", err)
			}
			return err
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("error running chromedp: %w", err)
	}

	log.Printf("Scraped %v", req.GetUrl())
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
