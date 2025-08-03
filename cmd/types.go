package cmd

import "time"

var response struct {
	Response string `json:"response"`
}

type Request interface {
	FetchMetrics(url string) ([]byte, error)
	LLMConverter(naturalQuery string) (string, error)
}

type RequestHandler struct {
	PrometheusAvailableMetrics PrometheusAvailableMetricReponse
	LastPrometheusCall         time.Time
}

type QueryValidationRequest struct {
	Hash   string `json:"hash"`
	Status bool   `json:"status"`
}

type PrometheusAvailableMetricReponse struct {
	Status string   `json:"status"`
	Data   []string `json:"data"`
	Error  string   `json:"error"`
}

type Response struct {
	ID     string   `json:"id"`
	Model  string   `json:"model"`
	Output []Output `json:"output"`
}

type Output struct {
	Type    string        `json:"type"` // "message" beklenir
	Role    string        `json:"role"` // "assistant"
	Content []OutputPiece `json:"content"`
}

type OutputPiece struct {
	Type string `json:"type"`           // "output_text" vb.
	Text string `json:"text,omitempty"` // type=output_text ise dolar
	// Başka türler (tool_call, reasoning, etc.) için ek alanlar olabilir.
}
type ResponsesRequest struct {
	Model string      `json:"model"`
	Input interface{} `json:"input"` // string veya []any olabilir
	// İstersen: Instructions, Temperature, MaxOutputTokens vs.
	// Instructions string `json:"instructions,omitempty"`
}
