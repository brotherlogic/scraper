package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/chromedp"
	"github.com/go-rod/stealth"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"

	cu "github.com/Davincible/chromedp-undetected"

	pb "github.com/brotherlogic/scraper/proto"
)

var (
	port        = flag.Int("port", 8080, "The server port.")
	metricsPort = flag.Int("metrics_port", 8081, "Metrics port")

	localScrapesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "scraper_local_scrapes_total",
			Help: "Total number of local scrapes.",
		},
		[]string{"result"},
	)
	localScrapeLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "scraper_local_scrape_latency_seconds",
			Help:    "Latency of local scrapes in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"result"},
	)

	remoteScrapesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "scraper_remote_scrapes_total",
			Help: "Total number of remote scrapes to scrape.do.",
		},
		[]string{"result"},
	)
	remoteScrapeLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "scraper_remote_scrape_latency_seconds",
			Help:    "Latency of remote scrapes in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"result"},
	)
)

// Server represents the scraper gRPC server.
type Server struct {
	httpClient  *http.Client
	scrapeDoURL string
}

func (s *Server) getHTTPClient() *http.Client {
	if s.httpClient != nil {
		return s.httpClient
	}
	return http.DefaultClient
}

func (s *Server) getScrapeDoURL() string {
	if s.scrapeDoURL != "" {
		return s.scrapeDoURL
	}
	return "http://api.scrape.do"
}

func (s *Server) fallbackScrapeDo(targetURL string) (string, error) {
	token := os.Getenv("SCRAPE_DO_TOKEN")
	if token == "" {
		log.Printf("SCRAPE_DO_TOKEN not set, cannot fallback")
		return "", fmt.Errorf("SCRAPE_DO_TOKEN not set")
	}

	reqUrl := fmt.Sprintf("%s?token=%s&url=%s", s.getScrapeDoURL(), token, targetURL)
	rStart := time.Now()
	resp, err := s.getHTTPClient().Get(reqUrl)
	if err != nil {
		remoteScrapesTotal.WithLabelValues("failure").Inc()
		remoteScrapeLatency.WithLabelValues("failure").Observe(time.Since(rStart).Seconds())
		log.Printf("Error requesting scrape.do: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil || resp.StatusCode != http.StatusOK {
		remoteScrapesTotal.WithLabelValues("failure").Inc()
		remoteScrapeLatency.WithLabelValues("failure").Observe(time.Since(rStart).Seconds())
		if err != nil {
			log.Printf("Error reading scrape.do response: %v", err)
			return "", err
		}
		log.Printf("Error response from scrape.do: status %d", resp.StatusCode)
		return "", fmt.Errorf("scrape.do returned status %d", resp.StatusCode)
	}

	remoteScrapesTotal.WithLabelValues("success").Inc()
	remoteScrapeLatency.WithLabelValues("success").Observe(time.Since(rStart).Seconds())
	return string(bodyBytes), nil
}

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
		localScrapesTotal.WithLabelValues("failure").Inc()
		localScrapeLatency.WithLabelValues("failure").Observe(time.Since(t).Seconds())
		log.Printf("Failed to build CU: %v", err)
		panic(fmt.Sprintf("error building chrome headless: %v", err))
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
		localScrapesTotal.WithLabelValues("failure").Inc()
		localScrapeLatency.WithLabelValues("failure").Observe(time.Since(t).Seconds())
		return nil, fmt.Errorf("error running chromedp: %w", err)
	}

	localScrapesTotal.WithLabelValues("success").Inc()
	localScrapeLatency.WithLabelValues("success").Observe(time.Since(t).Seconds())

	if strings.Contains(html, "<title>Just a moment...</title>") || strings.Contains(html, "Cloudflare") {
		log.Printf("Detected Cloudflare, falling back to scrape.do")
		if remoteHTML, err := s.fallbackScrapeDo(req.GetUrl()); err == nil {
			html = remoteHTML
		}
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

	http.Handle("/metrics", promhttp.Handler())
	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", *metricsPort), nil); err != nil {
			log.Fatalf("failed to serve metrics: %v", err)
		}
	}()

	if err := gs.Serve(lis); err != nil {
		log.Fatalf("failed to serve grpc: %v", err)
	}
}
