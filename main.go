package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

func fetchMetrics(url string) ([]byte, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {

	end := time.Now()
	start := end.Add(-10 * time.Minute) // son 10 dakikanın verisi

	url := fmt.Sprintf(
		"http://localhost:9090/api/v1/query_range?query=up&start=%d&end=%d&step=15",
		start.Unix(), end.Unix(),
	)

	//prometheusURL := "http://localhost:9090/api/v1/query_range?query=up"

	metrics, err := fetchMetrics(url)
	if err != nil {
		http.Error(w, "Failed to fetch metrics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	w.Write(metrics)
}

func main() {
	http.HandleFunc("/api/v1/query_range", metricsHandler)

	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
