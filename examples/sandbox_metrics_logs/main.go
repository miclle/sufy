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

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/sufy-dev/sufy/sandbox"
)

func main() {
	apiKey := os.Getenv("SUFY_API_KEY")
	if apiKey == "" {
		log.Fatal("SUFY_API_KEY environment variable is required")
	}

	c := sandbox.New(&sandbox.Config{
		APIKey:  apiKey,
		BaseURL: os.Getenv("SUFY_BASE_URL"),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// 1. Create the sandbox.
	templateID := "base"
	timeout := int32(120)
	sb, _, err := c.CreateAndWait(ctx, sandbox.CreateParams{
		TemplateID: templateID,
		Timeout:    &timeout,
	}, sandbox.WithPollInterval(2*time.Second))
	if err != nil {
		log.Fatalf("CreateAndWait failed: %v", err)
	}
	fmt.Printf("sandbox ready: %s\n", sb.ID())
	defer func() {
		_ = sb.Kill(context.Background())
		fmt.Println("sandbox terminated")
	}()

	// Wait a moment so the sandbox has metrics to report.
	fmt.Println("waiting 5 seconds to collect metrics...")
	time.Sleep(5 * time.Second)

	// 2. Fetch metrics.
	metrics, err := sb.GetMetrics(ctx, nil)
	if err != nil {
		log.Fatalf("GetMetrics failed: %v", err)
	}
	fmt.Printf("metrics (%d entries):\n", len(metrics))
	for _, m := range metrics {
		fmt.Printf("  - time: %s, CPU: %.1f%%, memory: %d/%d bytes\n",
			m.Timestamp.Format(time.RFC3339), m.CPUUsedPct, m.MemUsed, m.MemTotal)
	}

	// 3. Fetch logs.
	logs, err := sb.GetLogs(ctx, nil)
	if err != nil {
		fmt.Printf("\nGetLogs failed (server may not support it yet): %v\n", err)
	} else {
		fmt.Printf("\nlogs (%d entries):\n", len(logs.Logs))
		for _, entry := range logs.Logs {
			fmt.Printf("  [%s] %s\n", entry.Timestamp.Format(time.RFC3339), entry.Line)
		}
	}
}
