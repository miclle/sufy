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

import "strings"

// logLevelOrder orders log levels for hierarchical filtering. Higher numbers
// include lower ones (info includes warn and error, etc.).
var logLevelOrder = map[string]int{
	"debug": 0,
	"info":  1,
	"warn":  2,
	"error": 3,
}

// IsLogLevelIncluded reports whether an entry's level should be included given
// the configured minimum level. Unknown levels are included by default.
func IsLogLevelIncluded(entryLevel, minLevel string) bool {
	if minLevel == "" {
		return true
	}
	e, ok1 := logLevelOrder[strings.ToLower(entryLevel)]
	m, ok2 := logLevelOrder[strings.ToLower(minLevel)]
	if !ok1 || !ok2 {
		return true
	}
	return e >= m
}

// MatchesLoggerPrefix reports whether the logger name starts with any of the
// given prefixes. Used to filter by structured logger.
func MatchesLoggerPrefix(logger string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(logger, p) {
			return true
		}
	}
	return false
}

// InternalLogFields names log fields that should be hidden from user-facing
// output (e.g. trace/span IDs).
var InternalLogFields = map[string]bool{
	"traceID":     true,
	"instanceID":  true,
	"teamID":      true,
	"source":      true,
	"service":     true,
	"envID":       true,
	"sandboxID":   true,
	"source_type": true,
}

// StripInternalFields returns a copy of fields with internal keys removed. If
// no user-visible keys remain, returns nil.
func StripInternalFields(fields map[string]string) map[string]string {
	if len(fields) == 0 {
		return nil
	}
	out := make(map[string]string)
	for k, v := range fields {
		if !InternalLogFields[k] {
			out[k] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// CleanLoggerName strips the "Svc" suffix common to service-logger names.
func CleanLoggerName(logger string) string {
	return strings.TrimSuffix(logger, "Svc")
}
