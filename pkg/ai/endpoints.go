package ai

import "strings"

// normalizeProviderEndpoint trims provider endpoints to a canonical form so that
// universal clients can safely append API paths without duplicating segments like /v1.
func normalizeProviderEndpoint(provider, endpoint string) string {
	trimmed := strings.TrimSpace(endpoint)
	if trimmed == "" {
		return ""
	}

	trimmed = strings.TrimRight(trimmed, "/")
	normalized := strings.ToLower(strings.TrimSpace(provider))

	if normalized == "openai" || normalized == "deepseek" {
		for strings.HasSuffix(trimmed, "/v1") {
			trimmed = strings.TrimSuffix(trimmed, "/v1")
			trimmed = strings.TrimRight(trimmed, "/")
		}
	}

	return trimmed
}
