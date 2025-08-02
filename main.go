package main

import (
	"net/http"
	"os"
	"time"

	cmd "github.com/WoodProgrammer/prometheus-llm-proxy/cmd"
	db "github.com/WoodProgrammer/prometheus-llm-proxy/db"

	"github.com/rs/zerolog/log"
)

func NewProxyHandler() *cmd.ProxyHandler {

	_query_map := map[string]db.QueryValidation{}
	promUrl := os.Getenv("PROMETHEUS_URL")
	if len(promUrl) == 0 {
		panic("Please set prometheus url as environment variable")
	}
	llmEndpoint := os.Getenv("LLM_ENDPOINT")
	if len(promUrl) == 0 {
		panic("Please set ollama address url as environment variable")
	}
	return &cmd.ProxyHandler{
		PromBaseUrl: promUrl,
		LLMEndpoint: llmEndpoint,
		DBHandler: db.QueryValidationHandler{
			QueryValidationMap: _query_map,
		},
		Requester: cmd.RequestHandler{
			LastPrometheusCall: time.Now(),
		},
	}
}
func main() {

	proxy := NewProxyHandler()
	http.HandleFunc("/api/v1/query_range", proxy.MetricsHandler)
	http.HandleFunc("/api/v1/label/__name__/values", proxy.PrometheusProxyHandler)
	http.HandleFunc("/api/v1/labels", proxy.PrometheusProxyHandler)
	http.HandleFunc("/api/v1/label/que/values", proxy.PrometheusProxyHandler)

	http.HandleFunc("/api/v1/validate_query", proxy.ValidateQuery)
	http.HandleFunc("/api/v1/get_all_queries", proxy.GetAllQueries)

	log.Info().Msg("Starting server on :8080")
	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Err(err).Msg("Error starting server:")
	}
}
