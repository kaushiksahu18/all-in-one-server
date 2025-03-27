package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type PingResult struct {
	Status  string `json:"status"`
	Loss    string `json:"loss"`
	AvgTime string `json:"avg_time"`
	Error   string `json:"error,omitempty"`
}

var (
	websites = []string{"google.com", "https://todoappdb-kaushiksahu18.onrender.com/", "https://theconnect-fa2u.onrender.com/"}
	results  = make(map[string]PingResult)
	mu       sync.Mutex
)

func checkSites() {
	for {
		for _, site := range websites {
			log.Printf("Pinging %s...", site)
			pingResult := PingResult{Status: "checking"}

			out, err := exec.Command("ping", "-c", "1", "-W", "5", site).CombinedOutput()
			mu.Lock()
			if err != nil {
				pingResult.Status = "failed"
				pingResult.Error = err.Error()
				log.Printf("Failed to ping %s: %v", site, err)
			} else {
				pingResult.Status = "success"
				lines := strings.Split(string(out), "\n")
				for _, line := range lines {
					if strings.Contains(line, "packet loss") {
						parts := strings.Split(line, ",")
						for _, part := range parts {
							if strings.Contains(part, "packet loss") {
								pingResult.Loss = strings.TrimSpace(part)
								break
							}
						}
					}
					if strings.Contains(line, "rtt min/avg/max/mdev") {
						parts := strings.Split(line, "=")
						if len(parts) == 2 {
							vals := strings.Split(strings.TrimSpace(parts[1]), "/")
							if len(vals) > 1 {
								pingResult.AvgTime = vals[1] + " ms"
							}
						}
					}
				}
				log.Printf("Successfully pinged %s - Loss: %s, Avg time: %s", site, pingResult.Loss, pingResult.AvgTime)
			}
			results[site] = pingResult
			mu.Unlock()
			time.Sleep(50 * time.Second)
		}
	}
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Ping Service. Go to /ping for results")
}

func main() {
	log.Println("Starting ping service on port 8080")
	go checkSites()
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/ping", pingHandler)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
