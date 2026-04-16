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

	// 1. List templates and pick the first usable one.
	templates, err := c.ListTemplates(ctx, nil)
	if err != nil {
		log.Fatalf("ListTemplates failed: %v", err)
	}
	if len(templates) == 0 {
		log.Fatal("no templates available")
	}

	var templateID string
	for _, tmpl := range templates {
		if tmpl.BuildStatus == sandbox.BuildStatusReady || tmpl.BuildStatus == sandbox.BuildStatusUploaded {
			templateID = tmpl.TemplateID
			break
		}
	}
	if templateID == "" {
		log.Fatal("no successfully built template available")
	}
	fmt.Printf("using template: %s\n", templateID)

	// 2. Create the sandbox and wait for it to become ready.
	timeout := int32(120)
	sb, info, err := c.CreateAndWait(ctx, sandbox.CreateParams{
		TemplateID: templateID,
		Timeout:    &timeout,
	}, sandbox.WithPollInterval(2*time.Second))
	if err != nil {
		log.Fatalf("CreateAndWait failed: %v", err)
	}
	fmt.Printf("sandbox ready: %s (state: %s)\n", sb.ID(), info.State)

	// 3. Sandbox details (use the info returned by CreateAndWait).
	fmt.Printf("CPU: %d cores, memory: %d MB\n", info.CPUCount, info.MemoryMB)

	// 4. Check running status.
	running, err := sb.IsRunning(ctx)
	if err != nil {
		log.Fatalf("IsRunning failed: %v", err)
	}
	fmt.Printf("running: %v\n", running)

	// 5. Update the timeout.
	if err := sb.SetTimeout(ctx, 5*time.Minute); err != nil {
		log.Fatalf("SetTimeout failed: %v", err)
	}
	fmt.Println("timeout updated to 5 minutes")

	// 6. Extend the lifetime (Refresh).
	duration := 300
	if err := sb.Refresh(ctx, sandbox.RefreshParams{
		Duration: &duration,
	}); err != nil {
		log.Fatalf("Refresh failed: %v", err)
	}
	fmt.Println("sandbox lifetime extended by 300 seconds")

	// 7. Terminate the sandbox.
	if err := sb.Kill(ctx); err != nil {
		log.Fatalf("Kill failed: %v", err)
	}
	fmt.Printf("sandbox %s terminated\n", sb.ID())
}
