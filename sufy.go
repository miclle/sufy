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

package sufy

import (
	"github.com/sufy-dev/sufy/auth"
	"github.com/sufy-dev/sufy/baseconf"
	"github.com/sufy-dev/sufy/sandbox"
)

// -----------------------------------------------------------------------------

// Config is the configuration for the SUFY client.
type Config struct {
	// APIKey is the API key for authentication.
	APIKey string

	// AccessKey is the public key identifier for AK/SK authentication.
	// Must be used together with SecretKey. When both AK/SK and APIKey are
	// configured, AK/SK takes priority.
	AccessKey string

	// SecretKey is the secret key for AK/SK authentication.
	// Must be used together with AccessKey.
	SecretKey string

	// BaseURL is the base URL for the SUFY API.
	BaseURL string
}

// Client is the client for SUFY.
type Client struct {
	apiKey      string
	credentials *auth.Credentials
	baseURL     string
}

// New creates a new SUFY client.
func New(__xgo_optional_conf *Config) *Client {
	conf := __xgo_optional_conf
	if conf == nil {
		conf = &Config{}
	}

	var cred *auth.Credentials
	ak, sk := baseconf.RequireCredentials(conf.AccessKey, conf.SecretKey)
	if ak != "" && sk != "" {
		cred = auth.New(ak, sk)
	}

	return &Client{
		apiKey:      baseconf.RequireAPIKey(conf.APIKey),
		credentials: cred,
		baseURL:     baseconf.RequireBaseURL(conf.BaseURL),
	}
}

// -----------------------------------------------------------------------------

// Sandbox returns the sandbox client.
func (c *Client) Sandbox() *sandbox.Client {
	return sandbox.New(&sandbox.Config{
		APIKey:      c.apiKey,
		Credentials: c.credentials,
		BaseURL:     c.baseURL,
	})
}

// -----------------------------------------------------------------------------
