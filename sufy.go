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
	"github.com/sufy-dev/sufy/baseconf"
	"github.com/sufy-dev/sufy/sandbox"
)

// -----------------------------------------------------------------------------

// Config is the configuration for the SUFY client.
type Config struct {
	APIKey  string // API key for authentication
	BaseURL string // Base URL for the SUFY API
}

// Client is the client for SUFY.
type Client struct {
	apiKey  string
	baseURL string
}

// New creates a new SUFY client.
func New(__xgo_optional_conf *Config) *Client {
	conf := __xgo_optional_conf
	if conf == nil {
		conf = &Config{}
	}
	return &Client{
		apiKey:  baseconf.RequireAPIKey(conf.APIKey),
		baseURL: baseconf.RequireBaseURL(conf.BaseURL),
	}
}

// -----------------------------------------------------------------------------

// Sandbox returns the sandbox client.
func (c *Client) Sandbox() *sandbox.Client {
	return sandbox.New(&sandbox.Config{
		APIKey:  c.apiKey,
		BaseURL: c.baseURL,
	})
}

// -----------------------------------------------------------------------------
