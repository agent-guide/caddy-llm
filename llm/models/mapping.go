package models

import (
	"strings"
)

// Default model mappings for each provider
var DefaultModelMappings = map[string]map[string]string{
	"openai": {
		"haiku": "gpt-4o-mini",
		"sonnet": "gpt-4o",
		"opus":   "gpt-4o",
	},
	"gemini": {
		"haiku":  "gemini-2.0-flash-exp",
		"sonnet": "gemini-2.5-pro-exp",
		"opus":   "gemini-2.5-pro-exp",
	},
	"anthropic": {
		"haiku":  "claude-3-5-haiku-20241022",
		"sonnet": "claude-3-5-sonnet-20241022",
		"opus":   "claude-3-opus-20240229",
	},
	"groq": {
		"haiku":  "llama-3.1-8b-instant",
		"sonnet": "llama-3.3-70b-versatile",
		"opus":   "llama-3.3-70b-versatile",
	},
	"ollama": {
		"haiku":  "llama3.1:8b",
		"sonnet": "llama3.1:70b",
		"opus":   "llama3.1:70b",
	},
}

// GetMappedModel returns the mapped model for a given provider and model name
func GetMappedModel(provider, model string, customMappings map[string]string) string {
	// First check custom mappings
	if customMappings != nil {
		if mapped, ok := customMappings[model]; ok {
			return mapped
		}
	}

	// Then check default mappings for the provider
	if mappings, ok := DefaultModelMappings[provider]; ok {
		// Try exact match
		if mapped, ok := mappings[model]; ok {
			return mapped
		}

		// Try partial match (e.g., "claude-3-5-sonnet" -> "sonnet")
		for key, mapped := range mappings {
			if strings.Contains(model, key) {
				return mapped
			}
		}
	}

	// If no mapping found, return the original model
	return model
}

// DetectModelType detects if a model is haiku, sonnet, or opus based on the name
func DetectModelType(model string) string {
	model = strings.ToLower(model)

	if strings.Contains(model, "haiku") {
		return "haiku"
	}
	if strings.Contains(model, "sonnet") {
		return "sonnet"
	}
	if strings.Contains(model, "opus") {
		return "opus"
	}

	// Default to sonnet for unknown models
	return "sonnet"
}

// ParseModelName extracts the base model name without provider prefix
func ParseModelName(model string) string {
	// Remove provider prefixes
	prefixes := []string{"anthropic/", "openai/", "gemini/", "groq/", "ollama/", "mistral/", "openrouter/"}

	for _, prefix := range prefixes {
		model = strings.TrimPrefix(model, prefix)
	}

	return model
}
