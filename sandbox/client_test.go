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
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sufy-dev/sufy/auth"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/sufy-dev/sufy/sandbox/internal/apis"
	"github.com/sufy-dev/sufy/sandbox/internal/reqid"
)

// mockAPI implements apis.ClientWithResponsesInterface for tests.
// Each method field can be set per test; unset methods panic.
type mockAPI struct {
	createSandboxFn          func(ctx context.Context, body apis.CreateSandboxJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.CreateSandboxResponse, error)
	getSandboxFn             func(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.GetSandboxResponse, error)
	deleteSandboxFn          func(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.DeleteSandboxResponse, error)
	listSandboxesFn          func(ctx context.Context, params *apis.ListSandboxesParams, editors ...apis.RequestEditorFn) (*apis.ListSandboxesResponse, error)
	listSandboxesV2Fn        func(ctx context.Context, params *apis.ListSandboxesV2Params, editors ...apis.RequestEditorFn) (*apis.ListSandboxesV2Response, error)
	connectSandboxFn         func(ctx context.Context, sandboxID apis.SandboxID, body apis.ConnectSandboxJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.ConnectSandboxResponse, error)
	updateSandboxTimeoutFn   func(ctx context.Context, sandboxID apis.SandboxID, body apis.UpdateSandboxTimeoutJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.UpdateSandboxTimeoutResponse, error)
	pauseSandboxFn           func(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.PauseSandboxResponse, error)
	getSandboxMetricsFn      func(ctx context.Context, sandboxID apis.SandboxID, params *apis.GetSandboxMetricsParams, editors ...apis.RequestEditorFn) (*apis.GetSandboxMetricsResponse, error)
	getSandboxLogsFn         func(ctx context.Context, sandboxID apis.SandboxID, params *apis.GetSandboxLogsParams, editors ...apis.RequestEditorFn) (*apis.GetSandboxLogsResponse, error)
	refreshSandboxFn         func(ctx context.Context, sandboxID apis.SandboxID, body apis.RefreshSandboxJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.RefreshSandboxResponse, error)
	listTemplatesFn          func(ctx context.Context, params *apis.ListTemplatesParams, editors ...apis.RequestEditorFn) (*apis.ListTemplatesResponse, error)
	createTemplateV3Fn       func(ctx context.Context, body apis.CreateTemplateV3JSONRequestBody, editors ...apis.RequestEditorFn) (*apis.CreateTemplateV3Response, error)
	getTemplateFn            func(ctx context.Context, templateID apis.TemplateID, params *apis.GetTemplateParams, editors ...apis.RequestEditorFn) (*apis.GetTemplateResponse, error)
	deleteTemplateFn         func(ctx context.Context, templateID apis.TemplateID, editors ...apis.RequestEditorFn) (*apis.DeleteTemplateResponse, error)
	getTemplateBuildStatusFn func(ctx context.Context, templateID apis.TemplateID, buildID apis.BuildID, params *apis.GetTemplateBuildStatusParams, editors ...apis.RequestEditorFn) (*apis.GetTemplateBuildStatusResponse, error)
	getTemplateByAliasFn     func(ctx context.Context, alias string, editors ...apis.RequestEditorFn) (*apis.GetTemplateByAliasResponse, error)
	updateTemplateFn         func(ctx context.Context, templateID apis.TemplateID, body apis.UpdateTemplateJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.UpdateTemplateResponse, error)
	getTemplateBuildLogsFn   func(ctx context.Context, templateID apis.TemplateID, buildID apis.BuildID, params *apis.GetTemplateBuildLogsParams, editors ...apis.RequestEditorFn) (*apis.GetTemplateBuildLogsResponse, error)
	startTemplateBuildV2Fn   func(ctx context.Context, templateID apis.TemplateID, buildID apis.BuildID, body apis.StartTemplateBuildV2JSONRequestBody, editors ...apis.RequestEditorFn) (*apis.StartTemplateBuildV2Response, error)
	getTemplateFilesFn       func(ctx context.Context, templateID apis.TemplateID, hash string, editors ...apis.RequestEditorFn) (*apis.GetTemplateFilesResponse, error)
	assignTemplateTagsFn     func(ctx context.Context, body apis.AssignTemplateTagsJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.AssignTemplateTagsResponse, error)
	deleteTemplateTagsFn     func(ctx context.Context, body apis.DeleteTemplateTagsJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.DeleteTemplateTagsResponse, error)
	getSandboxesMetricsFn    func(ctx context.Context, params *apis.GetSandboxesMetricsParams, editors ...apis.RequestEditorFn) (*apis.GetSandboxesMetricsResponse, error)

	getInjectionRulesFn            func(ctx context.Context, editors ...apis.RequestEditorFn) (*apis.GetInjectionRulesResponse, error)
	postInjectionRulesFn           func(ctx context.Context, body apis.PostInjectionRulesJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.PostInjectionRulesResponse, error)
	deleteInjectionRulesRuleIDFn   func(ctx context.Context, ruleID apis.InjectionRuleID, editors ...apis.RequestEditorFn) (*apis.DeleteInjectionRulesRuleIDResponse, error)
	getInjectionRulesRuleIDFn      func(ctx context.Context, ruleID apis.InjectionRuleID, editors ...apis.RequestEditorFn) (*apis.GetInjectionRulesRuleIDResponse, error)
	putInjectionRulesRuleIDFn      func(ctx context.Context, ruleID apis.InjectionRuleID, body apis.PutInjectionRulesRuleIDJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.PutInjectionRulesRuleIDResponse, error)
}

func httpResponse(statusCode int) *http.Response {
	return &http.Response{StatusCode: statusCode, Header: http.Header{}}
}

func httpResponseWithReqid(statusCode int, reqidVal string) *http.Response {
	h := http.Header{}
	h.Set("X-Reqid", reqidVal)
	return &http.Response{StatusCode: statusCode, Header: h}
}

// --- Sandbox operations ---

func (m *mockAPI) CreateSandboxWithResponse(ctx context.Context, body apis.CreateSandboxJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.CreateSandboxResponse, error) {
	return m.createSandboxFn(ctx, body, editors...)
}

func (m *mockAPI) CreateSandboxWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, editors ...apis.RequestEditorFn) (*apis.CreateSandboxResponse, error) {
	panic("not implemented")
}

func (m *mockAPI) GetSandboxWithResponse(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.GetSandboxResponse, error) {
	return m.getSandboxFn(ctx, sandboxID, editors...)
}

func (m *mockAPI) DeleteSandboxWithResponse(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.DeleteSandboxResponse, error) {
	return m.deleteSandboxFn(ctx, sandboxID, editors...)
}

func (m *mockAPI) ListSandboxesWithResponse(ctx context.Context, params *apis.ListSandboxesParams, editors ...apis.RequestEditorFn) (*apis.ListSandboxesResponse, error) {
	return m.listSandboxesFn(ctx, params, editors...)
}

func (m *mockAPI) ListSandboxesV2WithResponse(ctx context.Context, params *apis.ListSandboxesV2Params, editors ...apis.RequestEditorFn) (*apis.ListSandboxesV2Response, error) {
	return m.listSandboxesV2Fn(ctx, params, editors...)
}

func (m *mockAPI) ConnectSandboxWithResponse(ctx context.Context, sandboxID apis.SandboxID, body apis.ConnectSandboxJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.ConnectSandboxResponse, error) {
	return m.connectSandboxFn(ctx, sandboxID, body, editors...)
}

func (m *mockAPI) ConnectSandboxWithBodyWithResponse(ctx context.Context, sandboxID apis.SandboxID, contentType string, body io.Reader, editors ...apis.RequestEditorFn) (*apis.ConnectSandboxResponse, error) {
	panic("not implemented")
}

func (m *mockAPI) UpdateSandboxTimeoutWithResponse(ctx context.Context, sandboxID apis.SandboxID, body apis.UpdateSandboxTimeoutJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.UpdateSandboxTimeoutResponse, error) {
	return m.updateSandboxTimeoutFn(ctx, sandboxID, body, editors...)
}

func (m *mockAPI) UpdateSandboxTimeoutWithBodyWithResponse(ctx context.Context, sandboxID apis.SandboxID, contentType string, body io.Reader, editors ...apis.RequestEditorFn) (*apis.UpdateSandboxTimeoutResponse, error) {
	panic("not implemented")
}

func (m *mockAPI) PauseSandboxWithResponse(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.PauseSandboxResponse, error) {
	return m.pauseSandboxFn(ctx, sandboxID, editors...)
}

func (m *mockAPI) ResumeSandboxWithResponse(ctx context.Context, sandboxID apis.SandboxID, body apis.ResumeSandboxJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.ResumeSandboxResponse, error) {
	panic("not implemented")
}

func (m *mockAPI) ResumeSandboxWithBodyWithResponse(ctx context.Context, sandboxID apis.SandboxID, contentType string, body io.Reader, editors ...apis.RequestEditorFn) (*apis.ResumeSandboxResponse, error) {
	panic("not implemented")
}

func (m *mockAPI) GetSandboxMetricsWithResponse(ctx context.Context, sandboxID apis.SandboxID, params *apis.GetSandboxMetricsParams, editors ...apis.RequestEditorFn) (*apis.GetSandboxMetricsResponse, error) {
	return m.getSandboxMetricsFn(ctx, sandboxID, params, editors...)
}

func (m *mockAPI) GetSandboxLogsWithResponse(ctx context.Context, sandboxID apis.SandboxID, params *apis.GetSandboxLogsParams, editors ...apis.RequestEditorFn) (*apis.GetSandboxLogsResponse, error) {
	return m.getSandboxLogsFn(ctx, sandboxID, params, editors...)
}

func (m *mockAPI) RefreshSandboxWithResponse(ctx context.Context, sandboxID apis.SandboxID, body apis.RefreshSandboxJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.RefreshSandboxResponse, error) {
	return m.refreshSandboxFn(ctx, sandboxID, body, editors...)
}

func (m *mockAPI) RefreshSandboxWithBodyWithResponse(ctx context.Context, sandboxID apis.SandboxID, contentType string, body io.Reader, editors ...apis.RequestEditorFn) (*apis.RefreshSandboxResponse, error) {
	panic("not implemented")
}

func (m *mockAPI) GetSandboxesMetricsWithResponse(ctx context.Context, params *apis.GetSandboxesMetricsParams, editors ...apis.RequestEditorFn) (*apis.GetSandboxesMetricsResponse, error) {
	return m.getSandboxesMetricsFn(ctx, params, editors...)
}

// --- Template operations ---

func (m *mockAPI) ListDefaultTemplatesWithResponse(ctx context.Context, editors ...apis.RequestEditorFn) (*apis.ListDefaultTemplatesResponse, error) {
	panic("not implemented")
}

func (m *mockAPI) ListTemplatesWithResponse(ctx context.Context, params *apis.ListTemplatesParams, editors ...apis.RequestEditorFn) (*apis.ListTemplatesResponse, error) {
	return m.listTemplatesFn(ctx, params, editors...)
}

func (m *mockAPI) CreateTemplateV3WithResponse(ctx context.Context, body apis.CreateTemplateV3JSONRequestBody, editors ...apis.RequestEditorFn) (*apis.CreateTemplateV3Response, error) {
	return m.createTemplateV3Fn(ctx, body, editors...)
}

func (m *mockAPI) CreateTemplateV3WithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, editors ...apis.RequestEditorFn) (*apis.CreateTemplateV3Response, error) {
	panic("not implemented")
}

func (m *mockAPI) GetTemplateWithResponse(ctx context.Context, templateID apis.TemplateID, params *apis.GetTemplateParams, editors ...apis.RequestEditorFn) (*apis.GetTemplateResponse, error) {
	return m.getTemplateFn(ctx, templateID, params, editors...)
}

func (m *mockAPI) DeleteTemplateWithResponse(ctx context.Context, templateID apis.TemplateID, editors ...apis.RequestEditorFn) (*apis.DeleteTemplateResponse, error) {
	return m.deleteTemplateFn(ctx, templateID, editors...)
}

func (m *mockAPI) UpdateTemplateWithResponse(ctx context.Context, templateID apis.TemplateID, body apis.UpdateTemplateJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.UpdateTemplateResponse, error) {
	return m.updateTemplateFn(ctx, templateID, body, editors...)
}

func (m *mockAPI) UpdateTemplateWithBodyWithResponse(ctx context.Context, templateID apis.TemplateID, contentType string, body io.Reader, editors ...apis.RequestEditorFn) (*apis.UpdateTemplateResponse, error) {
	panic("not implemented")
}

func (m *mockAPI) RebuildTemplateWithResponse(ctx context.Context, templateID apis.TemplateID, body apis.RebuildTemplateJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.RebuildTemplateResponse, error) {
	panic("not implemented")
}

func (m *mockAPI) RebuildTemplateWithBodyWithResponse(ctx context.Context, templateID apis.TemplateID, contentType string, body io.Reader, editors ...apis.RequestEditorFn) (*apis.RebuildTemplateResponse, error) {
	panic("not implemented")
}

func (m *mockAPI) CreateTemplateWithResponse(ctx context.Context, body apis.CreateTemplateJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.CreateTemplateResponse, error) {
	panic("not implemented")
}

func (m *mockAPI) CreateTemplateWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, editors ...apis.RequestEditorFn) (*apis.CreateTemplateResponse, error) {
	panic("not implemented")
}

func (m *mockAPI) CreateTemplateV2WithResponse(ctx context.Context, body apis.CreateTemplateV2JSONRequestBody, editors ...apis.RequestEditorFn) (*apis.CreateTemplateV2Response, error) {
	panic("not implemented")
}

func (m *mockAPI) CreateTemplateV2WithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, editors ...apis.RequestEditorFn) (*apis.CreateTemplateV2Response, error) {
	panic("not implemented")
}

func (m *mockAPI) GetTemplateBuildStatusWithResponse(ctx context.Context, templateID apis.TemplateID, buildID apis.BuildID, params *apis.GetTemplateBuildStatusParams, editors ...apis.RequestEditorFn) (*apis.GetTemplateBuildStatusResponse, error) {
	return m.getTemplateBuildStatusFn(ctx, templateID, buildID, params, editors...)
}

func (m *mockAPI) GetTemplateBuildLogsWithResponse(ctx context.Context, templateID apis.TemplateID, buildID apis.BuildID, params *apis.GetTemplateBuildLogsParams, editors ...apis.RequestEditorFn) (*apis.GetTemplateBuildLogsResponse, error) {
	return m.getTemplateBuildLogsFn(ctx, templateID, buildID, params, editors...)
}

func (m *mockAPI) StartTemplateBuildWithResponse(ctx context.Context, templateID apis.TemplateID, buildID apis.BuildID, editors ...apis.RequestEditorFn) (*apis.StartTemplateBuildResponse, error) {
	panic("not implemented")
}

func (m *mockAPI) StartTemplateBuildV2WithResponse(ctx context.Context, templateID apis.TemplateID, buildID apis.BuildID, body apis.StartTemplateBuildV2JSONRequestBody, editors ...apis.RequestEditorFn) (*apis.StartTemplateBuildV2Response, error) {
	return m.startTemplateBuildV2Fn(ctx, templateID, buildID, body, editors...)
}

func (m *mockAPI) StartTemplateBuildV2WithBodyWithResponse(ctx context.Context, templateID apis.TemplateID, buildID apis.BuildID, contentType string, body io.Reader, editors ...apis.RequestEditorFn) (*apis.StartTemplateBuildV2Response, error) {
	panic("not implemented")
}

func (m *mockAPI) GetTemplateFilesWithResponse(ctx context.Context, templateID apis.TemplateID, hash string, editors ...apis.RequestEditorFn) (*apis.GetTemplateFilesResponse, error) {
	return m.getTemplateFilesFn(ctx, templateID, hash, editors...)
}

func (m *mockAPI) GetTemplateByAliasWithResponse(ctx context.Context, alias string, editors ...apis.RequestEditorFn) (*apis.GetTemplateByAliasResponse, error) {
	return m.getTemplateByAliasFn(ctx, alias, editors...)
}

func (m *mockAPI) AssignTemplateTagsWithResponse(ctx context.Context, body apis.AssignTemplateTagsJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.AssignTemplateTagsResponse, error) {
	return m.assignTemplateTagsFn(ctx, body, editors...)
}

func (m *mockAPI) AssignTemplateTagsWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, editors ...apis.RequestEditorFn) (*apis.AssignTemplateTagsResponse, error) {
	panic("not implemented")
}

func (m *mockAPI) DeleteTemplateTagsWithResponse(ctx context.Context, body apis.DeleteTemplateTagsJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.DeleteTemplateTagsResponse, error) {
	return m.deleteTemplateTagsFn(ctx, body, editors...)
}

func (m *mockAPI) DeleteTemplateTagsWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, editors ...apis.RequestEditorFn) (*apis.DeleteTemplateTagsResponse, error) {
	panic("not implemented")
}

// --- Injection rule operations ---

func (m *mockAPI) GetInjectionRulesWithResponse(ctx context.Context, editors ...apis.RequestEditorFn) (*apis.GetInjectionRulesResponse, error) {
	return m.getInjectionRulesFn(ctx, editors...)
}

func (m *mockAPI) PostInjectionRulesWithResponse(ctx context.Context, body apis.PostInjectionRulesJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.PostInjectionRulesResponse, error) {
	return m.postInjectionRulesFn(ctx, body, editors...)
}

func (m *mockAPI) PostInjectionRulesWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, editors ...apis.RequestEditorFn) (*apis.PostInjectionRulesResponse, error) {
	panic("not implemented")
}

func (m *mockAPI) DeleteInjectionRulesRuleIDWithResponse(ctx context.Context, ruleID apis.InjectionRuleID, editors ...apis.RequestEditorFn) (*apis.DeleteInjectionRulesRuleIDResponse, error) {
	return m.deleteInjectionRulesRuleIDFn(ctx, ruleID, editors...)
}

func (m *mockAPI) GetInjectionRulesRuleIDWithResponse(ctx context.Context, ruleID apis.InjectionRuleID, editors ...apis.RequestEditorFn) (*apis.GetInjectionRulesRuleIDResponse, error) {
	return m.getInjectionRulesRuleIDFn(ctx, ruleID, editors...)
}

func (m *mockAPI) PutInjectionRulesRuleIDWithResponse(ctx context.Context, ruleID apis.InjectionRuleID, body apis.PutInjectionRulesRuleIDJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.PutInjectionRulesRuleIDResponse, error) {
	return m.putInjectionRulesRuleIDFn(ctx, ruleID, body, editors...)
}

func (m *mockAPI) PutInjectionRulesRuleIDWithBodyWithResponse(ctx context.Context, ruleID apis.InjectionRuleID, contentType string, body io.Reader, editors ...apis.RequestEditorFn) (*apis.PutInjectionRulesRuleIDResponse, error) {
	panic("not implemented")
}

// ============================================================
// Test cases
// ============================================================

func newTestClient(api apis.ClientWithResponsesInterface) *Client {
	return &Client{config: &Config{APIKey: "test-key"}, api: api}
}

func newTestSandbox(c *Client, id string) *Sandbox {
	return &Sandbox{sandboxID: id, client: c}
}

// --- Client tests ---

func TestNewClient(t *testing.T) {
	c := New(&Config{APIKey: "test-key", BaseURL: "https://sandbox.example.com"})
	if c == nil {
		t.Fatal("expected non-nil client")
	}
	if c.api == nil {
		t.Error("expected non-nil API client")
	}
}

func TestNewClientDefaultBaseURL(t *testing.T) {
	c := New(&Config{APIKey: "test-key", BaseURL: "https://sandbox.example.com"})
	if c.config.APIKey != "test-key" {
		t.Errorf("expected API key 'test-key', got %q", c.config.APIKey)
	}
}

func TestNewClientCustomBaseURL(t *testing.T) {
	c := New(&Config{APIKey: "test-key", BaseURL: "https://custom.example.com"})
	if c.config.BaseURL != "https://custom.example.com" {
		t.Errorf("expected custom base URL, got %q", c.config.BaseURL)
	}
}

func TestAPIKeyEditor(t *testing.T) {
	editor := apiKeyEditor("test-key")
	req, _ := http.NewRequest("GET", "https://example.com", nil)
	if err := editor(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req.Header.Get("X-API-Key"); got != "test-key" {
		t.Errorf("expected X-API-Key 'test-key', got %q", got)
	}
}

func TestAPIKeyEditorSkipsWhenAuthorizationSet(t *testing.T) {
	editor := apiKeyEditor("test-key")
	req, _ := http.NewRequest("GET", "https://example.com", nil)
	req.Header.Set("Authorization", "Sufy ak:sig")
	if err := editor(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req.Header.Get("X-API-Key"); got != "" {
		t.Errorf("expected X-API-Key to be empty when Authorization is set, got %q", got)
	}
}

func TestCredentialsEditor(t *testing.T) {
	cred := auth.New("test-ak", "test-sk")
	editor := credentialsEditor(cred)
	req, _ := http.NewRequest("GET", "https://example.com/injection-rules", nil)
	req.Header.Set("Content-Type", "application/json")
	if err := editor(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
		t.Fatal("expected Authorization header to be set")
	}
	if !strings.HasPrefix(authHeader, "Sufy test-ak:") {
		t.Errorf("expected Authorization to start with 'Sufy test-ak:', got %q", authHeader)
	}
}

func TestReqidEditor(t *testing.T) {
	editor := reqidEditor()

	// When reqid is present, inject it.
	ctx := reqid.WithReqid(context.Background(), "test-reqid-123")
	req, _ := http.NewRequest("GET", "https://example.com", nil)
	if err := editor(ctx, req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req.Header.Get("X-Reqid"); got != "test-reqid-123" {
		t.Errorf("expected X-Reqid 'test-reqid-123', got %q", got)
	}

	// When reqid is absent, do not set the header.
	req2, _ := http.NewRequest("GET", "https://example.com", nil)
	if err := editor(context.Background(), req2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req2.Header.Get("X-Reqid"); got != "" {
		t.Errorf("expected empty X-Reqid, got %q", got)
	}
}

// --- Sandbox.Create ---

func TestCreate(t *testing.T) {
	token := "create-token"
	mock := &mockAPI{
		createSandboxFn: func(ctx context.Context, body apis.CreateSandboxJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.CreateSandboxResponse, error) {
			return &apis.CreateSandboxResponse{
				JSON201:      &apis.Sandbox{SandboxID: "sb-123", TemplateID: "tmpl-1", EnvdAccessToken: &token},
				HTTPResponse: httpResponse(201),
			}, nil
		},
	}
	c := newTestClient(mock)
	sb, err := c.Create(context.Background(), CreateParams{TemplateID: "tmpl-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sb.ID() != "sb-123" {
		t.Errorf("expected sandbox ID 'sb-123', got %q", sb.ID())
	}
	if sb.TemplateID() != "tmpl-1" {
		t.Errorf("expected template ID 'tmpl-1', got %q", sb.TemplateID())
	}
}

func TestCreateWithoutToken(t *testing.T) {
	// Create API does not echo the token; it should be fetched via GetSandbox.
	token := "fallback-token"
	mock := &mockAPI{
		createSandboxFn: func(ctx context.Context, body apis.CreateSandboxJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.CreateSandboxResponse, error) {
			return &apis.CreateSandboxResponse{
				JSON201:      &apis.Sandbox{SandboxID: "sb-123", TemplateID: "tmpl-1"},
				HTTPResponse: httpResponse(201),
			}, nil
		},
		getSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.GetSandboxResponse, error) {
			return &apis.GetSandboxResponse{
				JSON200:      &apis.SandboxDetail{SandboxID: sandboxID, EnvdAccessToken: &token},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	sb, err := c.Create(context.Background(), CreateParams{TemplateID: "tmpl-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sb.ID() != "sb-123" {
		t.Errorf("expected sandbox ID 'sb-123', got %q", sb.ID())
	}
	sb.envdTokenMu.RLock()
	tok := sb.envdAccessToken
	sb.envdTokenMu.RUnlock()
	if tok == nil || *tok != "fallback-token" {
		t.Errorf("expected envdAccessToken 'fallback-token', got %v", tok)
	}
}

func TestCreateRefreshTokenError(t *testing.T) {
	// Create API does not echo the token and GetSandbox also fails → Create returns an error.
	mock := &mockAPI{
		createSandboxFn: func(ctx context.Context, body apis.CreateSandboxJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.CreateSandboxResponse, error) {
			return &apis.CreateSandboxResponse{
				JSON201:      &apis.Sandbox{SandboxID: "sb-123", TemplateID: "tmpl-1"},
				HTTPResponse: httpResponse(201),
			}, nil
		},
		getSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.GetSandboxResponse, error) {
			return nil, fmt.Errorf("network error")
		},
	}
	c := newTestClient(mock)
	_, err := c.Create(context.Background(), CreateParams{TemplateID: "tmpl-1"})
	if err == nil {
		t.Fatal("expected error when GetSandbox fails")
	}
}

func TestCreateError(t *testing.T) {
	mock := &mockAPI{
		createSandboxFn: func(ctx context.Context, body apis.CreateSandboxJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.CreateSandboxResponse, error) {
			return &apis.CreateSandboxResponse{
				HTTPResponse: httpResponse(400),
				Body:         []byte(`{"message":"bad request"}`),
			}, nil
		},
	}
	c := newTestClient(mock)
	_, err := c.Create(context.Background(), CreateParams{TemplateID: "tmpl-1"})
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 400 {
		t.Errorf("expected status 400, got %d", apiErr.StatusCode)
	}
}

// --- Sandbox.Connect ---

func TestConnectWithToken(t *testing.T) {
	// When Connect echoes the token, GetSandbox should not be called.
	token := "connect-token"
	var getSandboxCalled atomic.Bool
	mock := &mockAPI{
		connectSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, body apis.ConnectSandboxJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.ConnectSandboxResponse, error) {
			return &apis.ConnectSandboxResponse{
				JSON200:      &apis.Sandbox{SandboxID: sandboxID, TemplateID: "tmpl-1", EnvdAccessToken: &token},
				HTTPResponse: httpResponse(200),
			}, nil
		},
		getSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.GetSandboxResponse, error) {
			getSandboxCalled.Store(true)
			return &apis.GetSandboxResponse{
				JSON200:      &apis.SandboxDetail{SandboxID: sandboxID},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	sb, err := c.Connect(context.Background(), "sb-123", ConnectParams{Timeout: 60})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sb.ID() != "sb-123" {
		t.Errorf("expected sandbox ID 'sb-123', got %q", sb.ID())
	}
	if getSandboxCalled.Load() {
		t.Error("GetSandbox should not be called when Connect returns a token")
	}
	sb.envdTokenMu.RLock()
	tok := sb.envdAccessToken
	sb.envdTokenMu.RUnlock()
	if tok == nil || *tok != "connect-token" {
		t.Errorf("expected envdAccessToken 'connect-token', got %v", tok)
	}
}

func TestConnectWithoutToken(t *testing.T) {
	// Connect does not echo the token; it is fetched via GetSandbox.
	token := "fallback-token"
	mock := &mockAPI{
		connectSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, body apis.ConnectSandboxJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.ConnectSandboxResponse, error) {
			return &apis.ConnectSandboxResponse{
				JSON200:      &apis.Sandbox{SandboxID: sandboxID, TemplateID: "tmpl-1"},
				HTTPResponse: httpResponse(200),
			}, nil
		},
		getSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.GetSandboxResponse, error) {
			return &apis.GetSandboxResponse{
				JSON200:      &apis.SandboxDetail{SandboxID: sandboxID, EnvdAccessToken: &token},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	sb, err := c.Connect(context.Background(), "sb-123", ConnectParams{Timeout: 60})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sb.ID() != "sb-123" {
		t.Errorf("expected sandbox ID 'sb-123', got %q", sb.ID())
	}
	sb.envdTokenMu.RLock()
	tok := sb.envdAccessToken
	sb.envdTokenMu.RUnlock()
	if tok == nil || *tok != "fallback-token" {
		t.Errorf("expected envdAccessToken 'fallback-token', got %v", tok)
	}
}

func TestConnectRefreshTokenError(t *testing.T) {
	// Connect does not echo the token and GetSandbox fails → Connect returns an error.
	mock := &mockAPI{
		connectSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, body apis.ConnectSandboxJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.ConnectSandboxResponse, error) {
			return &apis.ConnectSandboxResponse{
				JSON200:      &apis.Sandbox{SandboxID: sandboxID, TemplateID: "tmpl-1"},
				HTTPResponse: httpResponse(200),
			}, nil
		},
		getSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.GetSandboxResponse, error) {
			return nil, errors.New("network error")
		},
	}
	c := newTestClient(mock)
	_, err := c.Connect(context.Background(), "sb-123", ConnectParams{Timeout: 60})
	if err == nil {
		t.Fatal("expected error when GetSandbox fails")
	}
}

func TestConnect201Response(t *testing.T) {
	// Connect returns 201 (resuming a paused sandbox) with a token.
	token := "resumed-token"
	mock := &mockAPI{
		connectSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, body apis.ConnectSandboxJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.ConnectSandboxResponse, error) {
			return &apis.ConnectSandboxResponse{
				JSON201:      &apis.Sandbox{SandboxID: sandboxID, TemplateID: "tmpl-1", EnvdAccessToken: &token},
				HTTPResponse: httpResponse(201),
			}, nil
		},
	}
	c := newTestClient(mock)
	sb, err := c.Connect(context.Background(), "sb-123", ConnectParams{Timeout: 60})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sb.ID() != "sb-123" {
		t.Errorf("expected sandbox ID 'sb-123', got %q", sb.ID())
	}
}

// --- Sandbox.List ---

func TestList(t *testing.T) {
	mock := &mockAPI{
		listSandboxesV2Fn: func(ctx context.Context, params *apis.ListSandboxesV2Params, editors ...apis.RequestEditorFn) (*apis.ListSandboxesV2Response, error) {
			list := []apis.ListedSandbox{
				{SandboxID: "sb-1"},
				{SandboxID: "sb-2"},
			}
			return &apis.ListSandboxesV2Response{
				JSON200:      &list,
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	sandboxes, err := c.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sandboxes) != 2 {
		t.Errorf("expected 2 sandboxes, got %d", len(sandboxes))
	}
}

// --- Sandbox.Kill ---

func TestKill(t *testing.T) {
	mock := &mockAPI{
		deleteSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.DeleteSandboxResponse, error) {
			return &apis.DeleteSandboxResponse{HTTPResponse: httpResponse(204)}, nil
		},
	}
	c := newTestClient(mock)
	sb := newTestSandbox(c, "sb-123")
	if err := sb.Kill(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Sandbox.SetTimeout ---

func TestSetTimeout(t *testing.T) {
	var gotTimeout int32
	mock := &mockAPI{
		updateSandboxTimeoutFn: func(ctx context.Context, sandboxID apis.SandboxID, body apis.UpdateSandboxTimeoutJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.UpdateSandboxTimeoutResponse, error) {
			gotTimeout = body.Timeout
			return &apis.UpdateSandboxTimeoutResponse{HTTPResponse: httpResponse(204)}, nil
		},
	}
	c := newTestClient(mock)
	sb := newTestSandbox(c, "sb-123")
	if err := sb.SetTimeout(context.Background(), 2*time.Minute); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotTimeout != 120 {
		t.Errorf("expected timeout 120 seconds, got %d", gotTimeout)
	}
}

// --- Sandbox.GetInfo ---

func TestGetInfo(t *testing.T) {
	mock := &mockAPI{
		getSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.GetSandboxResponse, error) {
			return &apis.GetSandboxResponse{
				JSON200:      &apis.SandboxDetail{SandboxID: sandboxID, State: apis.Running},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	sb := newTestSandbox(c, "sb-123")
	info, err := sb.GetInfo(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.State != StateRunning {
		t.Errorf("expected state 'running', got %q", info.State)
	}
}

// --- Sandbox.IsRunning ---

// newEnvdHealthServer starts a mock envd server whose /health endpoint returns the given status code.
func newEnvdHealthServer(statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(statusCode)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
}

func TestIsRunning(t *testing.T) {
	ts := newEnvdHealthServer(http.StatusNoContent)
	defer ts.Close()

	// Build a sandbox whose envdURL() targets the test server. Since envdURL
	// depends on GetHost which builds an arbitrary hostname, redirect all
	// requests to the test server via a custom transport.
	c := &Client{config: &Config{APIKey: "test-key", HTTPClient: ts.Client()}}
	sb := &Sandbox{sandboxID: "sb-123", client: c}
	c.config.HTTPClient = redirectClient(ts.URL)

	running, err := sb.IsRunning(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !running {
		t.Error("expected sandbox to be running")
	}
}

func TestIsRunningNotReachable(t *testing.T) {
	ts := newEnvdHealthServer(http.StatusBadGateway)
	defer ts.Close()

	c := &Client{config: &Config{APIKey: "test-key", HTTPClient: redirectClient(ts.URL)}}
	sb := &Sandbox{sandboxID: "sb-123", client: c}

	running, err := sb.IsRunning(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if running {
		t.Error("expected sandbox to not be running")
	}
}

// redirectClient returns an HTTP client that redirects all requests to the given target URL.
func redirectClient(target string) *http.Client {
	return &http.Client{
		Transport: &redirectTransport{target: target},
	}
}

type redirectTransport struct {
	target string
}

func (t *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Preserve original path and query; replace scheme+host with the test server.
	newURL := t.target + req.URL.Path
	if req.URL.RawQuery != "" {
		newURL += "?" + req.URL.RawQuery
	}
	newReq, err := http.NewRequestWithContext(req.Context(), req.Method, newURL, req.Body)
	if err != nil {
		return nil, err
	}
	newReq.Header = req.Header
	return http.DefaultTransport.RoundTrip(newReq)
}

// --- Sandbox.Pause ---

func TestPause(t *testing.T) {
	mock := &mockAPI{
		pauseSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.PauseSandboxResponse, error) {
			return &apis.PauseSandboxResponse{HTTPResponse: httpResponse(204)}, nil
		},
	}
	c := newTestClient(mock)
	sb := newTestSandbox(c, "sb-123")
	if err := sb.Pause(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Sandbox.WaitForReady ---

func TestWaitForReadyImmediate(t *testing.T) {
	mock := &mockAPI{
		getSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.GetSandboxResponse, error) {
			return &apis.GetSandboxResponse{
				JSON200:      &apis.SandboxDetail{SandboxID: sandboxID, State: apis.Running},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	sb := newTestSandbox(c, "sb-123")
	info, err := sb.WaitForReady(context.Background(), WithPollInterval(100*time.Millisecond))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.SandboxID != "sb-123" {
		t.Errorf("expected sandbox ID 'sb-123', got %q", info.SandboxID)
	}
}

func TestWaitForReadyTimeout(t *testing.T) {
	mock := &mockAPI{
		getSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.GetSandboxResponse, error) {
			return &apis.GetSandboxResponse{
				JSON200:      &apis.SandboxDetail{SandboxID: sandboxID, State: apis.Paused},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	sb := newTestSandbox(c, "sb-123")
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err := sb.WaitForReady(ctx, WithPollInterval(50*time.Millisecond))
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

// --- Template tests ---

func TestListTemplates(t *testing.T) {
	mock := &mockAPI{
		listTemplatesFn: func(ctx context.Context, params *apis.ListTemplatesParams, editors ...apis.RequestEditorFn) (*apis.ListTemplatesResponse, error) {
			list := []apis.Template{
				{TemplateID: "tmpl-1"},
				{TemplateID: "tmpl-2"},
			}
			return &apis.ListTemplatesResponse{
				JSON200:      &list,
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	templates, err := c.ListTemplates(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(templates) != 2 {
		t.Errorf("expected 2 templates, got %d", len(templates))
	}
}

func TestDeleteTemplate(t *testing.T) {
	mock := &mockAPI{
		deleteTemplateFn: func(ctx context.Context, templateID apis.TemplateID, editors ...apis.RequestEditorFn) (*apis.DeleteTemplateResponse, error) {
			return &apis.DeleteTemplateResponse{HTTPResponse: httpResponse(204)}, nil
		},
	}
	c := newTestClient(mock)
	if err := c.DeleteTemplate(context.Background(), "tmpl-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetTemplateByAlias(t *testing.T) {
	mock := &mockAPI{
		getTemplateByAliasFn: func(ctx context.Context, alias string, editors ...apis.RequestEditorFn) (*apis.GetTemplateByAliasResponse, error) {
			return &apis.GetTemplateByAliasResponse{
				JSON200:      &apis.TemplateAliasResponse{TemplateID: "tmpl-1", Public: true},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	tmpl, err := c.GetTemplateByAlias(context.Background(), "my-alias")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tmpl.TemplateID != "tmpl-1" {
		t.Errorf("expected template ID 'tmpl-1', got %q", tmpl.TemplateID)
	}
}

// --- WaitForBuild ---

func TestWaitForBuildReady(t *testing.T) {
	mock := &mockAPI{
		getTemplateBuildStatusFn: func(ctx context.Context, templateID apis.TemplateID, buildID apis.BuildID, params *apis.GetTemplateBuildStatusParams, editors ...apis.RequestEditorFn) (*apis.GetTemplateBuildStatusResponse, error) {
			return &apis.GetTemplateBuildStatusResponse{
				JSON200:      &apis.TemplateBuildInfo{TemplateID: templateID, BuildID: buildID, Status: apis.TemplateBuildStatusReady},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	info, err := c.WaitForBuild(context.Background(), "tmpl-1", "build-1", WithPollInterval(100*time.Millisecond))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Status != BuildStatusReady {
		t.Errorf("expected status 'ready', got %q", info.Status)
	}
}

func TestWaitForBuildError(t *testing.T) {
	mock := &mockAPI{
		getTemplateBuildStatusFn: func(ctx context.Context, templateID apis.TemplateID, buildID apis.BuildID, params *apis.GetTemplateBuildStatusParams, editors ...apis.RequestEditorFn) (*apis.GetTemplateBuildStatusResponse, error) {
			return &apis.GetTemplateBuildStatusResponse{
				JSON200:      &apis.TemplateBuildInfo{TemplateID: templateID, BuildID: buildID, Status: apis.TemplateBuildStatusError},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	info, err := c.WaitForBuild(context.Background(), "tmpl-1", "build-1", WithPollInterval(100*time.Millisecond))
	if err == nil {
		t.Fatal("expected error for failed build")
	}
	if info == nil {
		t.Fatal("expected non-nil build info even on error")
	}
	if info.Status != BuildStatusError {
		t.Errorf("expected status 'error', got %q", info.Status)
	}
}

// --- APIError ---

func TestAPIErrorMessage(t *testing.T) {
	// Constructed directly (no factory); Message is empty and falls back to body text.
	err := &APIError{StatusCode: 404, Body: []byte(`{"message":"not found"}`)}
	msg := err.Error()
	if msg != `api error: status 404, body: {"message":"not found"}` {
		t.Errorf("unexpected error message: %s", msg)
	}

	// Using the newAPIError factory, which parses the JSON body's message field.
	err2 := newAPIError(httpResponse(404), []byte(`{"code":"not_found","message":"resource not found"}`))
	if err2.Code != "not_found" {
		t.Errorf("expected code 'not_found', got %q", err2.Code)
	}
	if err2.Message != "resource not found" {
		t.Errorf("expected message 'resource not found', got %q", err2.Message)
	}
	msg2 := err2.Error()
	if msg2 != "api error: status 404: resource not found" {
		t.Errorf("unexpected error message: %s", msg2)
	}

	// Non-JSON body falls back to the body-text format.
	err3 := newAPIError(httpResponse(500), []byte("internal error"))
	if err3.Code != "" || err3.Message != "" {
		t.Errorf("expected empty code/message for non-JSON body, got code=%q message=%q", err3.Code, err3.Message)
	}
	msg3 := err3.Error()
	if msg3 != "api error: status 500, body: internal error" {
		t.Errorf("unexpected error message: %s", msg3)
	}

	// Empty body.
	err4 := newAPIError(httpResponse(502), nil)
	if err4.Code != "" || err4.Message != "" {
		t.Errorf("expected empty code/message for nil body, got code=%q message=%q", err4.Code, err4.Message)
	}
	msg4 := err4.Error()
	if msg4 != "api error: status 502" {
		t.Errorf("unexpected error message: %s", msg4)
	}

	// Response carries the X-Reqid header.
	respWithReqid := httpResponseWithReqid(400, "req-abc-123")
	err5 := newAPIError(respWithReqid, []byte(`{"code":"bad_request","message":"invalid param"}`))
	if err5.Reqid != "req-abc-123" {
		t.Errorf("expected reqid 'req-abc-123', got %q", err5.Reqid)
	}
	msg5 := err5.Error()
	if msg5 != "api error: status 400, reqid: req-abc-123: invalid param" {
		t.Errorf("unexpected error message: %s", msg5)
	}
}

// --- Sandbox.GetMetrics ---

func TestGetMetrics(t *testing.T) {
	mock := &mockAPI{
		getSandboxMetricsFn: func(ctx context.Context, sandboxID apis.SandboxID, params *apis.GetSandboxMetricsParams, editors ...apis.RequestEditorFn) (*apis.GetSandboxMetricsResponse, error) {
			metrics := []apis.SandboxMetric{{CPUUsedPct: 50.0, CPUCount: 2}}
			return &apis.GetSandboxMetricsResponse{
				JSON200:      &metrics,
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	sb := newTestSandbox(c, "sb-123")
	metrics, err := sb.GetMetrics(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(metrics))
	}
	if metrics[0].CPUUsedPct != 50.0 {
		t.Errorf("expected CPU 50%%, got %f", metrics[0].CPUUsedPct)
	}
}

// --- Sandbox.GetLogs ---

func TestGetLogs(t *testing.T) {
	mock := &mockAPI{
		getSandboxLogsFn: func(ctx context.Context, sandboxID apis.SandboxID, params *apis.GetSandboxLogsParams, editors ...apis.RequestEditorFn) (*apis.GetSandboxLogsResponse, error) {
			return &apis.GetSandboxLogsResponse{
				JSON200:      &apis.SandboxLogs{Logs: []apis.SandboxLog{{Line: "hello"}}},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	sb := newTestSandbox(c, "sb-123")
	logs, err := sb.GetLogs(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs.Logs) != 1 || logs.Logs[0].Line != "hello" {
		t.Errorf("unexpected logs: %v", logs)
	}
}

// --- Sandbox.Refresh ---

func TestRefresh(t *testing.T) {
	mock := &mockAPI{
		refreshSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, body apis.RefreshSandboxJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.RefreshSandboxResponse, error) {
			return &apis.RefreshSandboxResponse{HTTPResponse: httpResponse(204)}, nil
		},
	}
	c := newTestClient(mock)
	sb := newTestSandbox(c, "sb-123")
	if err := sb.Refresh(context.Background(), RefreshParams{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Client.CreateAndWait ---

func TestCreateAndWait(t *testing.T) {
	mock := &mockAPI{
		createSandboxFn: func(ctx context.Context, body apis.CreateSandboxJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.CreateSandboxResponse, error) {
			return &apis.CreateSandboxResponse{
				JSON201:      &apis.Sandbox{SandboxID: "sb-new", TemplateID: "tmpl-1"},
				HTTPResponse: httpResponse(201),
			}, nil
		},
		getSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.GetSandboxResponse, error) {
			return &apis.GetSandboxResponse{
				JSON200:      &apis.SandboxDetail{SandboxID: sandboxID, State: apis.Running},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	sb, info, err := c.CreateAndWait(context.Background(), CreateParams{TemplateID: "tmpl-1"}, WithPollInterval(100*time.Millisecond))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sb.ID() != "sb-new" {
		t.Errorf("expected sandbox ID 'sb-new', got %q", sb.ID())
	}
	if info.State != StateRunning {
		t.Errorf("expected state 'running', got %q", info.State)
	}
}

func TestCreateAndWaitCreateFails(t *testing.T) {
	mock := &mockAPI{
		createSandboxFn: func(ctx context.Context, body apis.CreateSandboxJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.CreateSandboxResponse, error) {
			return &apis.CreateSandboxResponse{
				HTTPResponse: httpResponse(500),
				Body:         []byte("internal error"),
			}, nil
		},
	}
	c := newTestClient(mock)
	_, _, err := c.CreateAndWait(context.Background(), CreateParams{TemplateID: "tmpl-1"}, WithPollInterval(100*time.Millisecond))
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Sandbox.WaitForReady polling ---

func TestWaitForReadyPolling(t *testing.T) {
	callCount := 0
	mock := &mockAPI{
		getSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.GetSandboxResponse, error) {
			callCount++
			state := apis.Paused
			if callCount >= 3 {
				state = apis.Running
			}
			return &apis.GetSandboxResponse{
				JSON200:      &apis.SandboxDetail{SandboxID: sandboxID, State: state},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	sb := newTestSandbox(c, "sb-123")
	info, err := sb.WaitForReady(context.Background(), WithPollInterval(50*time.Millisecond))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount < 3 {
		t.Errorf("expected at least 3 calls, got %d", callCount)
	}
	if info.State != StateRunning {
		t.Errorf("expected state 'running', got %q", info.State)
	}
}

// --- Error cases on instance methods ---

func TestKillError(t *testing.T) {
	mock := &mockAPI{
		deleteSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.DeleteSandboxResponse, error) {
			return &apis.DeleteSandboxResponse{
				HTTPResponse: httpResponse(404),
				Body:         []byte(`{"message":"not found"}`),
			}, nil
		},
	}
	c := newTestClient(mock)
	sb := newTestSandbox(c, "sb-999")
	err := sb.Kill(context.Background())
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

func TestSetTimeoutError(t *testing.T) {
	mock := &mockAPI{
		updateSandboxTimeoutFn: func(ctx context.Context, sandboxID apis.SandboxID, body apis.UpdateSandboxTimeoutJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.UpdateSandboxTimeoutResponse, error) {
			return &apis.UpdateSandboxTimeoutResponse{
				HTTPResponse: httpResponse(404),
				Body:         []byte(`{"message":"not found"}`),
			}, nil
		},
	}
	c := newTestClient(mock)
	sb := newTestSandbox(c, "sb-999")
	err := sb.SetTimeout(context.Background(), time.Minute)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPauseError(t *testing.T) {
	mock := &mockAPI{
		pauseSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.PauseSandboxResponse, error) {
			return &apis.PauseSandboxResponse{
				HTTPResponse: httpResponse(409),
				Body:         []byte(`{"message":"conflict"}`),
			}, nil
		},
	}
	c := newTestClient(mock)
	sb := newTestSandbox(c, "sb-123")
	err := sb.Pause(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestConnectError(t *testing.T) {
	mock := &mockAPI{
		connectSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, body apis.ConnectSandboxJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.ConnectSandboxResponse, error) {
			return &apis.ConnectSandboxResponse{
				HTTPResponse: httpResponse(404),
				Body:         []byte(`{"message":"not found"}`),
			}, nil
		},
	}
	c := newTestClient(mock)
	_, err := c.Connect(context.Background(), "sb-999", ConnectParams{Timeout: 60})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetInfoError(t *testing.T) {
	mock := &mockAPI{
		getSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.GetSandboxResponse, error) {
			return &apis.GetSandboxResponse{
				HTTPResponse: httpResponse(404),
				Body:         []byte(`{"message":"not found"}`),
			}, nil
		},
	}
	c := newTestClient(mock)
	sb := newTestSandbox(c, "sb-999")
	_, err := sb.GetInfo(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Template: CreateTemplate ---

func TestCreateTemplate(t *testing.T) {
	mock := &mockAPI{
		createTemplateV3Fn: func(ctx context.Context, body apis.CreateTemplateV3JSONRequestBody, editors ...apis.RequestEditorFn) (*apis.CreateTemplateV3Response, error) {
			return &apis.CreateTemplateV3Response{
				JSON202:      &apis.TemplateRequestResponseV3{TemplateID: "tmpl-new", BuildID: "build-1"},
				HTTPResponse: httpResponse(202),
			}, nil
		},
	}
	c := newTestClient(mock)
	resp, err := c.CreateTemplate(context.Background(), CreateTemplateParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TemplateID != "tmpl-new" {
		t.Errorf("expected template ID 'tmpl-new', got %q", resp.TemplateID)
	}
	if resp.BuildID != "build-1" {
		t.Errorf("expected build ID 'build-1', got %q", resp.BuildID)
	}
}

func TestCreateTemplateError(t *testing.T) {
	mock := &mockAPI{
		createTemplateV3Fn: func(ctx context.Context, body apis.CreateTemplateV3JSONRequestBody, editors ...apis.RequestEditorFn) (*apis.CreateTemplateV3Response, error) {
			return &apis.CreateTemplateV3Response{
				HTTPResponse: httpResponse(400),
				Body:         []byte(`{"message":"bad request"}`),
			}, nil
		},
	}
	c := newTestClient(mock)
	_, err := c.CreateTemplate(context.Background(), CreateTemplateParams{})
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Template: GetTemplate ---

func TestGetTemplate(t *testing.T) {
	mock := &mockAPI{
		getTemplateFn: func(ctx context.Context, templateID apis.TemplateID, params *apis.GetTemplateParams, editors ...apis.RequestEditorFn) (*apis.GetTemplateResponse, error) {
			return &apis.GetTemplateResponse{
				JSON200: &apis.TemplateWithBuilds{
					TemplateID: templateID,
					Builds:     []apis.TemplateBuild{{}},
				},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	tmpl, err := c.GetTemplate(context.Background(), "tmpl-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tmpl.TemplateID != "tmpl-1" {
		t.Errorf("expected template ID 'tmpl-1', got %q", tmpl.TemplateID)
	}
	if len(tmpl.Builds) != 1 {
		t.Errorf("expected 1 build, got %d", len(tmpl.Builds))
	}
}

// --- Template: GetTemplateBuildStatus ---

func TestGetTemplateBuildStatus(t *testing.T) {
	mock := &mockAPI{
		getTemplateBuildStatusFn: func(ctx context.Context, templateID apis.TemplateID, buildID apis.BuildID, params *apis.GetTemplateBuildStatusParams, editors ...apis.RequestEditorFn) (*apis.GetTemplateBuildStatusResponse, error) {
			return &apis.GetTemplateBuildStatusResponse{
				JSON200:      &apis.TemplateBuildInfo{TemplateID: templateID, BuildID: buildID, Status: apis.TemplateBuildStatusBuilding},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	info, err := c.GetTemplateBuildStatus(context.Background(), "tmpl-1", "build-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Status != BuildStatusBuilding {
		t.Errorf("expected status 'building', got %q", info.Status)
	}
}

// --- Template: WaitForBuild polling ---

func TestWaitForBuildPolling(t *testing.T) {
	callCount := 0
	mock := &mockAPI{
		getTemplateBuildStatusFn: func(ctx context.Context, templateID apis.TemplateID, buildID apis.BuildID, params *apis.GetTemplateBuildStatusParams, editors ...apis.RequestEditorFn) (*apis.GetTemplateBuildStatusResponse, error) {
			callCount++
			status := apis.TemplateBuildStatusBuilding
			if callCount >= 3 {
				status = apis.TemplateBuildStatusReady
			}
			return &apis.GetTemplateBuildStatusResponse{
				JSON200:      &apis.TemplateBuildInfo{TemplateID: templateID, BuildID: buildID, Status: status},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	info, err := c.WaitForBuild(context.Background(), "tmpl-1", "build-1", WithPollInterval(50*time.Millisecond))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount < 3 {
		t.Errorf("expected at least 3 calls, got %d", callCount)
	}
	if info.Status != BuildStatusReady {
		t.Errorf("expected status 'ready', got %q", info.Status)
	}
}

func TestWaitForBuildTimeout(t *testing.T) {
	mock := &mockAPI{
		getTemplateBuildStatusFn: func(ctx context.Context, templateID apis.TemplateID, buildID apis.BuildID, params *apis.GetTemplateBuildStatusParams, editors ...apis.RequestEditorFn) (*apis.GetTemplateBuildStatusResponse, error) {
			return &apis.GetTemplateBuildStatusResponse{
				JSON200:      &apis.TemplateBuildInfo{TemplateID: templateID, BuildID: buildID, Status: apis.TemplateBuildStatusBuilding},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err := c.WaitForBuild(ctx, "tmpl-1", "build-1", WithPollInterval(50*time.Millisecond))
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

// --- Template: DeleteTemplate errors ---

func TestDeleteTemplateError(t *testing.T) {
	mock := &mockAPI{
		deleteTemplateFn: func(ctx context.Context, templateID apis.TemplateID, editors ...apis.RequestEditorFn) (*apis.DeleteTemplateResponse, error) {
			return &apis.DeleteTemplateResponse{
				HTTPResponse: httpResponse(500),
				Body:         []byte("internal error"),
			}, nil
		},
	}
	c := newTestClient(mock)
	err := c.DeleteTemplate(context.Background(), "tmpl-1")
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Template: GetTemplateByAlias not found ---

func TestGetTemplateByAliasNotFound(t *testing.T) {
	mock := &mockAPI{
		getTemplateByAliasFn: func(ctx context.Context, alias string, editors ...apis.RequestEditorFn) (*apis.GetTemplateByAliasResponse, error) {
			return &apis.GetTemplateByAliasResponse{
				HTTPResponse: httpResponse(404),
				Body:         []byte(`{"message":"not found"}`),
			}, nil
		},
	}
	c := newTestClient(mock)
	_, err := c.GetTemplateByAlias(context.Background(), "nonexistent")
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

// --- Template: ListTemplates errors ---

func TestListTemplatesError(t *testing.T) {
	mock := &mockAPI{
		listTemplatesFn: func(ctx context.Context, params *apis.ListTemplatesParams, editors ...apis.RequestEditorFn) (*apis.ListTemplatesResponse, error) {
			return &apis.ListTemplatesResponse{
				HTTPResponse: httpResponse(401),
				Body:         []byte(`{"message":"unauthorized"}`),
			}, nil
		},
	}
	c := newTestClient(mock)
	_, err := c.ListTemplates(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Sandbox.GetSandboxesMetrics ---

func TestGetSandboxesMetrics(t *testing.T) {
	mock := &mockAPI{
		getSandboxesMetricsFn: func(ctx context.Context, params *apis.GetSandboxesMetricsParams, editors ...apis.RequestEditorFn) (*apis.GetSandboxesMetricsResponse, error) {
			return &apis.GetSandboxesMetricsResponse{
				JSON200:      &apis.SandboxesWithMetrics{Sandboxes: map[string]apis.SandboxMetric{"sb-1": {CPUCount: 4}}},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	metrics, err := c.GetSandboxesMetrics(context.Background(), &GetSandboxesMetricsParams{SandboxIds: []string{"sb-1"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if metrics.Sandboxes["sb-1"].CPUCount != 4 {
		t.Errorf("unexpected cpu count: %v", metrics.Sandboxes["sb-1"].CPUCount)
	}
}

// --- Connect returns JSON201 (new sandbox) ---

// boolPtr returns a pointer to the given bool value.
func boolPtr(v bool) *bool { return &v }

// --- Template: UpdateTemplate ---

func TestUpdateTemplate(t *testing.T) {
	mock := &mockAPI{
		updateTemplateFn: func(ctx context.Context, templateID apis.TemplateID, body apis.UpdateTemplateJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.UpdateTemplateResponse, error) {
			return &apis.UpdateTemplateResponse{HTTPResponse: httpResponse(200)}, nil
		},
	}
	c := newTestClient(mock)
	if err := c.UpdateTemplate(context.Background(), "tmpl-1", UpdateTemplateParams{Public: boolPtr(true)}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateTemplateError(t *testing.T) {
	mock := &mockAPI{
		updateTemplateFn: func(ctx context.Context, templateID apis.TemplateID, body apis.UpdateTemplateJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.UpdateTemplateResponse, error) {
			return &apis.UpdateTemplateResponse{
				HTTPResponse: httpResponse(400),
				Body:         []byte(`{"message":"bad request"}`),
			}, nil
		},
	}
	c := newTestClient(mock)
	err := c.UpdateTemplate(context.Background(), "tmpl-1", UpdateTemplateParams{Public: boolPtr(true)})
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Template: GetTemplateBuildLogs ---

func TestGetTemplateBuildLogs(t *testing.T) {
	mock := &mockAPI{
		getTemplateBuildLogsFn: func(ctx context.Context, templateID apis.TemplateID, buildID apis.BuildID, params *apis.GetTemplateBuildLogsParams, editors ...apis.RequestEditorFn) (*apis.GetTemplateBuildLogsResponse, error) {
			return &apis.GetTemplateBuildLogsResponse{
				JSON200:      &apis.TemplateBuildLogsResponse{Logs: []apis.BuildLogEntry{{Message: "step 1", Level: apis.LogLevelInfo}}},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	logs, err := c.GetTemplateBuildLogs(context.Background(), "tmpl-1", "build-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs.Logs) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(logs.Logs))
	}
	if logs.Logs[0].Message != "step 1" {
		t.Errorf("expected message 'step 1', got %q", logs.Logs[0].Message)
	}
}

func TestGetTemplateBuildLogsError(t *testing.T) {
	mock := &mockAPI{
		getTemplateBuildLogsFn: func(ctx context.Context, templateID apis.TemplateID, buildID apis.BuildID, params *apis.GetTemplateBuildLogsParams, editors ...apis.RequestEditorFn) (*apis.GetTemplateBuildLogsResponse, error) {
			return &apis.GetTemplateBuildLogsResponse{
				HTTPResponse: httpResponse(404),
				Body:         []byte(`{"message":"not found"}`),
			}, nil
		},
	}
	c := newTestClient(mock)
	_, err := c.GetTemplateBuildLogs(context.Background(), "tmpl-1", "build-1", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Template: StartTemplateBuild ---

func TestStartTemplateBuild(t *testing.T) {
	mock := &mockAPI{
		startTemplateBuildV2Fn: func(ctx context.Context, templateID apis.TemplateID, buildID apis.BuildID, body apis.StartTemplateBuildV2JSONRequestBody, editors ...apis.RequestEditorFn) (*apis.StartTemplateBuildV2Response, error) {
			return &apis.StartTemplateBuildV2Response{HTTPResponse: httpResponse(202)}, nil
		},
	}
	c := newTestClient(mock)
	if err := c.StartTemplateBuild(context.Background(), "tmpl-1", "build-1", StartTemplateBuildParams{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStartTemplateBuildError(t *testing.T) {
	mock := &mockAPI{
		startTemplateBuildV2Fn: func(ctx context.Context, templateID apis.TemplateID, buildID apis.BuildID, body apis.StartTemplateBuildV2JSONRequestBody, editors ...apis.RequestEditorFn) (*apis.StartTemplateBuildV2Response, error) {
			return &apis.StartTemplateBuildV2Response{
				HTTPResponse: httpResponse(500),
				Body:         []byte("internal error"),
			}, nil
		},
	}
	c := newTestClient(mock)
	err := c.StartTemplateBuild(context.Background(), "tmpl-1", "build-1", StartTemplateBuildParams{})
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Template: GetTemplateFiles ---

func TestGetTemplateFiles(t *testing.T) {
	mock := &mockAPI{
		getTemplateFilesFn: func(ctx context.Context, templateID apis.TemplateID, hash string, editors ...apis.RequestEditorFn) (*apis.GetTemplateFilesResponse, error) {
			return &apis.GetTemplateFilesResponse{
				JSON201:      &apis.TemplateBuildFileUpload{Present: true},
				HTTPResponse: httpResponse(201),
			}, nil
		},
	}
	c := newTestClient(mock)
	upload, err := c.GetTemplateFiles(context.Background(), "tmpl-1", "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !upload.Present {
		t.Error("expected Present to be true")
	}
}

func TestGetTemplateFilesError(t *testing.T) {
	mock := &mockAPI{
		getTemplateFilesFn: func(ctx context.Context, templateID apis.TemplateID, hash string, editors ...apis.RequestEditorFn) (*apis.GetTemplateFilesResponse, error) {
			return &apis.GetTemplateFilesResponse{
				HTTPResponse: httpResponse(404),
				Body:         []byte(`{"message":"not found"}`),
			}, nil
		},
	}
	c := newTestClient(mock)
	_, err := c.GetTemplateFiles(context.Background(), "tmpl-1", "abc123")
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Template: AssignTemplateTags ---

func TestAssignTemplateTags(t *testing.T) {
	buildUUID := openapi_types.UUID{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0}
	mock := &mockAPI{
		assignTemplateTagsFn: func(ctx context.Context, body apis.AssignTemplateTagsJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.AssignTemplateTagsResponse, error) {
			return &apis.AssignTemplateTagsResponse{
				JSON201:      &apis.AssignedTemplateTags{BuildID: buildUUID, Tags: []string{"latest"}},
				HTTPResponse: httpResponse(201),
			}, nil
		},
	}
	c := newTestClient(mock)
	result, err := c.AssignTemplateTags(context.Background(), ManageTagsParams{Tags: []string{"latest"}, Target: "my-tmpl:v1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Tags) != 1 || result.Tags[0] != "latest" {
		t.Errorf("unexpected tags: %v", result.Tags)
	}
}

func TestAssignTemplateTagsError(t *testing.T) {
	mock := &mockAPI{
		assignTemplateTagsFn: func(ctx context.Context, body apis.AssignTemplateTagsJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.AssignTemplateTagsResponse, error) {
			return &apis.AssignTemplateTagsResponse{
				HTTPResponse: httpResponse(400),
				Body:         []byte(`{"message":"bad request"}`),
			}, nil
		},
	}
	c := newTestClient(mock)
	_, err := c.AssignTemplateTags(context.Background(), ManageTagsParams{Tags: []string{"latest"}, Target: "my-tmpl:v1"})
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Template: DeleteTemplateTags ---

func TestDeleteTemplateTags(t *testing.T) {
	mock := &mockAPI{
		deleteTemplateTagsFn: func(ctx context.Context, body apis.DeleteTemplateTagsJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.DeleteTemplateTagsResponse, error) {
			return &apis.DeleteTemplateTagsResponse{HTTPResponse: httpResponse(204)}, nil
		},
	}
	c := newTestClient(mock)
	if err := c.DeleteTemplateTags(context.Background(), DeleteTagsParams{Name: "my-tmpl", Tags: []string{"old"}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteTemplateTagsError(t *testing.T) {
	mock := &mockAPI{
		deleteTemplateTagsFn: func(ctx context.Context, body apis.DeleteTemplateTagsJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.DeleteTemplateTagsResponse, error) {
			return &apis.DeleteTemplateTagsResponse{
				HTTPResponse: httpResponse(404),
				Body:         []byte(`{"message":"not found"}`),
			}, nil
		},
	}
	c := newTestClient(mock)
	err := c.DeleteTemplateTags(context.Background(), DeleteTagsParams{Name: "my-tmpl", Tags: []string{"old"}})
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Poll: WithBackoff ---

func TestPollWithBackoff(t *testing.T) {
	callCount := 0
	mock := &mockAPI{
		getTemplateBuildStatusFn: func(ctx context.Context, templateID apis.TemplateID, buildID apis.BuildID, params *apis.GetTemplateBuildStatusParams, editors ...apis.RequestEditorFn) (*apis.GetTemplateBuildStatusResponse, error) {
			callCount++
			status := apis.TemplateBuildStatusBuilding
			if callCount >= 4 {
				status = apis.TemplateBuildStatusReady
			}
			return &apis.GetTemplateBuildStatusResponse{
				JSON200:      &apis.TemplateBuildInfo{TemplateID: templateID, BuildID: buildID, Status: status},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	info, err := c.WaitForBuild(context.Background(), "tmpl-1", "build-1",
		WithPollInterval(50*time.Millisecond),
		WithBackoff(2.0, 500*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Status != BuildStatusReady {
		t.Errorf("expected status 'ready', got %q", info.Status)
	}
	if callCount < 4 {
		t.Errorf("expected at least 4 calls, got %d", callCount)
	}
}

// --- Poll: WithOnPoll ---

func TestPollWithOnPoll(t *testing.T) {
	callCount := 0
	var attempts []int
	mock := &mockAPI{
		getTemplateBuildStatusFn: func(ctx context.Context, templateID apis.TemplateID, buildID apis.BuildID, params *apis.GetTemplateBuildStatusParams, editors ...apis.RequestEditorFn) (*apis.GetTemplateBuildStatusResponse, error) {
			callCount++
			status := apis.TemplateBuildStatusBuilding
			if callCount >= 3 {
				status = apis.TemplateBuildStatusReady
			}
			return &apis.GetTemplateBuildStatusResponse{
				JSON200:      &apis.TemplateBuildInfo{TemplateID: templateID, BuildID: buildID, Status: status},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	_, err := c.WaitForBuild(context.Background(), "tmpl-1", "build-1",
		WithPollInterval(50*time.Millisecond),
		WithOnPoll(func(attempt int) { attempts = append(attempts, attempt) }),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(attempts) != 3 {
		t.Fatalf("expected 3 onPoll calls, got %d", len(attempts))
	}
	for i, a := range attempts {
		if a != i+1 {
			t.Errorf("expected attempt %d, got %d", i+1, a)
		}
	}
}

func TestConnectCreated(t *testing.T) {
	mock := &mockAPI{
		connectSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, body apis.ConnectSandboxJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.ConnectSandboxResponse, error) {
			return &apis.ConnectSandboxResponse{
				JSON201:      &apis.Sandbox{SandboxID: sandboxID, TemplateID: "tmpl-1"},
				HTTPResponse: httpResponse(201),
			}, nil
		},
		getSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.GetSandboxResponse, error) {
			return &apis.GetSandboxResponse{
				JSON200:      &apis.SandboxDetail{SandboxID: sandboxID},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	sb, err := c.Connect(context.Background(), "sb-new", ConnectParams{Timeout: 60})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sb.ID() != "sb-new" {
		t.Errorf("expected sandbox ID 'sb-new', got %q", sb.ID())
	}
}
