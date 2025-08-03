package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

func (p *RequestHandler) FetchMetrics(url string) ([]byte, error) {
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

func (p *RequestHandler) FetchAvailableMetrics(prometheusAddress string) ([]string, error) {
	req, err := http.NewRequest(http.MethodGet, prometheusAddress+"/api/v1/label/__name__/values", nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 10 * time.Second}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 304 => değişiklik yok
	if resp.StatusCode == http.StatusNotModified {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		err := errors.New(fmt.Sprintf("Error while fetching available metrics status code is %s", resp.StatusCode))
		return nil, err
	}

	payload := p.PrometheusAvailableMetrics
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if payload.Status != "success" {
		return nil, errors.New("prometheus api error: " + payload.Error)
	}

	p.LastPrometheusCall = time.Now()
	p.PrometheusAvailableMetrics = payload
	return payload.Data, nil
}

func (p *RequestHandler) LLMConverter(naturalQuery string, llmEndpoint string) (string, error) {
	var req *http.Request
	var isOpenAI bool
	var llmResponse string
	var openAIAPIModel string
	client := &http.Client{Timeout: 10 * time.Second}
	prompt := fmt.Sprintf(`
Generate a single valid PromQL expression from the request below:
Just return the query, no markdown, no quotes, no explanation
REQUEST: %s

Rules:
- Return only the query on one line. No explanation, markdown, quotes, code fences, or comments.
- Use only these functions when appropriate: rate(), irate(), avg_over_time().
- If the request specifies a time window, use it; otherwise default to [5m] for range vectors.
- Keep metric and label names exactly as given; do not invent metrics or labels.
- Preserve label filters from the request; add by()/without() only if explicitly requested.
- Use rate() for trends over time, irate() for instantaneous/current rate, avg_over_time() for period averages.
- Use offset only if explicitly requested; do not rewrite units.
- If ambiguous, choose the simplest reasonable query.

`, naturalQuery)

	isOpenAI = true
	openAIAPIKey := os.Getenv("OPENAI_API_KEY")
	if len(openAIAPIKey) == 0 {
		isOpenAI = false
	}

	openAIAPIModel = os.Getenv("OPENAI_MODEL")
	if len(openAIAPIModel) == 0 {
		openAIAPIModel = "gpt-4o-mini"
	}

	if !isOpenAI {
		payload := map[string]interface{}{
			"prompt": prompt,
			"stream": false,
			"model":  "mistral",
		}

		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return "", err
		}
		req, err = http.NewRequest("POST", llmEndpoint, bytes.NewBuffer(payloadBytes))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		if err := json.Unmarshal(bodyBytes, &response); err != nil {
			return "", err
		}
		log.Info().Msgf("The Response is %s", response.Response)
		llmResponse = response.Response

	} else {
		log.Info().Msgf("This prometheus-llm-proxy will use OPENAI_MODEL %s", openAIAPIModel)
		payload := map[string]interface{}{
			"input": prompt,
			"model": openAIAPIModel,
		}
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return "", err
		}

		req, err = http.NewRequest("POST", llmEndpoint, bytes.NewBuffer(payloadBytes))
		req.Header.Set("Authorization", "Bearer "+openAIAPIKey)
		req.Header.Set("Content-Type", "application/json")
		if err != nil {
			return "", err
		}

		res, err := client.Do(req)
		if err != nil {
			panic(err)
		}
		defer res.Body.Close()

		if res.StatusCode >= 300 {
			blob, _ := io.ReadAll(res.Body)
			panic(fmt.Errorf("API hata: %s\n%s", res.Status, string(blob)))
		}

		var r Response
		if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
			panic(fmt.Errorf("JSON decode hata: %w", err))
		}
		var out string
		for _, o := range r.Output {
			for _, c := range o.Content {
				if c.Type == "output_text" {
					out += c.Text
				}
			}
		}

		llmResponse = out

	}

	trimmedString := strings.ReplaceAll(llmResponse, "`", "")
	trimmedString = strings.ReplaceAll(trimmedString, " ", "")
	return trimmedString, nil
}
