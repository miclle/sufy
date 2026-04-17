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

package auth

import (
	"net/http"
	"strings"
	"testing"
)

func TestSign(t *testing.T) {
	cred := New("ak", "sk")

	tests := []struct {
		data string
		want string
	}{
		{"hello", "ak:NDN8cM0rwosxhHJ6QAcI7ialr0g="},
		{"world", "ak:wZ-sw41ayh3PFDmQA-D3o7eBJIY="},
	}
	for _, tt := range tests {
		got := cred.Sign([]byte(tt.data))
		if got != tt.want {
			t.Errorf("Sign(%q) = %q, want %q", tt.data, got, tt.want)
		}
	}
}

func TestSignRequestV2_JSON(t *testing.T) {
	cred := New("ak", "sk")

	body := `{"name":"test"}`
	req, _ := http.NewRequest("POST", "https://api.example.com/injection-rules", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	token, err := cred.SignRequestV2(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	// Token should start with "ak:"
	if !strings.HasPrefix(token, "ak:") {
		t.Errorf("expected token to start with 'ak:', got %q", token)
	}
}

func TestSignRequestV2_NoBody(t *testing.T) {
	cred := New("ak", "sk")

	req, _ := http.NewRequest("GET", "https://api.example.com/injection-rules", nil)

	token, err := cred.SignRequestV2(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(token, "ak:") {
		t.Errorf("expected token to start with 'ak:', got %q", token)
	}
}

func TestSignRequestV2_DefaultContentType(t *testing.T) {
	cred := New("ak", "sk")

	req, _ := http.NewRequest("GET", "https://api.example.com/test", nil)
	// Don't set Content-Type — should default to form.

	_, err := cred.SignRequestV2(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ct := req.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
		t.Errorf("expected default Content-Type, got %q", ct)
	}
}

func TestSignRequestV2_XSufyHeaders(t *testing.T) {
	cred := New("ak", "sk")

	req, _ := http.NewRequest("GET", "https://api.example.com/test", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Sufy-Aaa", "val-a")
	req.Header.Set("X-Sufy-Bbb", "val-b")

	token1, err := cred.SignRequestV2(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Same request without X-Sufy headers should produce different signature.
	req2, _ := http.NewRequest("GET", "https://api.example.com/test", nil)
	req2.Header.Set("Content-Type", "application/json")

	token2, err := cred.SignRequestV2(req2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if token1 == token2 {
		t.Error("expected different tokens with and without X-Sufy headers")
	}
}

func TestSignRequestV2_Deterministic(t *testing.T) {
	cred := New("test-ak", "test-sk")

	makeReq := func() *http.Request {
		req, _ := http.NewRequest("POST", "https://api.example.com/injection-rules", strings.NewReader(`{"name":"rule1"}`))
		req.Header.Set("Content-Type", "application/json")
		return req
	}

	token1, _ := cred.SignRequestV2(makeReq())
	token2, _ := cred.SignRequestV2(makeReq())

	if token1 != token2 {
		t.Errorf("expected deterministic signing, got %q and %q", token1, token2)
	}
}

func TestNew(t *testing.T) {
	cred := New("my-ak", "my-sk")
	if cred.AccessKey != "my-ak" {
		t.Errorf("expected AccessKey 'my-ak', got %q", cred.AccessKey)
	}
	if string(cred.SecretKey) != "my-sk" {
		t.Errorf("expected SecretKey 'my-sk', got %q", string(cred.SecretKey))
	}
}
