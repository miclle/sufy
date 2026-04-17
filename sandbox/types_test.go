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

//go:build unit

package sandbox

import (
	"testing"
	"time"

	"github.com/sufy-dev/sufy/sandbox/internal/apis"
)

func int32Ptr(v int32) *int32 { return &v }
func int64Ptr(v int64) *int64 { return &v }
func intPtr(v int) *int       { return &v }

func TestCreateParamsToAPI(t *testing.T) {
	envVars := map[string]string{"FOO": "bar"}
	meta := Metadata{"key": "val"}
	allowPublic := true
	params := CreateParams{
		TemplateID:          "tmpl-1",
		Timeout:             int32Ptr(300),
		AutoPause:           boolPtr(true),
		AllowInternetAccess: boolPtr(false),
		Secure:              boolPtr(true),
		EnvVars:             &envVars,
		Metadata:            &meta,
		Network: &NetworkConfig{
			AllowPublicTraffic: &allowPublic,
		},
		Injections: &[]SandboxInjectionSpec{
			{OpenAI: &OpenAIInjection{APIKey: strPtr("sk-test")}},
		},
	}
	body, err := params.toAPI()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body.TemplateID != "tmpl-1" {
		t.Errorf("TemplateID = %q, want %q", body.TemplateID, "tmpl-1")
	}
	if body.Timeout == nil || *body.Timeout != 300 {
		t.Errorf("Timeout = %v, want 300", body.Timeout)
	}
	if body.AutoPause == nil || !*body.AutoPause {
		t.Error("AutoPause should be true")
	}
	if body.AllowInternetAccess == nil || *body.AllowInternetAccess {
		t.Error("AllowInternetAccess should be false")
	}
	if body.Secure == nil || !*body.Secure {
		t.Error("Secure should be true")
	}
	if body.EnvVars == nil {
		t.Fatal("EnvVars should be set")
	}
	if body.Metadata == nil {
		t.Fatal("Metadata should be set")
	}
	if body.Network == nil {
		t.Fatal("Network should be set")
	}
	if body.Network.AllowPublicTraffic == nil || !*body.Network.AllowPublicTraffic {
		t.Error("AllowPublicTraffic should be true")
	}
	if body.Injections == nil || len(*body.Injections) != 1 {
		t.Fatalf("Injections length = %v, want 1", body.Injections)
	}
}

func TestCreateParamsToAPINilOptionals(t *testing.T) {
	params := CreateParams{TemplateID: "tmpl-2"}
	body, err := params.toAPI()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body.TemplateID != "tmpl-2" {
		t.Errorf("TemplateID = %q, want %q", body.TemplateID, "tmpl-2")
	}
	if body.EnvVars != nil {
		t.Error("EnvVars should be nil")
	}
	if body.Metadata != nil {
		t.Error("Metadata should be nil")
	}
	if body.Network != nil {
		t.Error("Network should be nil")
	}
	if body.Injections != nil {
		t.Error("Injections should be nil")
	}
}

func TestConnectParamsToAPI(t *testing.T) {
	params := ConnectParams{Timeout: 600}
	body := params.toAPI()
	if body.Timeout != 600 {
		t.Errorf("Timeout = %d, want 600", body.Timeout)
	}
}

func TestRefreshParamsToAPI(t *testing.T) {
	params := RefreshParams{Duration: intPtr(120)}
	body := params.toAPI()
	if body.Duration == nil || *body.Duration != 120 {
		t.Errorf("Duration = %v, want 120", body.Duration)
	}
}

func TestListParamsToAPI(t *testing.T) {
	states := []SandboxState{StateRunning, StatePaused}
	params := ListParams{State: &states}
	result := params.toAPI()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.State == nil || len(*result.State) != 2 {
		t.Fatalf("State length = %v, want 2", result.State)
	}
	if (*result.State)[0] != apis.SandboxState("running") {
		t.Errorf("State[0] = %q, want %q", (*result.State)[0], "running")
	}
	if (*result.State)[1] != apis.SandboxState("paused") {
		t.Errorf("State[1] = %q, want %q", (*result.State)[1], "paused")
	}
}

func TestListParamsToAPINil(t *testing.T) {
	var p *ListParams
	if p.toAPI() != nil {
		t.Error("nil receiver should return nil")
	}
}

func TestGetMetricsParamsToAPI(t *testing.T) {
	params := GetMetricsParams{Start: int64Ptr(1000), End: int64Ptr(2000)}
	result := params.toAPI()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if *result.Start != 1000 || *result.End != 2000 {
		t.Errorf("Start/End = %v/%v, want 1000/2000", *result.Start, *result.End)
	}
}

func TestGetMetricsParamsToAPINil(t *testing.T) {
	var p *GetMetricsParams
	if p.toAPI() != nil {
		t.Error("nil receiver should return nil")
	}
}

func TestGetLogsParamsToAPI(t *testing.T) {
	params := GetLogsParams{Start: int64Ptr(500), Limit: int32Ptr(100)}
	result := params.toAPI()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if *result.Start != 500 || *result.Limit != 100 {
		t.Errorf("Start/Limit = %v/%v, want 500/100", *result.Start, *result.Limit)
	}
}

func TestGetLogsParamsToAPINil(t *testing.T) {
	var p *GetLogsParams
	if p.toAPI() != nil {
		t.Error("nil receiver should return nil")
	}
}

func TestGetSandboxesMetricsParamsToAPI(t *testing.T) {
	params := GetSandboxesMetricsParams{SandboxIds: []string{"sb-1", "sb-2"}}
	result := params.toAPI()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.SandboxIds) != 2 || result.SandboxIds[0] != "sb-1" {
		t.Errorf("SandboxIds = %v, want [sb-1 sb-2]", result.SandboxIds)
	}
}

func TestGetSandboxesMetricsParamsToAPINil(t *testing.T) {
	var p *GetSandboxesMetricsParams
	if p.toAPI() != nil {
		t.Error("nil receiver should return nil")
	}
}

func TestSandboxInfoFromAPI(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	alias := "my-alias"
	domain := "example.com"
	meta := apis.SandboxMetadata{"k": "v"}
	detail := &apis.SandboxDetail{
		SandboxID:   "sb-1",
		TemplateID:  "tmpl-1",
		ClientID:    "client-1",
		Alias:       &alias,
		Domain:      &domain,
		State:       apis.SandboxState("running"),
		CPUCount:    2,
		MemoryMB:    512,
		DiskSizeMB:  1024,
		EnvdVersion: "0.1.0",
		StartedAt:   now,
		EndAt:       now.Add(time.Hour),
		Metadata:    &meta,
	}
	info := sandboxInfoFromAPI(detail)
	if info == nil {
		t.Fatal("expected non-nil info")
	}
	if info.SandboxID != "sb-1" {
		t.Errorf("SandboxID = %q, want %q", info.SandboxID, "sb-1")
	}
	if info.Alias == nil || *info.Alias != "my-alias" {
		t.Errorf("Alias = %v, want %q", info.Alias, "my-alias")
	}
	if info.Domain == nil || *info.Domain != "example.com" {
		t.Errorf("Domain = %v, want %q", info.Domain, "example.com")
	}
	if info.State != StateRunning {
		t.Errorf("State = %q, want %q", info.State, StateRunning)
	}
	if info.CPUCount != 2 || info.MemoryMB != 512 || info.DiskSizeMB != 1024 {
		t.Errorf("resources = %d/%d/%d, want 2/512/1024", info.CPUCount, info.MemoryMB, info.DiskSizeMB)
	}
	if info.Metadata == nil || (*info.Metadata)["k"] != "v" {
		t.Errorf("Metadata = %v, want map[k:v]", info.Metadata)
	}
	if !info.StartedAt.Equal(now) {
		t.Errorf("StartedAt = %v, want %v", info.StartedAt, now)
	}

	// nil input
	if sandboxInfoFromAPI(nil) != nil {
		t.Error("nil input should return nil")
	}
}
