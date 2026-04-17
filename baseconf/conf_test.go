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

package baseconf

import (
	"testing"
)

func TestRequireAPIKey(t *testing.T) {
	// Case 1: value provided, returns it.
	if got := RequireAPIKey("my-key"); got != "my-key" {
		t.Errorf("RequireAPIKey(\"my-key\") = %q, want %q", got, "my-key")
	}

	// Case 2: empty value, env var set.
	t.Setenv("SUFY_API_KEY", "env-key")
	if got := RequireAPIKey(""); got != "env-key" {
		t.Errorf("RequireAPIKey(\"\") with env = %q, want %q", got, "env-key")
	}

	// Case 3: empty value, no env var.
	t.Setenv("SUFY_API_KEY", "")
	if got := RequireAPIKey(""); got != "" {
		t.Errorf("RequireAPIKey(\"\") without env = %q, want %q", got, "")
	}
}

func TestRequireCredentials(t *testing.T) {
	// Case 1: both provided.
	ak, sk := RequireCredentials("ak", "sk")
	if ak != "ak" || sk != "sk" {
		t.Errorf("RequireCredentials(\"ak\", \"sk\") = (%q, %q), want (\"ak\", \"sk\")", ak, sk)
	}

	// Case 2: empty ak, env var set.
	t.Setenv("SUFY_ACCESS_KEY", "env-ak")
	t.Setenv("SUFY_SECRET_KEY", "env-sk")
	ak, sk = RequireCredentials("", "")
	if ak != "env-ak" || sk != "env-sk" {
		t.Errorf("RequireCredentials(\"\", \"\") with env = (%q, %q), want (\"env-ak\", \"env-sk\")", ak, sk)
	}

	// Case 3: empty both, no env vars.
	t.Setenv("SUFY_ACCESS_KEY", "")
	t.Setenv("SUFY_SECRET_KEY", "")
	ak, sk = RequireCredentials("", "")
	if ak != "" || sk != "" {
		t.Errorf("RequireCredentials(\"\", \"\") without env = (%q, %q), want (\"\", \"\")", ak, sk)
	}
}

func TestRequireBaseURL(t *testing.T) {
	// Case 1: value provided.
	if got := RequireBaseURL("https://custom.api"); got != "https://custom.api" {
		t.Errorf("RequireBaseURL(\"https://custom.api\") = %q, want %q", got, "https://custom.api")
	}

	// Case 2: empty, env var set.
	t.Setenv("SUFY_BASE_URL", "https://env.api")
	if got := RequireBaseURL(""); got != "https://env.api" {
		t.Errorf("RequireBaseURL(\"\") with env = %q, want %q", got, "https://env.api")
	}

	// Case 3: empty, no env var — returns DefaultBaseURL.
	t.Setenv("SUFY_BASE_URL", "")
	if got := RequireBaseURL(""); got != DefaultBaseURL {
		t.Errorf("RequireBaseURL(\"\") without env = %q, want %q", got, DefaultBaseURL)
	}
}
