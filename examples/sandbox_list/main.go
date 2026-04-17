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
	"time"

	"github.com/sufy-dev/sufy/examples/internal/exampleutil"
	"github.com/sufy-dev/sufy/sandbox"
)

func main() {
	c := exampleutil.MustNewClient()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// List all running sandboxes.
	sandboxes, err := c.List(ctx, nil)
	if err != nil {
		log.Fatalf("List failed: %v", err)
	}

	fmt.Printf("found %d running sandboxes:\n", len(sandboxes))
	for _, sb := range sandboxes {
		fmt.Printf("  - %s (template: %s)\n", sb.SandboxID, sb.TemplateID)
	}

	// Filter sandboxes by state.
	fmt.Println("\n=== filter by state ===")
	states := []sandbox.SandboxState{sandbox.StateRunning}
	filtered, err := c.List(ctx, &sandbox.ListParams{
		State: &states,
	})
	if err != nil {
		log.Fatalf("List failed: %v", err)
	}
	fmt.Printf("found %d running sandboxes\n", len(filtered))

	// Batch fetch sandbox metrics.
	if len(sandboxes) > 0 {
		fmt.Println("\n=== batch sandbox metrics ===")
		ids := make([]string, 0, len(sandboxes))
		for _, sb := range sandboxes {
			ids = append(ids, sb.SandboxID)
		}
		metrics, err := c.GetSandboxesMetrics(ctx, &sandbox.GetSandboxesMetricsParams{
			SandboxIds: ids,
		})
		if err != nil {
			fmt.Printf("GetSandboxesMetrics failed: %v\n", err)
		} else {
			fmt.Printf("received metrics for %d sandboxes\n", len(metrics.Sandboxes))
		}
	}
}
