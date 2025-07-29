package internal

import (
	"regexp"
)

func ParseQuery(query string) string {
	re := regexp.MustCompile(`llm_dashboard_metric\{query="([^"]+)"\}`)
	match := re.FindStringSubmatch(query)
	if len(match) == 0 {
		return ""
	}
	return match[1]
}
