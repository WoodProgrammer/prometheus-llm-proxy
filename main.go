package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"

	cmd "github.com/WoodProgrammer/prometheus-llm-proxy/cmd"
	"github.com/rs/zerolog/log"
)

func ParseQuery(query string) string {
	re := regexp.MustCompile(`llm_dashboard_metric\{query="([^"]+)"\}`)
	match := re.FindStringSubmatch(query)
	if len(match) == 0 {
		return ""
	}
	return match[1]
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {

	parsedURL, err := url.Parse(r.URL.String())
	if err != nil {
		panic(err)
	}
	queryParams := parsedURL.Query()
	requestHandler := cmd.RequestHandler{}
	query := ParseQuery(queryParams.Get("query"))

	result, err := requestHandler.LLMConverter(query)

	if err != nil {
		log.Err(err).Msg("Error while calling LLM source")
	}
	url := fmt.Sprintf(
		"http://localhost:9090/api/v1/query_range?query=%s&start=%s&end=%s&step=15", result,
		queryParams.Get("start"), queryParams.Get("end"),
	)

	metrics, err := requestHandler.FetchMetrics(url)
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

	req.Header = r.Header.Clone()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to reach Prometheus", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
	w.WriteHeader(resp.StatusCode)

	io.Copy(w, resp.Body)
}

func main() {
	http.HandleFunc("/api/v1/query_range", metricsHandler)
	http.HandleFunc("/api/v1/label/__name__/values", prometheusProxyHandler)
	http.HandleFunc("/api/v1/labels", prometheusProxyHandler)
	http.HandleFunc("/api/v1/label/que/values", prometheusProxyHandler)

	log.Info().Msg("Starting server on :8080")
	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Err(err).Msg("Error starting server:")
	}
}
