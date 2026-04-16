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
	"net/http"

	"github.com/sufy-dev/sufy/baseconf"
	"github.com/sufy-dev/sufy/sandbox/internal/apis"
)

// Config configures the sandbox client.
type Config struct {
	// APIKey is the API key used for authentication. If empty, the SUFY_API_KEY
	// environment variable is used. If neither is set, New panics.
	APIKey string

	// BaseURL is the sandbox API endpoint. If empty, the SUFY_BASE_URL
	// environment variable is used, falling back to baseconf.DefaultBaseURL.
	BaseURL string

	// HTTPClient is an optional custom HTTP client. Defaults to http.DefaultClient.
	HTTPClient *http.Client
}

// Client is the sandbox client.
type Client struct {
	config *Config
	api    apis.ClientWithResponsesInterface
}

// New creates a new sandbox client.
//
// The configuration parameter is optional. When omitted, the API key and base
// URL fall back to the SUFY_API_KEY and SUFY_BASE_URL environment variables.
func New(__xgo_optional_conf *Config) *Client {
	conf := __xgo_optional_conf
	if conf == nil {
		conf = &Config{}
	}
	conf.APIKey = baseconf.RequireAPIKey(conf.APIKey)
	conf.BaseURL = baseconf.RequireBaseURL(conf.BaseURL)
	if conf.HTTPClient == nil {
		conf.HTTPClient = http.DefaultClient
	}

	opts := []apis.ClientOption{
		apis.WithHTTPClient(conf.HTTPClient),
		apis.WithRequestEditorFn(reqidEditor()),
		apis.WithRequestEditorFn(apiKeyEditor(conf.APIKey)),
	}

	api, err := apis.NewClientWithResponses(conf.BaseURL, opts...)
	if err != nil {
		// NewClientWithResponses only fails when the server URL cannot be
		// parsed, which should not happen with baseconf-validated input.
		panic(err)
	}

	return &Client{config: conf, api: api}
}

// reqidEditor returns a RequestEditorFn that injects the X-Reqid header using
// the request ID carried by the context.
func reqidEditor() apis.RequestEditorFn {
	return func(ctx context.Context, req *http.Request) error {
		setReqidHeader(ctx, req)
		return nil
	}
}

// apiKeyEditor returns a RequestEditorFn that injects the X-API-Key header.
func apiKeyEditor(apiKey string) apis.RequestEditorFn {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Set("X-API-Key", apiKey)
		return nil
	}
}
