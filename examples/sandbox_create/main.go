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

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create a sandbox. Replace templateID with an actually available template.
	templateID := "base"
	timeout := int32(300)
	sb, err := c.Create(ctx, sandbox.CreateParams{
		TemplateID: templateID,
		Timeout:    &timeout,
	})
	if err != nil {
		log.Fatalf("Create failed: %v", err)
	}

	fmt.Printf("sandbox created: %s (template: %s)\n", sb.ID(), sb.TemplateID())

	// Tear the sandbox down to release resources.
	killCtx, killCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer killCancel()
	if err := sb.Kill(killCtx); err != nil {
		log.Printf("Kill failed: %v", err)
	} else {
		fmt.Printf("sandbox %s terminated\n", sb.ID())
	}
}
