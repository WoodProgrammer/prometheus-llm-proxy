package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
)

func parseQuery(query string) string {
	re := regexp.MustCompile(`llm_dashboard_metric\{query="([^"]+)"\}`)

	match := re.FindStringSubmatch(query)
	if len(match) > 1 {
		fmt.Println("Captured query:", match[1])
	} else {
		fmt.Println("No match found.")
	}
	return match[1]
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {

	parsedURL, err := url.Parse(r.URL.String())
	if err != nil {
		panic(err)
	}
	queryParams := parsedURL.Query()

	query := parseQuery(queryParams.Get("query"))

	main_result := LLMConverter(query)

	url := fmt.Sprintf(
		"http://localhost:9090/api/v1/query_range?query=%s&start=%s&end=%s&step=15", main_result,
		queryParams.Get("start"), queryParams.Get("end"),
	)

	metrics, err := fetchMetrics(url)
	if err != nil {
		http.Error(w, "Failed to fetch metrics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	w.Write(metrics)
}

func prometheusProxyHandler(w http.ResponseWriter, r *http.Request) {
	targetURL, err := url.Parse("http://localhost:9090")
	if err != nil {
		http.Error(w, "Invalid Prometheus URL", http.StatusInternalServerError)
		return
	}
	targetURL.Path = r.URL.Path
	targetURL.RawQuery = r.URL.RawQuery

	req, err := http.NewRequest(r.Method, targetURL.String(), r.Body)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	// Orijinal başlıkları kopyala
	req.Header = r.Header.Clone()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to reach Prometheus", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Yanıt başlıklarını kopyala
	for k, v := range resp.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
	w.WriteHeader(resp.StatusCode)

	// Yanıt içeriğini aktar
	io.Copy(w, resp.Body)
}

func main() {
	http.HandleFunc("/api/v1/query_range", metricsHandler)
	http.HandleFunc("/api/v1/label/__name__/values", prometheusProxyHandler)
	http.HandleFunc("/api/v1/labels", prometheusProxyHandler)
	http.HandleFunc("/api/v1/label/que/values", prometheusProxyHandler)

	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
