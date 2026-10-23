package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"prometheus-llm-proxy/cmd"

	"github.com/rs/zerolog/log"
)

func NewProxyHandler() (*cmd.ProxyHandler, error) {

	_query_map := map[string]cmd.QueryValidation{}
	promUrl := os.Getenv("PROMETHEUS_URL")
	if len(promUrl) == 0 {
		return nil, errors.New("PROMETHEUS_URL environment variable is required")
	}
	llmEndpoint := os.Getenv("LLM_ENDPOINT")
	if len(llmEndpoint) == 0 {
		return nil, errors.New("LLM_ENDPOINT environment variable is required")
	}
	return &cmd.ProxyHandler{
		PromBaseUrl: promUrl,
		LLMEndpoint: llmEndpoint,
		DBHandler: cmd.QueryValidationHandler{
			QueryValidationMap: _query_map,
		},
		Requester: cmd.RequestHandler{
			HTTPClient: &http.Client{
				Timeout: 10 * time.Second,
			},
			LastPrometheusCall: time.Now(),
		},
	}, nil
}

func main() {

	proxy, err := NewProxyHandler()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize proxy handler")
	}
	http.HandleFunc("/api/v1/query_range", proxy.MetricsHandler)
	http.HandleFunc("/api/v1/label/__name__/values", proxy.PrometheusProxyHandler)
	http.HandleFunc("/api/v1/labels", proxy.PrometheusProxyHandler)
	http.HandleFunc("/api/v1/label/que/values", proxy.PrometheusProxyHandler)
	http.HandleFunc("/api/v1/status/buildinfo", proxy.PrometheusProxyHandler)
	http.HandleFunc("/api/v1/query", proxy.PrometheusProxyHandler)
	http.HandleFunc("/api/v1/validate_query", proxy.ValidateQuery)
	http.HandleFunc("/api/v1/get_all_queries", proxy.GetAllQueries)

	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "8000"
	}

	log.Info().Msgf("Starting server on :%s", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil); err != nil {
		log.Err(err).Msg("Error starting server:")
	}
}
