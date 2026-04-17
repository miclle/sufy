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
	"fmt"
	"net/http"

	"github.com/sufy-dev/sufy/auth"
	"github.com/sufy-dev/sufy/baseconf"
	"github.com/sufy-dev/sufy/sandbox/internal/apis"
)

// Config configures the sandbox client.
type Config struct {
	// APIKey is the API key used for authentication.
	APIKey string

	// Credentials holds an AK/SK pair for request signing.
	// When both Credentials and APIKey are configured, Credentials takes
	// priority (the Authorization header is set first; API Key is skipped).
	Credentials *auth.Credentials

	// BaseURL is the sandbox API endpoint. Defaults to baseconf.DefaultBaseURL
	// if empty.
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
// At least one authentication method must be available: either an API Key
// (Config.APIKey) or AK/SK credentials (Config.Credentials). New panics if
// neither is configured.
func New(__xgo_optional_conf *Config) *Client {
	conf := __xgo_optional_conf
	if conf == nil {
		conf = &Config{}
	}
	if conf.BaseURL == "" {
		conf.BaseURL = baseconf.DefaultBaseURL
	}
	if conf.HTTPClient == nil {
		conf.HTTPClient = http.DefaultClient
	}

	// At least one auth method is required.
	if conf.APIKey == "" && conf.Credentials == nil {
		panic("sandbox: at least one authentication method is required (API Key or AK/SK credentials)")
	}

	opts := []apis.ClientOption{
		apis.WithHTTPClient(conf.HTTPClient),
		apis.WithRequestEditorFn(reqidEditor()),
	}

	// Register editors in priority order: credentials first, then API Key.
	// When credentials are present, the Authorization header is set before
	// apiKeyEditor runs, causing it to skip the X-API-Key header.
	if conf.Credentials != nil {
		opts = append(opts, apis.WithRequestEditorFn(credentialsEditor(conf.Credentials)))
	}
	if conf.APIKey != "" {
		opts = append(opts, apis.WithRequestEditorFn(apiKeyEditor(conf.APIKey)))
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

// credentialsEditor returns a RequestEditorFn that signs the request with
// AK/SK V2 and sets the Authorization: Sufy <token> header.
func credentialsEditor(cred *auth.Credentials) apis.RequestEditorFn {
	return func(ctx context.Context, req *http.Request) error {
		token, err := cred.SignRequestV2(req)
		if err != nil {
			return fmt.Errorf("sign request: %w", err)
		}
		req.Header.Set("Authorization", auth.AuthorizationPrefix+token)
		return nil
	}
}

// apiKeyEditor returns a RequestEditorFn that injects the X-API-Key header.
// If the Authorization header is already set (e.g. by credentialsEditor), the
// API Key is skipped to avoid sending conflicting credentials.
func apiKeyEditor(apiKey string) apis.RequestEditorFn {
	return func(ctx context.Context, req *http.Request) error {
		if req.Header.Get("Authorization") != "" {
			return nil
		}
		req.Header.Set("X-API-Key", apiKey)
		return nil
	}
}
