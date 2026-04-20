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

package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sufy-dev/sufy/sandbox"
)

// WatchSandboxRunning closes done when the sandbox stops responding. Used by
// `logs --follow` and `metrics --follow` to terminate the streaming loop.
func WatchSandboxRunning(ctx context.Context, sb *sandbox.Sandbox, done chan<- struct{}) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			running, _ := sb.IsRunning(ctx)
			if !running {
				close(done)
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// PrintLogEntries renders sandbox logs to stdout, applying the minimum-level
// and logger-prefix filters. Falls back to raw LogLine rendering when the
// structured LogEntries field is empty.
func PrintLogEntries(logs *sandbox.SandboxLogs, level string, loggerPrefixes []string) {
	if len(logs.LogEntries) > 0 {
		for _, entry := range logs.LogEntries {
			if !IsLogLevelIncluded(string(entry.Level), level) {
				continue
			}
			logger := entry.Fields["logger"]
			if len(loggerPrefixes) > 0 && !MatchesLoggerPrefix(logger, loggerPrefixes) {
				continue
			}
			cleanLogger := CleanLoggerName(logger)
			userFields := StripInternalFields(entry.Fields)

			parts := []string{
				fmt.Sprintf("[%s]", entry.Timestamp.Format(time.RFC3339)),
				LogLevelBadge(string(entry.Level)),
			}
			if cleanLogger != "" {
				parts = append(parts, fmt.Sprintf("[%s]", cleanLogger))
			}
			parts = append(parts, entry.Message)

			if len(userFields) > 0 {
				var fieldParts []string
				for k, v := range userFields {
					if k == "logger" {
						continue
					}
					fieldParts = append(fieldParts, fmt.Sprintf("%s=%s", k, v))
				}
				if len(fieldParts) > 0 {
					parts = append(parts, strings.Join(fieldParts, " "))
				}
			}

			fmt.Println(strings.Join(parts, " "))
		}
		return
	}
	for _, l := range logs.Logs {
		fmt.Printf("[%s] %s\n", l.Timestamp.Format(time.RFC3339), l.Line)
	}
}
