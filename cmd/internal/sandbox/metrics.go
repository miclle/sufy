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

// MetricsInfo holds the flags passed to Metrics.
type MetricsInfo struct {
	SandboxID string
	Format    string
	Follow    bool
}

// Metrics prints resource-usage metrics for the given sandbox.
func Metrics(info MetricsInfo) {
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

	sandboxDone := make(chan struct{})
	if info.Follow {
		go WatchSandboxRunning(ctx, sb, sandboxDone)
	}

	var lastTimestamp *time.Time

	for {
		params := &sdk.GetMetricsParams{}
		if lastTimestamp != nil {
			start := lastTimestamp.Unix()
			params.Start = &start
		}

		metrics, mErr := sb.GetMetrics(ctx, params)
		if mErr != nil {
			PrintError("get sandbox metrics failed: %v", mErr)
			return
		}

		if info.Format == FormatJSON {
			if !info.Follow {
				PrintJSON(metrics)
				return
			}
			if len(metrics) > 0 {
				PrintJSON(metrics)
			}
		} else {
			if !info.Follow && len(metrics) == 0 {
				fmt.Println("No metrics available")
				return
			}
			for _, m := range metrics {
				if lastTimestamp != nil && !m.Timestamp.After(*lastTimestamp) {
					continue
				}
				fmt.Printf("[%s]  %s  %.1f%% / %d Cores | %s  %s / %s | %s  %s / %s\n",
					m.Timestamp.Format(time.RFC3339),
					ColorInfo.Sprint("CPU:"),
					m.CPUUsedPct,
					m.CPUCount,
					ColorInfo.Sprint("Memory:"),
					FormatBytes(m.MemUsed),
					FormatBytes(m.MemTotal),
					ColorInfo.Sprint("Disk:"),
					FormatBytes(m.DiskUsed),
					FormatBytes(m.DiskTotal),
				)
				ts := m.Timestamp
				lastTimestamp = &ts
			}
		}

		if !info.Follow {
			return
		}

		select {
		case <-sandboxDone:
			if info.Format != FormatJSON {
				fmt.Println("\nStopped printing metrics — sandbox is closed")
			}
			return
		case <-ctx.Done():
			return
		default:
		}

		time.Sleep(400 * time.Millisecond)
	}
}
