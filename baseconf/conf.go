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
	"os"
)

const (
	DefaultBaseURL = "https://api.sufy.com"
)

// -----------------------------------------------------------------------------

// RequireAPIKey returns the API key from the environment variable or the provided
// value. If the API key is not provided, it panics.
func RequireAPIKey(apiKey string) string {
	if apiKey == "" {
		apiKey = os.Getenv("SUFY_API_KEY")
		if apiKey == "" {
			panic("SUFY API key is required")
		}
	}
	return apiKey
}

// RequireBaseURL returns the base URL from the environment variable or the
// provided value. If the base URL is not provided, it returns the default
// base URL.
func RequireBaseURL(baseURL string) string {
	if baseURL == "" {
		baseURL = os.Getenv("SUFY_BASE_URL")
		if baseURL == "" {
			baseURL = DefaultBaseURL
		}
	}
	return baseURL
}

// -----------------------------------------------------------------------------
