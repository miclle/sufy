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

// Package sandbox provides the handlers and shared helpers for the sufy sandbox CLI commands.
package sandbox

import (
	"fmt"
	"os"

	"github.com/sufy-dev/sufy/auth"
	sdk "github.com/sufy-dev/sufy/sandbox"
)

// Environment variable names read by the CLI for authentication.
const (
	EnvAPIKey     = "SUFY_API_KEY"
	EnvAccessKey  = "SUFY_ACCESS_KEY"
	EnvSecretKey  = "SUFY_SECRET_KEY"
	EnvBaseURL    = "SUFY_BASE_URL"
	EnvSandboxURL = "SUFY_SANDBOX_API_URL"
	// EnvAuthMethod overrides automatic credential selection. Accepted values:
	//   "api-key" — use SUFY_API_KEY only
	//   "ak-sk"   — use SUFY_ACCESS_KEY + SUFY_SECRET_KEY only
	// When unset, AK/SK takes priority if both are configured.
	EnvAuthMethod = "SUFY_AUTH_METHOD"
)

// NewSandboxClient builds a sandbox client from environment variables.
//
// When SUFY_AUTH_METHOD is set, only that method is used:
//   - "api-key" — SUFY_API_KEY
//   - "ak-sk"   — SUFY_ACCESS_KEY + SUFY_SECRET_KEY
//
// Otherwise credentials are selected automatically with AK/SK taking priority.
func NewSandboxClient() (*sdk.Client, error) {
	method := os.Getenv(EnvAuthMethod)

	var cred *auth.Credentials
	var apiKey string

	switch method {
	case "api-key":
		apiKey = os.Getenv(EnvAPIKey)
		if apiKey == "" {
			return nil, fmt.Errorf("SUFY_AUTH_METHOD=api-key but %s is not set", EnvAPIKey)
		}
	case "ak-sk":
		ak, sk := os.Getenv(EnvAccessKey), os.Getenv(EnvSecretKey)
		if ak == "" || sk == "" {
			return nil, fmt.Errorf("SUFY_AUTH_METHOD=ak-sk but %s/%s are not both set", EnvAccessKey, EnvSecretKey)
		}
		cred = auth.New(ak, sk)
	case "":
		// Auto-detect: AK/SK takes priority when both are present.
		if ak, sk := os.Getenv(EnvAccessKey), os.Getenv(EnvSecretKey); ak != "" && sk != "" {
			cred = auth.New(ak, sk)
		}
		apiKey = os.Getenv(EnvAPIKey)
	default:
		return nil, fmt.Errorf("unknown SUFY_AUTH_METHOD %q, must be \"api-key\" or \"ak-sk\"", method)
	}

	if cred == nil && apiKey == "" {
		return nil, fmt.Errorf("missing credentials: set %s or %s/%s environment variables",
			EnvAPIKey, EnvAccessKey, EnvSecretKey)
	}

	baseURL := os.Getenv(EnvBaseURL)
	if baseURL == "" {
		baseURL = os.Getenv(EnvSandboxURL)
	}

	return sdk.New(&sdk.Config{
		APIKey:      apiKey,
		Credentials: cred,
		BaseURL:     baseURL,
	}), nil
}

// MustNewSandboxClient is like NewSandboxClient but prints the error and exits
// on failure. Intended for use at the top of command handlers.
func MustNewSandboxClient() *sdk.Client {
	c, err := NewSandboxClient()
	if err != nil {
		PrintError("%v", err)
		os.Exit(1)
	}
	return c
}
