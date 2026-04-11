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
	"time"

	"github.com/sufy-dev/sufy/baseconf"
)

// -----------------------------------------------------------------------------

// Config is the configuration for the sandbox client.
type Config struct {
	APIKey  string // API key for authentication
	BaseURL string // Base URL for the SUFY API
}

// Client is the client for sandbox.
type Client struct {
	apiKey  string
	baseURL string
}

// New creates a new sandbox client.
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

// List lists the sandboxes.
func List(ctx context.Context, __xgo_optional_opts *ListOptions) (items []*Sandbox, pagination ListPagination, err error) {
	return New(nil).List(ctx, __xgo_optional_opts)
}

// -----------------------------------------------------------------------------

// Sandbox represents a sandbox instance.
type Sandbox struct {
	ID string // Unique identifier for the sandbox
}

// -----------------------------------------------------------------------------

// ListPagination contains pagination information for listing sandboxes.
type ListPagination struct {
	Total int
	Next  string
}

// ListOptions contains options for listing sandboxes.
type ListOptions struct {
	// Labels to filter sandboxes. Only sandboxes that have all the specified labels
	// will be returned.
	Labels map[string]string

	// Limit the number of sandboxes returned. Default is 20, maximum is 100.
	Limit int

	// Pagination token for fetching the next page.
	From string

	// Time range for filtering. Don't set both From and Since/Until at the same time.
	Since time.Time
	Until time.Time
}

// List lists the sandboxes.
func (c *Client) List(ctx context.Context, __xgo_optional_opts *ListOptions) (items []*Sandbox, pagination ListPagination, err error) {
	panic("todo")
}

// -----------------------------------------------------------------------------
