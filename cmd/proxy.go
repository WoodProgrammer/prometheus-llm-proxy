package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"time"

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

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Err(err).Msg("Error while reading body")
	}

	err = json.Unmarshal(body, &req)
	if err != nil {
		log.Err(err).Msg("Error while json.Unmarshal() call")
		http.Error(w, "Error while json.Unmarshal() call", http.StatusInternalServerError)
		return
	}

	query, ok := p.DBHandler.QueryValidationMap[req.Hash]

	if !ok {
		http.NotFound(w, r)
		return
	}
	query.Status = req.Status
	p.DBHandler.QueryValidationMap[req.Hash] = query

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
		panic(err)
	}
	queryParams := parsedURL.Query()
	query := ParseQuery(queryParams.Get("query"))

	_hash := db.GenerateHash(query)
	val, ok := p.DBHandler.QueryValidationMap[_hash]
	if !ok || !val.Status {

		if time.Since(p.Requester.LastPrometheusCall) >= 1*time.Minute || len(p.Requester.PrometheusAvailableMetrics.Data) == 0 {
			log.Info().Msgf("Cagir abi %s", p.Requester.LastPrometheusCall)
			p.Requester.FetchAvailableMetrics(p.PromBaseUrl)
		} else {
			log.Info().Msgf("The recent prometheus call has been in %s", p.Requester.LastPrometheusCall)
		}

		queryForPrometheus, err = p.Requester.LLMConverter(query, p.LLMEndpoint)
		if err != nil {
			log.Err(err).Msg("Error while calling LLM source")
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
		"%s/api/v1/query_range?query=%s&start=%s&end=%s&step=15", p.PromBaseUrl, queryForPrometheus,
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
