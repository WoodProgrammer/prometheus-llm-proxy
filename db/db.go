package db

import (
	"fmt"
	"hash/fnv"
)

// keep it as a map struct
type QueryValidation struct {
	Prompt string `json:"prompt"`
	Output string `json:"output"`
	Status bool   `json:"status"`
}

func GenerateHash(s string) string {
	h := fnv.New32a()
	h.Write([]byte(s))
	str_hash := fmt.Sprint(h.Sum32())
	return str_hash
}

type QueryValidationInterface interface {
	ValidateQuery(status bool, hash string)
	GetAllQueries() map[string]QueryValidation
}

type QueryValidationHandler struct {
	QueryValidationMap map[string]QueryValidation
}

func (q *QueryValidationHandler) ValidateQuery(status bool, hash string) {
	v, ok := q.QueryValidationMap[hash]
	if !ok {
		v = QueryValidation{}
	}
	q.QueryValidationMap[hash] = v
}

func (q *QueryValidationHandler) SetQueries(prompt, output, hash string, status bool) QueryValidation {
	query := QueryValidation{
		Prompt: prompt,
		Output: output,
		Status: false,
	}

	q.QueryValidationMap[hash] = query
	return query
}

func (q *QueryValidationHandler) GetAllQueries() map[string]QueryValidation {
	return q.QueryValidationMap
}
