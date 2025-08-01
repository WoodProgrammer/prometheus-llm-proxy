package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"

	"github.com/WoodProgrammer/prometheus-llm-proxy/db"
	"github.com/rs/zerolog/log"
)

type Proxy interface {
	MetricsHandler(w http.ResponseWriter, r *http.Request)
	PrometheusProxyHandler(w http.ResponseWriter, r *http.Request)
}

type ProxyHandler struct {
	PromBaseUrl string
	LLMEndpoint string
	DBHandler   db.QueryValidationHandler
}

func ParseQuery(query string) string {
	re := regexp.MustCompile(`llm_dashboard_metric\{query="([^"]+)"\}`)
	match := re.FindStringSubmatch(query)
	if len(match) == 0 {
		return ""
	}
	return match[1]
}

func (p *ProxyHandler) GetAllQueries(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p.DBHandler.GetAllQueries())

}

func (p *ProxyHandler) MetricsHandler(w http.ResponseWriter, r *http.Request) {
	var queryForPrometheus string
	parsedURL, err := url.Parse(r.URL.String())
	if err != nil {
		panic(err)
	}
	queryParams := parsedURL.Query()
	requestHandler := RequestHandler{}
	query := ParseQuery(queryParams.Get("query"))

	_hash := db.GenerateHash(query)
	val, ok := p.DBHandler.QueryValidationMap[_hash]
	if !ok {
		queryForPrometheus, err = requestHandler.LLMConverter(query, p.LLMEndpoint)
		if err != nil {
			log.Err(err).Msg("Error while calling LLM source")
		}
		p.DBHandler.SetQueries(query, queryForPrometheus, _hash, false)

	} else {
		queryForPrometheus = val.Output
	}

	url := fmt.Sprintf(
		"%s/api/v1/query_range?query=%s&start=%s&end=%s&step=15", p.PromBaseUrl, queryForPrometheus,
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

func (p *ProxyHandler) PrometheusProxyHandler(w http.ResponseWriter, r *http.Request) {
	targetURL, err := url.Parse(p.PromBaseUrl)
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
