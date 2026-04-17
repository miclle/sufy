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

	"github.com/sufy-dev/sufy/examples/internal/exampleutil"
	"github.com/sufy-dev/sufy/sandbox"
)

func main() {
	c := exampleutil.MustNewClient()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Connect to an existing sandbox. Replace with a real sandbox ID.
	sandboxID := os.Getenv("SANDBOX_ID")
	if sandboxID == "" {
		// Without a sandbox ID, pick the first sandbox from the list.
		sandboxes, err := c.List(ctx, nil)
		if err != nil {
			log.Fatalf("List failed: %v", err)
		}
		if len(sandboxes) > 0 {
			sandboxID = sandboxes[0].SandboxID
			fmt.Printf("found running sandbox: %s\n", sandboxID)
		}
	}

	// If no sandbox is available, create one for the demo.
	if sandboxID == "" {
		fmt.Println("no running sandbox found; creating one for the demo...")

		templates, err := c.ListTemplates(ctx, nil)
		if err != nil {
			log.Fatalf("ListTemplates failed: %v", err)
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

		timeout := int32(60)
		created, _, err := c.CreateAndWait(ctx, sandbox.CreateParams{
			TemplateID: templateID,
			Timeout:    &timeout,
		}, sandbox.WithPollInterval(2*time.Second))
		if err != nil {
			log.Fatalf("CreateAndWait failed: %v", err)
		}
		sandboxID = created.ID()
		fmt.Printf("sandbox created: %s\n", sandboxID)

		defer func() {
			killCtx, killCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer killCancel()
			if err := created.Kill(killCtx); err != nil {
				log.Printf("Kill failed: %v", err)
			} else {
				fmt.Printf("sandbox %s terminated\n", sandboxID)
			}
		}()
	}

	timeout := int32(300)
	sb, err := c.Connect(ctx, sandboxID, sandbox.ConnectParams{
		Timeout: timeout,
	})
	if err != nil {
		log.Fatalf("Connect failed: %v", err)
	}

	fmt.Printf("connected to sandbox: %s (template: %s)\n", sb.ID(), sb.TemplateID())

	// Fetch sandbox details.
	info, err := sb.GetInfo(ctx)
	if err != nil {
		log.Fatalf("GetInfo failed: %v", err)
	}
	fmt.Printf("state: %s, CPU: %d cores, memory: %d MB\n", info.State, info.CPUCount, info.MemoryMB)
}
