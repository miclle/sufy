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
	"strings"
	"testing"
)

// clearSandboxEnv wipes all sandbox-related env vars for the duration of the
// test, so cases start from a clean slate regardless of the shell environment.
func clearSandboxEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		EnvAPIKey, EnvAccessKey, EnvSecretKey,
		EnvBaseURL, EnvSandboxURL, EnvAuthMethod,
	} {
		t.Setenv(k, "")
	}
}

func TestNewSandboxClient_APIKeyOnly(t *testing.T) {
	clearSandboxEnv(t)
	t.Setenv(EnvAPIKey, "sk-test")

	client, err := NewSandboxClient()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatalf("expected non-nil client")
	}
}

func TestNewSandboxClient_AKSKOnly(t *testing.T) {
	clearSandboxEnv(t)
	t.Setenv(EnvAccessKey, "ak-test")
	t.Setenv(EnvSecretKey, "sk-test")

	client, err := NewSandboxClient()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatalf("expected non-nil client")
	}
}

func TestNewSandboxClient_AutoDetectPrefersAKSK(t *testing.T) {
	// When both credential sets are present, auto-detect should pick AK/SK.
	clearSandboxEnv(t)
	t.Setenv(EnvAPIKey, "sk-api")
	t.Setenv(EnvAccessKey, "ak-test")
	t.Setenv(EnvSecretKey, "sk-test")

	client, err := NewSandboxClient()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatalf("expected non-nil client")
	}
}

func TestNewSandboxClient_MissingCredentials(t *testing.T) {
	clearSandboxEnv(t)
	_, err := NewSandboxClient()
	if err == nil {
		t.Fatalf("expected error for missing credentials")
	}
	if !strings.Contains(err.Error(), "missing credentials") {
		t.Errorf("unexpected error text: %v", err)
	}
}

func TestNewSandboxClient_ExplicitAPIKeyMethod(t *testing.T) {
	clearSandboxEnv(t)
	t.Setenv(EnvAuthMethod, "api-key")
	t.Setenv(EnvAPIKey, "sk-test")

	if _, err := NewSandboxClient(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewSandboxClient_ExplicitAPIKeyMethodMissingKey(t *testing.T) {
	clearSandboxEnv(t)
	t.Setenv(EnvAuthMethod, "api-key")

	_, err := NewSandboxClient()
	if err == nil || !strings.Contains(err.Error(), "SUFY_API_KEY") {
		t.Errorf("expected SUFY_API_KEY error, got %v", err)
	}
}

func TestNewSandboxClient_ExplicitAPIKeyMethodIgnoresAKSK(t *testing.T) {
	// With method=api-key, AK/SK alone must not satisfy auth.
	clearSandboxEnv(t)
	t.Setenv(EnvAuthMethod, "api-key")
	t.Setenv(EnvAccessKey, "ak")
	t.Setenv(EnvSecretKey, "sk")

	_, err := NewSandboxClient()
	if err == nil {
		t.Errorf("expected error when api-key method forced without API key")
	}
}

func TestNewSandboxClient_ExplicitAKSKMethod(t *testing.T) {
	clearSandboxEnv(t)
	t.Setenv(EnvAuthMethod, "ak-sk")
	t.Setenv(EnvAccessKey, "ak-test")
	t.Setenv(EnvSecretKey, "sk-test")

	if _, err := NewSandboxClient(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewSandboxClient_ExplicitAKSKMissingOne(t *testing.T) {
	clearSandboxEnv(t)
	t.Setenv(EnvAuthMethod, "ak-sk")
	t.Setenv(EnvAccessKey, "ak-test")

	_, err := NewSandboxClient()
	if err == nil || !strings.Contains(err.Error(), "SUFY_ACCESS_KEY") {
		t.Errorf("expected AK/SK error, got %v", err)
	}
}

func TestNewSandboxClient_ExplicitAKSKIgnoresAPIKey(t *testing.T) {
	// With method=ak-sk, an API key alone must not satisfy auth.
	clearSandboxEnv(t)
	t.Setenv(EnvAuthMethod, "ak-sk")
	t.Setenv(EnvAPIKey, "sk-test")

	_, err := NewSandboxClient()
	if err == nil {
		t.Errorf("expected error when ak-sk method forced without AK/SK")
	}
}

func TestNewSandboxClient_UnknownAuthMethod(t *testing.T) {
	clearSandboxEnv(t)
	t.Setenv(EnvAuthMethod, "oauth")

	_, err := NewSandboxClient()
	if err == nil || !strings.Contains(err.Error(), "unknown SUFY_AUTH_METHOD") {
		t.Errorf("expected unknown method error, got %v", err)
	}
}

func TestNewSandboxClient_BaseURLFromEnvBaseURL(t *testing.T) {
	// Just assert construction succeeds with base URL env set.
	clearSandboxEnv(t)
	t.Setenv(EnvAPIKey, "sk-test")
	t.Setenv(EnvBaseURL, "https://api.example.com")

	if _, err := NewSandboxClient(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewSandboxClient_BaseURLFallbackToSandboxURL(t *testing.T) {
	clearSandboxEnv(t)
	t.Setenv(EnvAPIKey, "sk-test")
	t.Setenv(EnvSandboxURL, "https://sandbox.example.com")

	if _, err := NewSandboxClient(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
