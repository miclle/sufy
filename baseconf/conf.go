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
	// DefaultBaseURL is the default Sufy sandbox API endpoint.
	DefaultBaseURL = "https://api.sufy.com"
)

// RequireAPIKey returns the API key from the provided value, falling back to
// the SUFY_API_KEY environment variable. Returns an empty string if neither
// is set — the caller decides whether API Key is mandatory.
func RequireAPIKey(apiKey string) string {
	if apiKey == "" {
		apiKey = os.Getenv("SUFY_API_KEY")
	}
	return apiKey
}

// RequireCredentials returns the access key and secret key from the provided
// values, falling back to the SUFY_ACCESS_KEY and SUFY_SECRET_KEY environment
// variables. Returns empty strings if not set — the caller decides whether
// credentials are mandatory.
func RequireCredentials(accessKey, secretKey string) (string, string) {
	if accessKey == "" {
		accessKey = os.Getenv("SUFY_ACCESS_KEY")
	}
	if secretKey == "" {
		secretKey = os.Getenv("SUFY_SECRET_KEY")
	}
	return accessKey, secretKey
}

// RequireBaseURL returns the base URL from the provided value, falling back to
// the SUFY_BASE_URL environment variable, then to DefaultBaseURL.
func RequireBaseURL(baseURL string) string {
	if baseURL == "" {
		baseURL = os.Getenv("SUFY_BASE_URL")
		if baseURL == "" {
			baseURL = DefaultBaseURL
		}
	}
	return baseURL
}
