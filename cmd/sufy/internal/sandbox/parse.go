/*
 * Copyright (c) 2026 The SUFY Authors (sufy.com). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package sandbox

import (
	"fmt"
	"net/url"
	"strings"

	sdk "github.com/sufy-dev/sufy/sandbox"
)

// DefaultState is the default sandbox state filter when none is specified.
const DefaultState = "running"

// ParseStates parses a comma-separated state list into SandboxState values.
// Empty entries are skipped.
func ParseStates(raw string) []sdk.SandboxState {
	parts := strings.Split(raw, ",")
	states := make([]sdk.SandboxState, 0, len(parts))
	for _, s := range parts {
		s = strings.TrimSpace(s)
		if s != "" {
			states = append(states, sdk.SandboxState(s))
		}
	}
	return states
}

// ParseMetadataQuery converts "k1=v1,k2=v2" into the "k1=v1&k2=v2" URL query
// form expected by the List API's metadata filter.
func ParseMetadataQuery(raw string) string {
	if raw == "" {
		return ""
	}
	var parts []string
	for _, pair := range strings.Split(raw, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 && strings.TrimSpace(kv[0]) != "" && strings.TrimSpace(kv[1]) != "" {
			parts = append(parts, strings.TrimSpace(kv[0])+"="+strings.TrimSpace(kv[1]))
		}
	}
	return strings.Join(parts, "&")
}

// ParseKeyValueMap parses "k1=v1,k2=v2" into a map. Used for metadata, env
// vars, and inline HTTP header sets.
func ParseKeyValueMap(raw string) map[string]string {
	m := make(map[string]string)
	if raw == "" {
		return m
	}
	for _, pair := range strings.Split(raw, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 && strings.TrimSpace(kv[0]) != "" {
			m[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return m
}

// SplitCSV splits a comma-separated string, trimming whitespace and dropping
// empty entries. Returns nil for empty input.
func SplitCSV(raw string) []string {
	if raw == "" {
		return nil
	}
	var out []string
	for _, s := range strings.Split(raw, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// supportedInjectionTypes lists the provider strings accepted by
// BuildInjectionSpec and ParseInlineInjection.
const supportedInjectionTypes = "openai, anthropic, gemini, http"

// BuildInjectionSpec constructs a sdk.InjectionSpec for the given provider
// and configuration. Returns an error for unknown types or invalid URLs.
func BuildInjectionSpec(typ, apiKey, baseURL string, headers map[string]string) (sdk.InjectionSpec, error) {
	switch strings.ToLower(strings.TrimSpace(typ)) {
	case "openai":
		return sdk.InjectionSpec{
			OpenAI: &sdk.OpenAIInjection{APIKey: optionalString(apiKey), BaseURL: optionalString(baseURL)},
		}, nil
	case "anthropic":
		return sdk.InjectionSpec{
			Anthropic: &sdk.AnthropicInjection{APIKey: optionalString(apiKey), BaseURL: optionalString(baseURL)},
		}, nil
	case "gemini":
		return sdk.InjectionSpec{
			Gemini: &sdk.GeminiInjection{APIKey: optionalString(apiKey), BaseURL: optionalString(baseURL)},
		}, nil
	case "http":
		validated, err := validateBaseURL(baseURL, true)
		if err != nil {
			return sdk.InjectionSpec{}, err
		}
		http := &sdk.HTTPInjection{BaseURL: validated}
		if len(headers) > 0 {
			http.Headers = &headers
		}
		return sdk.InjectionSpec{HTTP: http}, nil
	case "":
		return sdk.InjectionSpec{}, fmt.Errorf("injection type is required and must be one of: %s", supportedInjectionTypes)
	default:
		return sdk.InjectionSpec{}, fmt.Errorf("unsupported injection type %q, must be one of: %s", typ, supportedInjectionTypes)
	}
}

// BuildSandboxInjections combines rule-ID references and inline injection
// specs into a single injection list, suitable for CreateParams.Injections.
// Returns nil when both inputs are empty.
func BuildSandboxInjections(ruleIDs, inlineSpecs []string) ([]sdk.SandboxInjectionSpec, error) {
	if len(ruleIDs) == 0 && len(inlineSpecs) == 0 {
		return nil, nil
	}
	out := make([]sdk.SandboxInjectionSpec, 0, len(ruleIDs)+len(inlineSpecs))
	for _, id := range ruleIDs {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			return nil, fmt.Errorf("injection rule ID cannot be empty")
		}
		out = append(out, sdk.SandboxInjectionSpec{ByID: &trimmed})
	}
	for _, raw := range inlineSpecs {
		spec, err := parseInlineSandboxInjection(raw)
		if err != nil {
			return nil, err
		}
		out = append(out, spec)
	}
	return out, nil
}

// parseInlineSandboxInjection parses a single inline injection spec into a
// SandboxInjectionSpec. The inline format is comma-separated "key=value"
// pairs, with the "headers=" key taking the remainder of the string.
func parseInlineSandboxInjection(raw string) (sdk.SandboxInjectionSpec, error) {
	fields := parseInlineInjectionFields(raw)
	spec, err := BuildInjectionSpec(
		fields["type"],
		fields["api-key"],
		fields["base-url"],
		ParseKeyValueMap(fields["headers"]),
	)
	if err != nil {
		return sdk.SandboxInjectionSpec{}, fmt.Errorf("invalid inline injection spec: %w", err)
	}
	return sdk.SandboxInjectionSpec{
		OpenAI:    spec.OpenAI,
		Anthropic: spec.Anthropic,
		Gemini:    spec.Gemini,
		HTTP:      spec.HTTP,
	}, nil
}

// parseInlineInjectionFields splits "k=v,k=v,headers=..." into a flat map.
// The "headers=" key consumes everything up to end-of-string so its value may
// itself contain commas.
func parseInlineInjectionFields(raw string) map[string]string {
	const headersKey = "headers="

	fields := make(map[string]string)
	headersVal := ""
	if idx := strings.Index(raw, ","+headersKey); idx >= 0 {
		headersVal = raw[idx+len(","+headersKey):]
		raw = raw[:idx]
	}
	if strings.HasPrefix(raw, headersKey) {
		headersVal = raw[len(headersKey):]
		raw = ""
	}
	for _, part := range strings.Split(raw, ",") {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		fields[key] = value
	}
	if strings.TrimSpace(headersVal) != "" {
		fields["headers"] = headersVal
	}
	return fields
}

// ParseEnvPairs converts "KEY=VALUE" items into a map. Entries without '=' are
// ignored, matching `docker run -e` behavior.
func ParseEnvPairs(pairs []string) map[string]string {
	if len(pairs) == 0 {
		return nil
	}
	m := make(map[string]string, len(pairs))
	for _, p := range pairs {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		key, value, ok := strings.Cut(p, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		m[key] = strings.TrimSpace(value)
	}
	if len(m) == 0 {
		return nil
	}
	return m
}

func optionalString(v string) *string {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// validateBaseURL returns the trimmed URL when it is a syntactically valid
// http/https URL. When required is false, an empty string is allowed.
func validateBaseURL(raw string, required bool) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		if required {
			return "", fmt.Errorf("base URL is required when injection type is http")
		}
		return "", nil
	}
	u, err := url.Parse(trimmed)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "", fmt.Errorf("base URL must be a valid http/https URL")
	}
	return trimmed, nil
}
