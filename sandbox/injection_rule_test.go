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
	"context"
	"testing"
	"time"

	"github.com/sufy-dev/sufy/sandbox/internal/apis"
)

func strPtr(s string) *string { return &s }

// newOpenaiAPIInjection builds an apis.Injection populated with an openai variant.
func newOpenaiAPIInjection(apiKey string) apis.Injection {
	var inj apis.Injection
	_ = inj.FromOpenaiInjection(apis.OpenaiInjection{
		APIKey: strPtr(apiKey),
		Type:   apis.Openai,
	})
	return inj
}

func TestListInjectionRules(t *testing.T) {
	now := time.Now()
	rules := []apis.InjectionRule{
		{
			RuleID:    "rule-1",
			Name:      "openai-key",
			Injection: newOpenaiAPIInjection("sk-a"),
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	mock := &mockAPI{
		getInjectionRulesFn: func(ctx context.Context, editors ...apis.RequestEditorFn) (*apis.GetInjectionRulesResponse, error) {
			return &apis.GetInjectionRulesResponse{
				JSON200:      &rules,
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	got, err := c.ListInjectionRules(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(got))
	}
	if got[0].RuleID != "rule-1" || got[0].Name != "openai-key" {
		t.Errorf("unexpected rule: %+v", got[0])
	}
	if got[0].Injection.OpenAI == nil || got[0].Injection.OpenAI.APIKey == nil || *got[0].Injection.OpenAI.APIKey != "sk-a" {
		t.Errorf("expected OpenAI injection with APIKey 'sk-a', got %+v", got[0].Injection)
	}
}

func TestCreateInjectionRule(t *testing.T) {
	now := time.Now()
	var receivedBody apis.PostInjectionRulesJSONRequestBody
	mock := &mockAPI{
		postInjectionRulesFn: func(ctx context.Context, body apis.PostInjectionRulesJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.PostInjectionRulesResponse, error) {
			receivedBody = body
			return &apis.PostInjectionRulesResponse{
				JSON201: &apis.InjectionRule{
					RuleID:    "rule-42",
					Name:      body.Name,
					Injection: body.Injection,
					CreatedAt: now,
					UpdatedAt: now,
				},
				HTTPResponse: httpResponse(201),
			}, nil
		},
	}
	c := newTestClient(mock)
	rule, err := c.CreateInjectionRule(context.Background(), CreateInjectionRuleParams{
		Name: "anthropic-key",
		Injection: InjectionSpec{
			Anthropic: &AnthropicInjection{APIKey: strPtr("ak-1")},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rule.RuleID != "rule-42" || rule.Name != "anthropic-key" {
		t.Errorf("unexpected rule: %+v", rule)
	}
	if receivedBody.Name != "anthropic-key" {
		t.Errorf("expected body name 'anthropic-key', got %q", receivedBody.Name)
	}
	disc, err := receivedBody.Injection.Discriminator()
	if err != nil {
		t.Fatalf("discriminator error: %v", err)
	}
	if disc != string(apis.Anthropic) {
		t.Errorf("expected discriminator %q, got %q", apis.Anthropic, disc)
	}
}

func TestCreateInjectionRuleValidationError(t *testing.T) {
	// No injection type set → should fail before any API call is made.
	mock := &mockAPI{}
	c := newTestClient(mock)
	_, err := c.CreateInjectionRule(context.Background(), CreateInjectionRuleParams{
		Name:      "empty",
		Injection: InjectionSpec{},
	})
	if err == nil {
		t.Fatal("expected error for empty InjectionSpec")
	}
}

func TestCreateInjectionRuleMultipleFields(t *testing.T) {
	// More than one injection type set → validation error.
	mock := &mockAPI{}
	c := newTestClient(mock)
	_, err := c.CreateInjectionRule(context.Background(), CreateInjectionRuleParams{
		Name: "both",
		Injection: InjectionSpec{
			OpenAI:    &OpenAIInjection{APIKey: strPtr("sk")},
			Anthropic: &AnthropicInjection{APIKey: strPtr("ak")},
		},
	})
	if err == nil {
		t.Fatal("expected error when multiple injection types are set")
	}
}

func TestGetInjectionRule(t *testing.T) {
	now := time.Now()
	mock := &mockAPI{
		getInjectionRulesRuleIDFn: func(ctx context.Context, ruleID apis.InjectionRuleID, editors ...apis.RequestEditorFn) (*apis.GetInjectionRulesRuleIDResponse, error) {
			return &apis.GetInjectionRulesRuleIDResponse{
				JSON200: &apis.InjectionRule{
					RuleID:    ruleID,
					Name:      "gemini-key",
					Injection: newGeminiAPIInjection("g-1"),
					CreatedAt: now,
					UpdatedAt: now,
				},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	rule, err := c.GetInjectionRule(context.Background(), "rule-7")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rule.RuleID != "rule-7" || rule.Name != "gemini-key" {
		t.Errorf("unexpected rule: %+v", rule)
	}
	if rule.Injection.Gemini == nil || rule.Injection.Gemini.APIKey == nil || *rule.Injection.Gemini.APIKey != "g-1" {
		t.Errorf("expected Gemini injection with APIKey 'g-1', got %+v", rule.Injection)
	}
}

func newGeminiAPIInjection(apiKey string) apis.Injection {
	var inj apis.Injection
	_ = inj.FromGeminiInjection(apis.GeminiInjection{
		APIKey: strPtr(apiKey),
		Type:   apis.Gemini,
	})
	return inj
}

func TestUpdateInjectionRule(t *testing.T) {
	now := time.Now()
	var receivedBody apis.PutInjectionRulesRuleIDJSONRequestBody
	mock := &mockAPI{
		putInjectionRulesRuleIDFn: func(ctx context.Context, ruleID apis.InjectionRuleID, body apis.PutInjectionRulesRuleIDJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.PutInjectionRulesRuleIDResponse, error) {
			receivedBody = body
			return &apis.PutInjectionRulesRuleIDResponse{
				JSON200: &apis.InjectionRule{
					RuleID:    ruleID,
					Name:      "renamed",
					Injection: newOpenaiAPIInjection("sk-new"),
					CreatedAt: now,
					UpdatedAt: now,
				},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	newName := "renamed"
	rule, err := c.UpdateInjectionRule(context.Background(), "rule-9", UpdateInjectionRuleParams{
		Name: &newName,
		Injection: &InjectionSpec{
			OpenAI: &OpenAIInjection{APIKey: strPtr("sk-new")},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rule.Name != "renamed" {
		t.Errorf("expected name 'renamed', got %q", rule.Name)
	}
	if receivedBody.Name == nil || *receivedBody.Name != "renamed" {
		t.Errorf("expected body name 'renamed', got %v", receivedBody.Name)
	}
	if receivedBody.Injection == nil {
		t.Fatal("expected body injection to be set")
	}
}

func TestUpdateInjectionRuleNameOnly(t *testing.T) {
	// Updating only the name should leave body.Injection nil.
	var receivedBody apis.PutInjectionRulesRuleIDJSONRequestBody
	mock := &mockAPI{
		putInjectionRulesRuleIDFn: func(ctx context.Context, ruleID apis.InjectionRuleID, body apis.PutInjectionRulesRuleIDJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.PutInjectionRulesRuleIDResponse, error) {
			receivedBody = body
			return &apis.PutInjectionRulesRuleIDResponse{
				JSON200: &apis.InjectionRule{
					RuleID:    ruleID,
					Name:      *body.Name,
					Injection: newOpenaiAPIInjection("sk-existing"),
				},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	newName := "only-name"
	_, err := c.UpdateInjectionRule(context.Background(), "rule-10", UpdateInjectionRuleParams{
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedBody.Injection != nil {
		t.Errorf("expected body.Injection to be nil, got %+v", receivedBody.Injection)
	}
}

func TestDeleteInjectionRule(t *testing.T) {
	mock := &mockAPI{
		deleteInjectionRulesRuleIDFn: func(ctx context.Context, ruleID apis.InjectionRuleID, editors ...apis.RequestEditorFn) (*apis.DeleteInjectionRulesRuleIDResponse, error) {
			return &apis.DeleteInjectionRulesRuleIDResponse{HTTPResponse: httpResponse(204)}, nil
		},
	}
	c := newTestClient(mock)
	if err := c.DeleteInjectionRule(context.Background(), "rule-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteInjectionRuleError(t *testing.T) {
	mock := &mockAPI{
		deleteInjectionRulesRuleIDFn: func(ctx context.Context, ruleID apis.InjectionRuleID, editors ...apis.RequestEditorFn) (*apis.DeleteInjectionRulesRuleIDResponse, error) {
			return &apis.DeleteInjectionRulesRuleIDResponse{
				HTTPResponse: httpResponse(404),
				Body:         []byte(`{"message":"not found"}`),
			}, nil
		},
	}
	c := newTestClient(mock)
	err := c.DeleteInjectionRule(context.Background(), "rule-1")
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("expected status 404, got %d", apiErr.StatusCode)
	}
}

func TestSandboxInjectionSpecToAPI_ByID(t *testing.T) {
	si, err := sandboxInjectionSpecToAPI(SandboxInjectionSpec{ByID: strPtr("rule-xyz")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	disc, err := si.Discriminator()
	if err != nil {
		t.Fatalf("discriminator error: %v", err)
	}
	if disc != string(apis.ID) {
		t.Errorf("expected discriminator %q, got %q", apis.ID, disc)
	}
	v, err := si.AsInjectionByID()
	if err != nil {
		t.Fatalf("AsInjectionByID error: %v", err)
	}
	if v.ID != "rule-xyz" {
		t.Errorf("expected id 'rule-xyz', got %q", v.ID)
	}
}

func TestSandboxInjectionSpecToAPI_HTTP(t *testing.T) {
	headers := map[string]string{"X-Foo": "bar"}
	si, err := sandboxInjectionSpecToAPI(SandboxInjectionSpec{
		HTTP: &HTTPInjection{BaseURL: "https://api.example.com", Headers: &headers},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	disc, err := si.Discriminator()
	if err != nil {
		t.Fatalf("discriminator error: %v", err)
	}
	if disc != string(apis.HTTP) {
		t.Errorf("expected discriminator %q, got %q", apis.HTTP, disc)
	}
}

func TestSandboxInjectionSpecToAPI_Empty(t *testing.T) {
	_, err := sandboxInjectionSpecToAPI(SandboxInjectionSpec{})
	if err == nil {
		t.Fatal("expected error for empty SandboxInjectionSpec")
	}
}

func TestCreateParamsWithInjections(t *testing.T) {
	// Ensures CreateParams.toAPI populates body.Injections correctly.
	params := CreateParams{
		TemplateID: "tmpl-1",
		Injections: &[]SandboxInjectionSpec{
			{ByID: strPtr("rule-1")},
			{OpenAI: &OpenAIInjection{APIKey: strPtr("sk-x")}},
		},
	}
	body, err := params.toAPI()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body.Injections == nil || len(*body.Injections) != 2 {
		t.Fatalf("expected 2 injections, got %+v", body.Injections)
	}
	disc0, _ := (*body.Injections)[0].Discriminator()
	if disc0 != string(apis.ID) {
		t.Errorf("expected first injection discriminator %q, got %q", apis.ID, disc0)
	}
	disc1, _ := (*body.Injections)[1].Discriminator()
	if disc1 != string(apis.Openai) {
		t.Errorf("expected second injection discriminator %q, got %q", apis.Openai, disc1)
	}
}

func TestCreateParamsWithBadInjection(t *testing.T) {
	// Invalid injection spec → toAPI propagates the error.
	params := CreateParams{
		TemplateID: "tmpl-1",
		Injections: &[]SandboxInjectionSpec{{}},
	}
	if _, err := params.toAPI(); err == nil {
		t.Fatal("expected error for invalid injection spec")
	}
}
