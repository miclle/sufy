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

// Package auth provides AK/SK credential management and request signing for
// the Sufy platform. It implements the V2 signing algorithm compatible with
// the Sufy API gateway.
package auth

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/textproto"
	"sort"
	"strings"
)

const (
	// AuthorizationPrefix is the prefix used in the Authorization header for
	// Sufy V2-signed requests.
	AuthorizationPrefix = "Sufy "
)

// Content types that trigger body inclusion in the signing string.
const (
	contentTypeForm = "application/x-www-form-urlencoded"
	contentTypeJSON = "application/json"
)

// Credentials holds an Access Key / Secret Key pair used for request signing.
type Credentials struct {
	// AccessKey is the public key identifier.
	AccessKey string

	// SecretKey is the secret used for HMAC signing.
	SecretKey []byte
}

// New creates a Credentials from an access key and secret key.
func New(accessKey, secretKey string) *Credentials {
	return &Credentials{AccessKey: accessKey, SecretKey: []byte(secretKey)}
}

// Sign computes an HMAC-SHA1 signature over data and returns
// "<AccessKey>:<base64url-encoded-signature>".
func (c *Credentials) Sign(data []byte) string {
	h := hmac.New(sha1.New, c.SecretKey)
	h.Write(data)
	sig := base64.URLEncoding.EncodeToString(h.Sum(nil))
	return fmt.Sprintf("%s:%s", c.AccessKey, sig)
}

// SignRequestV2 signs an HTTP request using the V2 algorithm. The returned
// token should be used as: Authorization: Sufy <token>.
func (c *Credentials) SignRequestV2(req *http.Request) (string, error) {
	data, err := collectDataV2(req)
	if err != nil {
		return "", err
	}
	return c.Sign(data), nil
}

// collectDataV2 builds the canonical signing string for V2:
//
//	METHOD path[?query]\n
//	Host: host\n
//	Content-Type: content-type\n
//	[X-Sufy-*: value\n ...]
//	\n
//	[body — if Content-Type is form or json]
func collectDataV2(req *http.Request) ([]byte, error) {
	u := req.URL

	// Method and path.
	s := fmt.Sprintf("%s %s", req.Method, u.Path)
	if u.RawQuery != "" {
		s += "?" + u.RawQuery
	}

	// Host header.
	s += "\nHost: " + req.Host + "\n"

	// Content-Type header (default to form if missing).
	contentType := req.Header.Get("Content-Type")
	if contentType == "" {
		contentType = contentTypeForm
		req.Header.Set("Content-Type", contentType)
	}
	s += fmt.Sprintf("Content-Type: %s\n", contentType)

	// Sorted X-Sufy-* headers.
	var xHeaders xSufyHeaders
	for name := range req.Header {
		if len(name) > len("X-Sufy-") && strings.HasPrefix(name, "X-Sufy-") {
			xHeaders = append(xHeaders, xSufyHeaderItem{
				HeaderName:  textproto.CanonicalMIMEHeaderKey(name),
				HeaderValue: req.Header.Get(name),
			})
		}
	}
	if len(xHeaders) > 0 {
		sort.Sort(xHeaders)
		for _, h := range xHeaders {
			s += fmt.Sprintf("%s: %s\n", h.HeaderName, h.HeaderValue)
		}
	}

	// Trailing newline (separator before body).
	s += "\n"

	data := []byte(s)

	// Include body for form/json content types.
	if req.Body != nil && (contentType == contentTypeForm || contentType == contentTypeJSON) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		req.Body = io.NopCloser(bytes.NewReader(body))
		data = append(data, body...)
	}

	return data, nil
}

// xSufyHeaderItem is a name/value pair for X-Sufy-* header sorting.
type xSufyHeaderItem struct {
	HeaderName  string
	HeaderValue string
}

// xSufyHeaders implements sort.Interface for X-Sufy-* headers.
type xSufyHeaders []xSufyHeaderItem

func (h xSufyHeaders) Len() int      { return len(h) }
func (h xSufyHeaders) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h xSufyHeaders) Less(i, j int) bool {
	if h[i].HeaderName != h[j].HeaderName {
		return h[i].HeaderName < h[j].HeaderName
	}
	return h[i].HeaderValue < h[j].HeaderValue
}
