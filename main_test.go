package main

import (
	"context"
	"strings"
	"testing"

	pb "github.com/brotherlogic/scraper/proto"
)

func TestScrape(t *testing.T) {
	s := Server{}

	val, err := s.Scrape(context.Background(), &pb.ScrapeRequest{Url: "https://www.discogs.com/release/14330116-David-Haffner-Disco-With-A-Feeling"})
	if err != nil {
		t.Fatalf("Unable to scrape: %v", err)
	}

	if !strings.Contains(val.GetBody(), "Disco With A Feeling") {
		t.Errorf("Scrape failed - did not return correct body: %v", val.GetBody())
	}
}

func TestMultiScrape(t *testing.T) {
	s := Server{}

	for i := 0; i < 10; i++ {
		val, err := s.Scrape(context.Background(), &pb.ScrapeRequest{Url: "https://www.discogs.com/release/14330116-David-Haffner-Disco-With-A-Feeling"})
		if err != nil {
			t.Fatalf("Unable to scrape: %v", err)
		}

		if !strings.Contains(val.GetBody(), "Disco With A Feeling") {
			t.Errorf("Scrape failed - did not return correct body: %v", val.GetBody())
		}
	}
}
