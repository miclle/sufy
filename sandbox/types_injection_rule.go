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
	"time"

	"github.com/sufy-dev/sufy/sandbox/internal/apis"
)

// ---------------------------------------------------------------------------
// SDK-native types — request injection configuration (and saved rules)
// ---------------------------------------------------------------------------

// OpenAIInjection configures an OpenAI-compatible API injection. The sandbox
// automatically attaches an "Authorization: Bearer <APIKey>" header. Default
// host is api.openai.com.
type OpenAIInjection struct {
	// APIKey is the OpenAI API key, optional.
	APIKey *string

	// BaseURL is an optional base URL. Defaults to api.openai.com when unset.
	BaseURL *string
}

// AnthropicInjection configures an Anthropic API injection. The sandbox
// automatically attaches an "x-api-key: <APIKey>" header. Default host is
// api.anthropic.com.
type AnthropicInjection struct {
	// APIKey is the Anthropic API key, optional.
	APIKey *string

	// BaseURL is an optional base URL. Defaults to api.anthropic.com when unset.
	BaseURL *string
}

// GeminiInjection configures a Google Gemini API injection. The sandbox
// automatically attaches an "x-goog-api-key: <APIKey>" header. Default host
// is generativelanguage.googleapis.com.
type GeminiInjection struct {
	// APIKey is the Gemini API key, optional.
	APIKey *string

	// BaseURL is an optional base URL. Defaults to
	// generativelanguage.googleapis.com when unset.
	BaseURL *string
}

// HTTPInjection configures a custom HTTPS header injection.
type HTTPInjection struct {
	// BaseURL is the base URL to match outgoing HTTPS requests against. The
	// host component is used for matching. Scheme defaults to https.
	BaseURL string

	// Headers is an optional map of headers to inject or override.
	Headers *map[string]string
}

// InjectionSpec is a discriminated union that describes a single injection
// configuration. Exactly one field must be set.
type InjectionSpec struct {
	// OpenAI selects an OpenAI-compatible injection.
	OpenAI *OpenAIInjection

	// Anthropic selects an Anthropic injection.
	Anthropic *AnthropicInjection

	// Gemini selects a Google Gemini injection.
	Gemini *GeminiInjection

	// HTTP selects a custom HTTPS injection.
	HTTP *HTTPInjection
}

// SandboxInjectionSpec is a discriminated union that describes an injection
// for a sandbox. It can reference an existing saved rule by ID or embed a
// full InjectionSpec. Exactly one field must be set.
type SandboxInjectionSpec struct {
	// ByID references a previously saved injection rule by ID.
	ByID *string

	// OpenAI selects an OpenAI-compatible injection.
	OpenAI *OpenAIInjection

	// Anthropic selects an Anthropic injection.
	Anthropic *AnthropicInjection

	// Gemini selects a Google Gemini injection.
	Gemini *GeminiInjection

	// HTTP selects a custom HTTPS injection.
	HTTP *HTTPInjection
}

// InjectionRule is a saved injection rule that can be referenced by sandboxes.
type InjectionRule struct {
	// RuleID is the unique identifier for the rule.
	RuleID string

	// Name is the human-readable rule name, unique per user.
	Name string

	// Injection is the injection configuration for this rule.
	Injection InjectionSpec

	// CreatedAt is the rule creation timestamp.
	CreatedAt time.Time

	// UpdatedAt is the last update timestamp.
	UpdatedAt time.Time
}

// CreateInjectionRuleParams holds parameters for creating an injection rule.
type CreateInjectionRuleParams struct {
	// Name is the rule name. Required; must be unique per user.
	Name string

	// Injection is the injection configuration. Required.
	Injection InjectionSpec
}

func (p *CreateInjectionRuleParams) toAPI() (apis.PostInjectionRulesJSONRequestBody, error) {
	inj, err := injectionSpecToAPI(p.Injection)
	if err != nil {
		return apis.PostInjectionRulesJSONRequestBody{}, err
	}
	return apis.PostInjectionRulesJSONRequestBody{
		Name:      p.Name,
		Injection: inj,
	}, nil
}

// UpdateInjectionRuleParams holds parameters for updating an injection rule.
type UpdateInjectionRuleParams struct {
	// Name is the new rule name. Optional.
	Name *string

	// Injection is the new injection configuration. Optional.
	Injection *InjectionSpec
}

func (p *UpdateInjectionRuleParams) toAPI() (apis.PutInjectionRulesRuleIDJSONRequestBody, error) {
	body := apis.PutInjectionRulesRuleIDJSONRequestBody{
		Name: p.Name,
	}
	if p.Injection != nil {
		inj, err := injectionSpecToAPI(*p.Injection)
		if err != nil {
			return body, err
		}
		body.Injection = &inj
	}
	return body, nil
}

// ---------------------------------------------------------------------------
// Conversion helpers — SDK → apis
// ---------------------------------------------------------------------------

func injectionSpecToAPI(spec InjectionSpec) (apis.Injection, error) {
	count := 0
	if spec.OpenAI != nil {
		count++
	}
	if spec.Anthropic != nil {
		count++
	}
	if spec.Gemini != nil {
		count++
	}
	if spec.HTTP != nil {
		count++
	}
	if count == 0 {
		return apis.Injection{}, fmt.Errorf("InjectionSpec: exactly one injection type must be set (OpenAI, Anthropic, Gemini, or HTTP), got none")
	}
	if count > 1 {
		return apis.Injection{}, fmt.Errorf("InjectionSpec: exactly one injection type must be set, but got %d", count)
	}

	var inj apis.Injection
	var err error
	switch {
	case spec.OpenAI != nil:
		err = inj.FromOpenaiInjection(apis.OpenaiInjection{
			APIKey:  spec.OpenAI.APIKey,
			BaseURL: spec.OpenAI.BaseURL,
			Type:    apis.Openai,
		})
	case spec.Anthropic != nil:
		err = inj.FromAnthropicInjection(apis.AnthropicInjection{
			APIKey:  spec.Anthropic.APIKey,
			BaseURL: spec.Anthropic.BaseURL,
			Type:    apis.Anthropic,
		})
	case spec.Gemini != nil:
		err = inj.FromGeminiInjection(apis.GeminiInjection{
			APIKey:  spec.Gemini.APIKey,
			BaseURL: spec.Gemini.BaseURL,
			Type:    apis.Gemini,
		})
	case spec.HTTP != nil:
		err = inj.FromHTTPInjection(apis.HTTPInjection{
			BaseURL: spec.HTTP.BaseURL,
			Headers: spec.HTTP.Headers,
			Type:    apis.HTTP,
		})
	}
	return inj, err
}

func sandboxInjectionSpecToAPI(spec SandboxInjectionSpec) (apis.SandboxInjection, error) {
	count := 0
	if spec.ByID != nil {
		count++
	}
	if spec.OpenAI != nil {
		count++
	}
	if spec.Anthropic != nil {
		count++
	}
	if spec.Gemini != nil {
		count++
	}
	if spec.HTTP != nil {
		count++
	}
	if count == 0 {
		return apis.SandboxInjection{}, fmt.Errorf("SandboxInjectionSpec: exactly one injection type must be set (ByID, OpenAI, Anthropic, Gemini, or HTTP), got none")
	}
	if count > 1 {
		return apis.SandboxInjection{}, fmt.Errorf("SandboxInjectionSpec: exactly one injection type must be set, but got %d", count)
	}

	var si apis.SandboxInjection
	var err error
	switch {
	case spec.ByID != nil:
		err = si.FromInjectionByID(apis.InjectionByID{
			ID:   *spec.ByID,
			Type: apis.ID,
		})
	case spec.OpenAI != nil:
		err = si.FromOpenaiInjection(apis.OpenaiInjection{
			APIKey:  spec.OpenAI.APIKey,
			BaseURL: spec.OpenAI.BaseURL,
			Type:    apis.Openai,
		})
	case spec.Anthropic != nil:
		err = si.FromAnthropicInjection(apis.AnthropicInjection{
			APIKey:  spec.Anthropic.APIKey,
			BaseURL: spec.Anthropic.BaseURL,
			Type:    apis.Anthropic,
		})
	case spec.Gemini != nil:
		err = si.FromGeminiInjection(apis.GeminiInjection{
			APIKey:  spec.Gemini.APIKey,
			BaseURL: spec.Gemini.BaseURL,
			Type:    apis.Gemini,
		})
	case spec.HTTP != nil:
		err = si.FromHTTPInjection(apis.HTTPInjection{
			BaseURL: spec.HTTP.BaseURL,
			Headers: spec.HTTP.Headers,
			Type:    apis.HTTP,
		})
	}
	return si, err
}

// ---------------------------------------------------------------------------
// Conversion helpers — apis → SDK
// ---------------------------------------------------------------------------

func injectionSpecFromAPI(inj apis.Injection) (InjectionSpec, error) {
	disc, err := inj.Discriminator()
	if err != nil {
		return InjectionSpec{}, err
	}
	switch disc {
	case string(apis.Openai):
		v, err := inj.AsOpenaiInjection()
		if err != nil {
			return InjectionSpec{}, err
		}
		return InjectionSpec{OpenAI: &OpenAIInjection{APIKey: v.APIKey, BaseURL: v.BaseURL}}, nil
	case string(apis.Anthropic):
		v, err := inj.AsAnthropicInjection()
		if err != nil {
			return InjectionSpec{}, err
		}
		return InjectionSpec{Anthropic: &AnthropicInjection{APIKey: v.APIKey, BaseURL: v.BaseURL}}, nil
	case string(apis.Gemini):
		v, err := inj.AsGeminiInjection()
		if err != nil {
			return InjectionSpec{}, err
		}
		return InjectionSpec{Gemini: &GeminiInjection{APIKey: v.APIKey, BaseURL: v.BaseURL}}, nil
	case string(apis.HTTP):
		v, err := inj.AsHTTPInjection()
		if err != nil {
			return InjectionSpec{}, err
		}
		return InjectionSpec{HTTP: &HTTPInjection{BaseURL: v.BaseURL, Headers: v.Headers}}, nil
	default:
		return InjectionSpec{}, fmt.Errorf("unknown injection type: %s", disc)
	}
}

func injectionRuleFromAPI(a apis.InjectionRule) (InjectionRule, error) {
	spec, err := injectionSpecFromAPI(a.Injection)
	if err != nil {
		return InjectionRule{}, err
	}
	return InjectionRule{
		RuleID:    a.RuleID,
		Name:      a.Name,
		Injection: spec,
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt,
	}, nil
}

func injectionRulesFromAPI(a []apis.InjectionRule) ([]InjectionRule, error) {
	if a == nil {
		return nil, nil
	}
	result := make([]InjectionRule, len(a))
	for i, r := range a {
		rule, err := injectionRuleFromAPI(r)
		if err != nil {
			return nil, err
		}
		result[i] = rule
	}
	return result, nil
}
