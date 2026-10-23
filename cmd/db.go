package cmd

import (
	"fmt"
	"hash/fnv"
	"sync"
)

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

type QueryValidationHandler struct {
	mu                 sync.RWMutex
	QueryValidationMap map[string]QueryValidation
}

func (q *QueryValidationHandler) ValidateQuery(status bool, hash string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	v, ok := q.QueryValidationMap[hash]
	if !ok {
		v = QueryValidation{}
	}
	q.QueryValidationMap[hash] = v
}

func (q *QueryValidationHandler) SetQueries(prompt, output, hash string, status bool) QueryValidation {
	q.mu.Lock()
	defer q.mu.Unlock()
	query := QueryValidation{
		Prompt: prompt,
		Output: output,
		Status: status,
	}

	q.QueryValidationMap[hash] = query
	return query
}

func (q *QueryValidationHandler) GetQuery(hash string) (QueryValidation, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	v, ok := q.QueryValidationMap[hash]
	return v, ok
}

func (q *QueryValidationHandler) GetAllQueries() map[string]QueryValidation {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.QueryValidationMap
}
