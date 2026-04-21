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
	"reflect"
	"strings"
	"testing"

	sdk "github.com/sufy-dev/sufy/sandbox"
)

func TestParseStates(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []sdk.SandboxState
	}{
		{"empty", "", []sdk.SandboxState{}},
		{"single", "running", []sdk.SandboxState{sdk.SandboxState("running")}},
		{"multi", "running,paused", []sdk.SandboxState{sdk.SandboxState("running"), sdk.SandboxState("paused")}},
		{"whitespace", " running , paused ", []sdk.SandboxState{sdk.SandboxState("running"), sdk.SandboxState("paused")}},
		{"drops empties", "running,,paused,", []sdk.SandboxState{sdk.SandboxState("running"), sdk.SandboxState("paused")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseStates(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseStates(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseMetadataQuery(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"k=v", "k=v"},
		{"a=1,b=2", "a=1&b=2"},
		{" a = 1 , b = 2 ", "a=1&b=2"},
		{"missing,valid=v", "valid=v"},
		{"k=,=v", ""}, // empty key or value dropped
	}
	for _, tt := range tests {
		if got := ParseMetadataQuery(tt.in); got != tt.want {
			t.Errorf("ParseMetadataQuery(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestParseKeyValueMap(t *testing.T) {
	m := ParseKeyValueMap("a=1,b=2, c = 3")
	want := map[string]string{"a": "1", "b": "2", "c": "3"}
	if !reflect.DeepEqual(m, want) {
		t.Errorf("got %v, want %v", m, want)
	}

	if m := ParseKeyValueMap(""); len(m) != 0 {
		t.Errorf("expected empty map, got %v", m)
	}
}

func TestSplitCSV(t *testing.T) {
	if got := SplitCSV(""); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
	got := SplitCSV(" a , ,b , c ")
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestBuildInjectionSpec_OpenAI(t *testing.T) {
	spec, err := BuildInjectionSpec("openai", "sk-abc", "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spec.OpenAI == nil || spec.OpenAI.APIKey == nil || *spec.OpenAI.APIKey != "sk-abc" {
		t.Errorf("unexpected OpenAI spec: %+v", spec.OpenAI)
	}
}

func TestBuildInjectionSpec_Anthropic(t *testing.T) {
	spec, err := BuildInjectionSpec("ANTHROPIC", "", "https://api.anthropic.com", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spec.Anthropic == nil || spec.Anthropic.BaseURL == nil || *spec.Anthropic.BaseURL != "https://api.anthropic.com" {
		t.Errorf("unexpected Anthropic spec: %+v", spec.Anthropic)
	}
	if spec.Anthropic.APIKey != nil {
		t.Errorf("expected nil api key, got %v", *spec.Anthropic.APIKey)
	}
}

func TestBuildInjectionSpec_Gemini(t *testing.T) {
	spec, err := BuildInjectionSpec("gemini", "key", "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spec.Gemini == nil {
		t.Errorf("expected gemini spec")
	}
}

func TestBuildInjectionSpec_HTTP_RequiresBaseURL(t *testing.T) {
	_, err := BuildInjectionSpec("http", "", "", nil)
	if err == nil {
		t.Errorf("expected error when base url missing")
	}
}

func TestBuildInjectionSpec_HTTP_Valid(t *testing.T) {
	headers := map[string]string{"X-Auth": "token"}
	spec, err := BuildInjectionSpec("http", "", "https://example.com", headers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spec.HTTP == nil || spec.HTTP.BaseURL != "https://example.com" {
		t.Errorf("unexpected HTTP spec: %+v", spec.HTTP)
	}
	if spec.HTTP.Headers == nil || (*spec.HTTP.Headers)["X-Auth"] != "token" {
		t.Errorf("expected headers, got %+v", spec.HTTP.Headers)
	}
}

func TestBuildInjectionSpec_HTTP_InvalidURL(t *testing.T) {
	_, err := BuildInjectionSpec("http", "", "not-a-url", nil)
	if err == nil {
		t.Errorf("expected invalid URL error")
	}
}

func TestBuildInjectionSpec_EmptyType(t *testing.T) {
	_, err := BuildInjectionSpec("", "", "", nil)
	if err == nil || !strings.Contains(err.Error(), "required") {
		t.Errorf("expected required error, got %v", err)
	}
}

func TestBuildInjectionSpec_Unknown(t *testing.T) {
	_, err := BuildInjectionSpec("xgrok", "", "", nil)
	if err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("expected unsupported error, got %v", err)
	}
}

func TestBuildSandboxInjections_EmptyReturnsNil(t *testing.T) {
	got, err := BuildSandboxInjections(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestBuildSandboxInjections_RuleIDs(t *testing.T) {
	got, err := BuildSandboxInjections([]string{"rid-1", "rid-2"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 specs, got %d", len(got))
	}
	if got[0].ByID == nil || *got[0].ByID != "rid-1" {
		t.Errorf("unexpected first spec: %+v", got[0])
	}
}

func TestBuildSandboxInjections_EmptyRuleIDError(t *testing.T) {
	_, err := BuildSandboxInjections([]string{"  "}, nil)
	if err == nil || !strings.Contains(err.Error(), "cannot be empty") {
		t.Errorf("expected empty rule ID error, got %v", err)
	}
}

func TestBuildSandboxInjections_InlineOpenAI(t *testing.T) {
	got, err := BuildSandboxInjections(nil, []string{"type=openai,api-key=sk-xxx"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].OpenAI == nil {
		t.Fatalf("expected 1 OpenAI inline spec, got %+v", got)
	}
	if got[0].OpenAI.APIKey == nil || *got[0].OpenAI.APIKey != "sk-xxx" {
		t.Errorf("unexpected inline OpenAI key: %+v", got[0].OpenAI)
	}
}

func TestBuildSandboxInjections_InlineHTTPWithHeaders(t *testing.T) {
	got, err := BuildSandboxInjections(nil, []string{"type=http,base-url=https://h.example.com,headers=A=1,B=2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].HTTP == nil {
		t.Fatalf("expected HTTP inline spec, got %+v", got)
	}
	if got[0].HTTP.BaseURL != "https://h.example.com" {
		t.Errorf("unexpected base url: %q", got[0].HTTP.BaseURL)
	}
	if got[0].HTTP.Headers == nil || (*got[0].HTTP.Headers)["A"] != "1" || (*got[0].HTTP.Headers)["B"] != "2" {
		t.Errorf("headers parsed incorrectly: %+v", got[0].HTTP.Headers)
	}
}

func TestBuildSandboxInjections_InlineInvalid(t *testing.T) {
	_, err := BuildSandboxInjections(nil, []string{"type=bogus"})
	if err == nil {
		t.Errorf("expected error")
	}
}

func TestParseEnvPairs(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want map[string]string
	}{
		{"nil", nil, nil},
		{"empty", []string{"", "   "}, nil},
		{"no equal ignored", []string{"FOO"}, nil},
		{"basic", []string{"FOO=bar", "BAZ=qux"}, map[string]string{"FOO": "bar", "BAZ": "qux"}},
		{"trim", []string{" FOO = bar "}, map[string]string{"FOO": "bar"}},
		{"empty key skipped", []string{"=bar"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseEnvPairs(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseEnvPairs(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestValidateBaseURL(t *testing.T) {
	if _, err := validateBaseURL("", true); err == nil {
		t.Errorf("expected error when required and empty")
	}
	if s, err := validateBaseURL("", false); err != nil || s != "" {
		t.Errorf("expected empty ok, got %q %v", s, err)
	}
	if _, err := validateBaseURL("ftp://h.example.com", true); err == nil {
		t.Errorf("expected error for ftp scheme")
	}
	if _, err := validateBaseURL("no-scheme.example.com", true); err == nil {
		t.Errorf("expected error for no-scheme URL")
	}
	if s, err := validateBaseURL("https://h.example.com", true); err != nil || s != "https://h.example.com" {
		t.Errorf("unexpected result: %q %v", s, err)
	}
}

func TestOptionalString(t *testing.T) {
	if got := optionalString(""); got != nil {
		t.Errorf("expected nil, got %v", *got)
	}
	if got := optionalString(" "); got != nil {
		t.Errorf("expected nil, got %v", *got)
	}
	if got := optionalString("  value  "); got == nil || *got != "value" {
		t.Errorf("expected trimmed value, got %v", got)
	}
}

func TestParseInlineInjectionFields_HeadersLast(t *testing.T) {
	fields := parseInlineInjectionFields("type=http,base-url=https://x,headers=A=1,B=2,C=3")
	if fields["type"] != "http" {
		t.Errorf("type = %q", fields["type"])
	}
	if fields["base-url"] != "https://x" {
		t.Errorf("base-url = %q", fields["base-url"])
	}
	if fields["headers"] != "A=1,B=2,C=3" {
		t.Errorf("headers = %q", fields["headers"])
	}
}

func TestParseInlineInjectionFields_HeadersOnly(t *testing.T) {
	fields := parseInlineInjectionFields("headers=A=1,B=2")
	if fields["headers"] != "A=1,B=2" {
		t.Errorf("got %q", fields["headers"])
	}
}
