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

// Package exampleutil provides shared helpers for sandbox examples.
package exampleutil

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/sufy-dev/sufy/auth"
	"github.com/sufy-dev/sufy/sandbox"
)

// authMode is the -auth flag value, registered in init().
var authMode string

func init() {
	flag.StringVar(&authMode, "auth", "", `authentication mode: "apikey", "aksk", or "" (default: both, credentials take priority)`)
}

// MustNewClient parses flags (if not already parsed) and creates a sandbox
// client based on the -auth flag:
//   - "apikey": use only SUFY_API_KEY
//   - "aksk":   use only SUFY_ACCESS_KEY / SUFY_SECRET_KEY
//   - "":       use both (credentials take priority when available)
func MustNewClient() *sandbox.Client {
	if !flag.Parsed() {
		flag.Parse()
	}

	var (
		apiKey string
		cred   *auth.Credentials
	)

	switch authMode {
	case "apikey":
		apiKey = requireEnv("SUFY_API_KEY")
		fmt.Println("[auth] using API Key")

	case "aksk":
		ak := requireEnv("SUFY_ACCESS_KEY")
		sk := requireEnv("SUFY_SECRET_KEY")
		cred = auth.New(ak, sk)
		fmt.Println("[auth] using AK/SK credentials")

	case "":
		apiKey = os.Getenv("SUFY_API_KEY")
		if ak, sk := os.Getenv("SUFY_ACCESS_KEY"), os.Getenv("SUFY_SECRET_KEY"); ak != "" && sk != "" {
			cred = auth.New(ak, sk)
		}
		if apiKey == "" && cred == nil {
			log.Fatal("SUFY_API_KEY or SUFY_ACCESS_KEY/SUFY_SECRET_KEY environment variables are required")
		}
		if cred != nil {
			fmt.Println("[auth] using AK/SK credentials (with API Key fallback)")
		} else {
			fmt.Println("[auth] using API Key")
		}

	default:
		log.Fatalf("unknown -auth mode: %q (use \"apikey\", \"aksk\", or omit)", authMode)
	}

	return sandbox.New(&sandbox.Config{
		APIKey:      apiKey,
		Credentials: cred,
		BaseURL:     os.Getenv("SUFY_BASE_URL"),
	})
}

// requireEnv returns the value of the named environment variable or exits.
func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("%s environment variable is required (auth mode: %s)", key, authMode)
	}
	return v
}
