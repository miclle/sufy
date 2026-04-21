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
	"os"
	"time"

	sdk "github.com/sufy-dev/sufy/sandbox"
)

// LogsInfo holds the flags passed to Logs.
type LogsInfo struct {
	SandboxID string
	Level     string
	Limit     int32
	Format    string
	Follow    bool
	Loggers   string
}

// Logs streams or prints sandbox logs.
func Logs(info LogsInfo) {
	if info.SandboxID == "" {
		PrintError("sandbox ID is required")
		return
	}

	client := MustNewSandboxClient()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signalNotify(sigCh)
	go func() {
		<-sigCh
		cancel()
	}()
	defer signalStop(sigCh)

	sb, err := client.Connect(ctx, info.SandboxID, sdk.ConnectParams{Timeout: 10})
	if err != nil {
		PrintError("connect to sandbox %s failed: %v", info.SandboxID, err)
		return
	}

	level := info.Level
	if level == "" {
		level = "info"
	}
	loggerPrefixes := SplitCSV(info.Loggers)

	sandboxDone := make(chan struct{})
	if info.Follow {
		go WatchSandboxRunning(ctx, sb, sandboxDone)
	}

	var start *int64
	for {
		params := &sdk.GetLogsParams{Start: start}
		if info.Limit > 0 && start == nil {
			params.Limit = &info.Limit
		}

		logs, lErr := sb.GetLogs(ctx, params)
		if lErr != nil {
			PrintError("get sandbox logs failed: %v", lErr)
			return
		}

		if info.Format == FormatJSON {
			if !info.Follow {
				PrintJSON(logs)
				return
			}
			if len(logs.Logs) > 0 || len(logs.LogEntries) > 0 {
				PrintJSON(logs)
			}
		} else {
			PrintLogEntries(logs, level, loggerPrefixes)
		}

		if !info.Follow {
			if info.Format != FormatJSON && len(logs.Logs) == 0 && len(logs.LogEntries) == 0 {
				fmt.Println("No logs found")
			}
			return
		}

		if len(logs.Logs) > 0 {
			ts := logs.Logs[len(logs.Logs)-1].Timestamp.UnixMilli() + 1
			start = &ts
		} else if len(logs.LogEntries) > 0 {
			ts := logs.LogEntries[len(logs.LogEntries)-1].Timestamp.UnixMilli() + 1
			start = &ts
		}

		select {
		case <-sandboxDone:
			if info.Format != FormatJSON {
				fmt.Println("\nStopped printing logs — sandbox is closed")
			}
			return
		case <-ctx.Done():
			return
		default:
		}

		time.Sleep(400 * time.Millisecond)
	}
}
