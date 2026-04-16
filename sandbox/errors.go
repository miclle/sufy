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
	"encoding/json"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
)

// APIError represents an unexpected HTTP response returned by the sandbox API.
type APIError struct {
	StatusCode int
	Body       []byte

	// Reqid is the request ID extracted from the X-Reqid response header. It is
	// used to correlate client-side issues with server-side logs.
	Reqid string
	// Code is the error code parsed from the JSON response body, if present.
	Code string
	// Message is the error message parsed from the JSON response body.
	Message string
}

// Error implements the error interface.
func (e *APIError) Error() string {
	prefix := fmt.Sprintf("api error: status %d", e.StatusCode)
	if e.Reqid != "" {
		prefix += ", reqid: " + e.Reqid
	}
	if e.Message != "" {
		return prefix + ": " + e.Message
	}
	if len(e.Body) > 0 {
		return prefix + ", body: " + string(e.Body)
	}
	return prefix
}

// newAPIError builds an APIError from an HTTP response, extracting the
// X-Reqid header and best-effort parsing a JSON body.
func newAPIError(resp *http.Response, body []byte) *APIError {
	e := &APIError{
		StatusCode: resp.StatusCode,
		Body:       body,
		Reqid:      resp.Header.Get("X-Reqid"),
	}
	e.Code, e.Message = parseAPIErrorBody(body)
	return e
}

// parseAPIErrorBody tries to extract the "code" and "message" fields from a
// JSON body. Failures are silently ignored.
func parseAPIErrorBody(body []byte) (code, message string) {
	if len(body) == 0 {
		return "", ""
	}
	var parsed struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &parsed) == nil {
		return parsed.Code, parsed.Message
	}
	return "", ""
}

// isNotFoundError reports whether the error represents a "not found" condition,
// whether it comes from the REST API or a ConnectRPC call.
func isNotFoundError(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusNotFound
	}
	if connect.CodeOf(err) == connect.CodeNotFound {
		return true
	}
	return false
}
