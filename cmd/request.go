package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

type Request interface {
	FetchMetrics(url string) ([]byte, error)
	LLMConverter(naturalQuery string) (string, error)
}

type RequestHandler struct {
}

var response struct {
	Response string `json:"response"`
}

type QueryValidationRequest struct {
	Hash   string `json:"hash"`
	Status bool   `json:"status"`
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

func (p *RequestHandler) LLMConverter(naturalQuery string, llmEndpoint string) (string, error) {
	var req *http.Request
	var isOpenAI bool
	var openAIAPIModel string
	client := &http.Client{Timeout: 10 * time.Second}

	prompt := fmt.Sprintf(`
Generate promql for this content: '%s' please only return the query
Only return the query. No explanation, no markdown, no quotes.
`, naturalQuery)

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
	} else {

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

	trimmedString := strings.ReplaceAll(response.Response, "`", "")
	trimmedString = strings.ReplaceAll(trimmedString, " ", "")
	return trimmedString, nil
}
