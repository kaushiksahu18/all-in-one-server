package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// PingResult represents the results of a website health check
type PingResult struct {
	Status  string `json:"status"`
	Loss    string `json:"loss"`
	AvgTime string `json:"avg_time"`
	Error   string `json:"error,omitempty"`
}

// WebsiteMonitor manages website health checking
type WebsiteMonitor struct {
	websites []string
	results  map[string]PingResult
	mu       sync.RWMutex
}

// NewWebsiteMonitor creates a new monitor with the given websites
func NewWebsiteMonitor(websites []string) *WebsiteMonitor {
	return &WebsiteMonitor{
		websites: websites,
		results:  make(map[string]PingResult),
	}
}

// StartMonitoring begins continuous checking of websites
func (wm *WebsiteMonitor) StartMonitoring(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(2 * time.Minute)
		defer ticker.Stop()

		// Do an initial check of all sites
		wm.checkAllSites()

		for {
			select {
			case <-ticker.C:
				wm.checkAllSites()
			case <-ctx.Done():
				log.Println("Monitoring stopped")
				return
			}
		}
	}()
}

// checkAllSites performs health checks on all configured websites
func (wm *WebsiteMonitor) checkAllSites() {
	for _, site := range wm.websites {
		go func(site string) {
			log.Printf("Checking %s...", site)
			result := wm.httpCheck(site)

			wm.mu.Lock()
			wm.results[site] = result
			wm.mu.Unlock()

			log.Printf("HTTP check for %s - Status: %s, Loss: %s, Avg time: %s",
				site, result.Status, result.Loss, result.AvgTime)
		}(site)
	}
}

// httpCheck performs an HTTP request to check website health
func (wm *WebsiteMonitor) httpCheck(site string) PingResult {
	// Make sure the URL has a scheme
	if !strings.HasPrefix(site, "http://") && !strings.HasPrefix(site, "https://") {
		site = "https://" + site
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, site, nil)
	if err != nil {
		return PingResult{
			Status: "failed",
			Loss:   "100%",
			Error:  fmt.Sprintf("Failed to create request: %v", err),
		}
	}

	start := time.Now()
	client := &http.Client{}
	resp, err := client.Do(req)
	duration := time.Since(start)

	if err != nil {
		return PingResult{
			Status: "failed",
			Loss:   "100%",
			Error:  fmt.Sprintf("Request failed: %v", err),
		}
	}
	defer resp.Body.Close()

	return PingResult{
		Status:  "success",
		Loss:    "0%",
		AvgTime: fmt.Sprintf("%.2f ms", float64(duration.Milliseconds())),
	}
}

// GetResults returns the current monitoring results
func (wm *WebsiteMonitor) GetResults() map[string]PingResult {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	// Create a copy to avoid external modification
	resultsCopy := make(map[string]PingResult, len(wm.results))
	for k, v := range wm.results {
		resultsCopy[k] = v
	}

	return resultsCopy
}

func main() {
	log.Println("Starting HTTP check service on port 8080")

	websites := []string{
		"google.com",
		"https://todoappdb-kaushiksahu18.onrender.com/",
		"https://theconnect-fa2u.onrender.com/",
	}

	monitor := NewWebsiteMonitor(websites)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	monitor.StartMonitoring(ctx)

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "HTTP Check Service. Go to /ping for results")
	})

	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(monitor.GetResults())
	})

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
