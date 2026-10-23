package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"

	"github.com/rs/zerolog/log"
)

type Proxy interface {
	MetricsHandler(w http.ResponseWriter, r *http.Request)
	PrometheusProxyHandler(w http.ResponseWriter, r *http.Request)
}

type ProxyHandler struct {
	PromBaseUrl string
	LLMEndpoint string
	DBHandler   QueryValidationHandler
	Requester   RequestHandler
}

func ParseQuery(query string) string {
	re := regexp.MustCompile(`llm_dashboard_metric\{query="([^"]+)"\}`)
	match := re.FindStringSubmatch(query)
	if len(match) == 0 {
		return ""
	}
	return match[1]
}

func (p *ProxyHandler) ValidateQuery(w http.ResponseWriter, r *http.Request) {
	req := QueryValidationRequest{}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Err(err).Msg("Error while reading body")
		http.Error(w, "Error while reading request body", http.StatusInternalServerError)
		return
	}

	err = json.Unmarshal(body, &req)
	if err != nil {
		log.Err(err).Msg("Error while json.Unmarshal() call")
		http.Error(w, "Error while json.Unmarshal() call", http.StatusInternalServerError)
		return
	}

	query, ok := p.DBHandler.GetQuery(req.Hash)
	if !ok {
		http.NotFound(w, r)
		return
	}
	query.Status = req.Status
	p.DBHandler.SetQueries(query.Prompt, query.Output, req.Hash, req.Status)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte("OK"))

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
		http.Error(w, "Invalid request URL", http.StatusBadRequest)
		return
	}

	queryParams := parsedURL.Query()
	query := ParseQuery(queryParams.Get("query"))

	_hash := GenerateHash(query)
	val, ok := p.DBHandler.GetQuery(_hash)
	if !ok || !val.Status {

		queryForPrometheus, err = p.Requester.LLMConverter(query, p.LLMEndpoint)
		if err != nil {
			log.Err(err).Msg("Error while calling LLM source")
		}
		if queryForPrometheus == "" {
			http.Error(w, "Empty query returned by LLM", http.StatusInternalServerError)
			return
		}
		log.Debug().Msgf("LLM Call required!")
		p.DBHandler.SetQueries(query, queryForPrometheus, _hash, false)

	} else {
		queryForPrometheus = val.Output
		log.Debug().Msgf("There is no LLM call need prompt hash is matching")
		log.Debug().Msgf("The running prompt is %s", val.Prompt)
		log.Debug().Msgf("The running query is %s", queryForPrometheus)
	}

	url := fmt.Sprintf(
		"%s/api/v1/query_range?query=%s&start=%s&end=%s&step=15", p.PromBaseUrl, url.QueryEscape(queryForPrometheus),
		queryParams.Get("start"), queryParams.Get("end"),
	)

	metrics, err := p.Requester.FetchMetrics(url)
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

	resp, err := http.DefaultClient.Do(req)
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
