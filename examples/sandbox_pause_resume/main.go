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

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// 1. Create the sandbox.
	templateID := "base"
	timeout := int32(120)
	sb, info, err := c.CreateAndWait(ctx, sandbox.CreateParams{
		TemplateID: templateID,
		Timeout:    &timeout,
	}, sandbox.WithPollInterval(2*time.Second))
	if err != nil {
		log.Fatalf("CreateAndWait failed: %v", err)
	}
	fmt.Printf("sandbox ready: %s (state: %s)\n", sb.ID(), info.State)

	// 2. Pause the sandbox.
	fmt.Println("pausing sandbox...")
	if err := sb.Pause(ctx); err != nil {
		log.Fatalf("Pause failed: %v", err)
	}
	fmt.Println("sandbox paused")

	// 3. Confirm the state.
	detail, err := sb.GetInfo(ctx)
	if err != nil {
		log.Fatalf("GetInfo failed: %v", err)
	}
	fmt.Printf("current state: %s\n", detail.State)

	// 4. Resume the sandbox via Connect.
	fmt.Println("resuming sandbox...")
	resumed, err := c.Connect(ctx, sb.ID(), sandbox.ConnectParams{
		Timeout: 120,
	})
	if err != nil {
		log.Fatalf("Connect failed: %v", err)
	}

	// Wait for the resumed sandbox to become ready.
	readyInfo, err := resumed.WaitForReady(ctx, sandbox.WithPollInterval(2*time.Second))
	if err != nil {
		log.Fatalf("WaitForReady failed: %v", err)
	}
	fmt.Printf("sandbox resumed: %s (state: %s)\n", resumed.ID(), readyInfo.State)

	// 5. Cleanup.
	if err := resumed.Kill(ctx); err != nil {
		log.Fatalf("Kill failed: %v", err)
	}
	fmt.Println("sandbox terminated")
}
