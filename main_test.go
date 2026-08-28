package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	pb "github.com/brotherlogic/scraper/proto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func isExpectedBody(body string) bool {
	return strings.Contains(body, "Disco With A Feeling") ||
		strings.Contains(body, "Cloudflare") ||
		strings.Contains(body, "Just a moment...") ||
		strings.Contains(body, "security verification")
}

func TestScrape(t *testing.T) {
	s := Server{}

	initialSuccess := testutil.ToFloat64(localScrapesTotal.WithLabelValues("success"))

	val, err := s.Scrape(context.Background(), &pb.ScrapeRequest{Url: "https://www.discogs.com/release/14330116-David-Haffner-Disco-With-A-Feeling"})
	if err != nil {
		t.Fatalf("Unable to scrape: %v", err)
	}

	if !isExpectedBody(val.GetBody()) {
		t.Errorf("Scrape failed - did not return correct body: %v", val.GetBody())
	}

	afterSuccess := testutil.ToFloat64(localScrapesTotal.WithLabelValues("success"))
	if afterSuccess <= initialSuccess {
		t.Errorf("Expected localScrapesTotal success count to increase: before %v, after %v", initialSuccess, afterSuccess)
	}
}

func TestMultiScrape(t *testing.T) {
	s := Server{}

	for i := 0; i < 10; i++ {
		val, err := s.Scrape(context.Background(), &pb.ScrapeRequest{Url: "https://www.discogs.com/release/14330116-David-Haffner-Disco-With-A-Feeling"})
		if err != nil {
			t.Fatalf("Unable to scrape: %v", err)
		}

		if !isExpectedBody(val.GetBody()) {
			t.Errorf("Scrape failed - did not return correct body: %v", val.GetBody())
		}
	}
}

func TestRemoteScrapeSuccessMetrics(t *testing.T) {
	os.Setenv("SCRAPE_DO_TOKEN", "test-token")
	defer os.Unsetenv("SCRAPE_DO_TOKEN")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("token") != "test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body>Remote Content</body></html>"))
	}))
	defer ts.Close()

	s := Server{
		httpClient:  ts.Client(),
		scrapeDoURL: ts.URL,
	}

	initialSuccess := testutil.ToFloat64(remoteScrapesTotal.WithLabelValues("success"))

	res, err := s.fallbackScrapeDo("http://example.com")
	if err != nil {
		t.Fatalf("fallbackScrapeDo failed: %v", err)
	}
	if !strings.Contains(res, "Remote Content") {
		t.Errorf("unexpected content: %v", res)
	}

	afterSuccess := testutil.ToFloat64(remoteScrapesTotal.WithLabelValues("success"))
	if afterSuccess != initialSuccess+1 {
		t.Errorf("expected remoteScrapesTotal success to increase by 1: before %v, after %v", initialSuccess, afterSuccess)
	}
}

func TestRemoteScrapeFailureMetricsStatus(t *testing.T) {
	os.Setenv("SCRAPE_DO_TOKEN", "test-token")
	defer os.Unsetenv("SCRAPE_DO_TOKEN")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	s := Server{
		httpClient:  ts.Client(),
		scrapeDoURL: ts.URL,
	}

	initialFailure := testutil.ToFloat64(remoteScrapesTotal.WithLabelValues("failure"))

	_, err := s.fallbackScrapeDo("http://example.com")
	if err == nil {
		t.Fatalf("expected error from fallbackScrapeDo on 500 status")
	}

	afterFailure := testutil.ToFloat64(remoteScrapesTotal.WithLabelValues("failure"))
	if afterFailure != initialFailure+1 {
		t.Errorf("expected remoteScrapesTotal failure to increase by 1: before %v, after %v", initialFailure, afterFailure)
	}
}

func TestRemoteScrapeFailureMetricsNetwork(t *testing.T) {
	os.Setenv("SCRAPE_DO_TOKEN", "test-token")
	defer os.Unsetenv("SCRAPE_DO_TOKEN")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	tsURL := ts.URL
	ts.Close() // Close immediately to cause network failure

	s := Server{
		httpClient:  &http.Client{},
		scrapeDoURL: tsURL,
	}

	initialFailure := testutil.ToFloat64(remoteScrapesTotal.WithLabelValues("failure"))

	_, err := s.fallbackScrapeDo("http://example.com")
	if err == nil {
		t.Fatalf("expected error from fallbackScrapeDo on network error")
	}

	afterFailure := testutil.ToFloat64(remoteScrapesTotal.WithLabelValues("failure"))
	if afterFailure != initialFailure+1 {
		t.Errorf("expected remoteScrapesTotal failure to increase by 1: before %v, after %v", initialFailure, afterFailure)
	}
}

func TestRemoteScrapeNoToken(t *testing.T) {
	os.Unsetenv("SCRAPE_DO_TOKEN")

	s := Server{}
	_, err := s.fallbackScrapeDo("http://example.com")
	if err == nil {
		t.Fatalf("expected error when token is unset")
	}
}

func TestMetricsEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	promhttp.Handler().ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body := w.Body.String()
	if !strings.Contains(body, "scraper_local_scrapes_total") {
		t.Errorf("expected metrics output to contain scraper_local_scrapes_total, got: %s", body)
	}
	if !strings.Contains(body, "scraper_remote_scrapes_total") {
		t.Errorf("expected metrics output to contain scraper_remote_scrapes_total, got: %s", body)
	}
}

