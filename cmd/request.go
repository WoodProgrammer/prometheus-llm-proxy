package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type Request interface {
	FetchMetrics(url string) ([]byte, error)
	LLMConverter(naturalQuery string) (string, error)
}

type RequestHandler struct {
}

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

func (p *RequestHandler) LLMConverter(naturalQuery string) (string, error) {

	payload := map[string]interface{}{
		"prompt": fmt.Sprintf(`
Generate promql for this content: '%s' please only return the query
Only return the query. No explanation, no markdown, no quotes.
`, naturalQuery),
		"stream": false,
		"model":  "mistral",
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", "http://localhost:11434/api/generate", bytes.NewBuffer(payloadBytes))
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

	var response struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return "", err
	}

	trimmedString := strings.ReplaceAll(response.Response, "`", "")
	trimmedString = strings.ReplaceAll(trimmedString, " ", "")
	return trimmedString, nil
}
