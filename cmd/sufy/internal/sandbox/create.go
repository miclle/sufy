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
	"strings"

	sdk "github.com/sufy-dev/sufy/sandbox"
)

// CreateInfo holds the flags passed to Create.
type CreateInfo struct {
	TemplateID       string
	Timeout          int32
	Metadata         string
	Detach           bool
	EnvVars          []string
	AutoPause        bool
	InjectionRuleIDs []string
	InlineInjections []string
}

// Create creates a sandbox and, unless Detach is set, opens a PTY session.
// When the terminal session ends the sandbox is killed.
func Create(info CreateInfo) {
	if info.TemplateID == "" {
		PrintError("template ID is required")
		return
	}

	client := MustNewSandboxClient()
	ctx := context.Background()

	params := sdk.CreateParams{TemplateID: info.TemplateID}
	if info.Timeout > 0 {
		params.Timeout = &info.Timeout
	}
	if info.Metadata != "" {
		meta := sdk.Metadata(ParseKeyValueMap(info.Metadata))
		params.Metadata = &meta
	}
	if len(info.EnvVars) > 0 {
		env := ParseEnvPairs(info.EnvVars)
		if len(env) > 0 {
			params.EnvVars = &env
		}
	}
	if info.AutoPause {
		autoPause := true
		params.AutoPause = &autoPause
	}

	injections, err := BuildSandboxInjections(info.InjectionRuleIDs, info.InlineInjections)
	if err != nil {
		PrintError("%v", err)
		return
	}
	if len(injections) > 0 {
		params.Injections = &injections
	}

	fmt.Printf("Creating sandbox from template %s...\n", info.TemplateID)
	sb, _, err := client.CreateAndWait(ctx, params)
	if err != nil {
		PrintError("create sandbox failed: %v", err)
		return
	}

	if info.Detach {
		PrintSuccess("Sandbox %s created", sb.ID())
		fmt.Printf("Sandbox ID:   %s\n", sb.ID())
		fmt.Printf("Template ID:  %s\n", sb.TemplateID())
		fmt.Println()
		fmt.Printf("Connect:  sufy sandbox connect %s\n", sb.ID())
		fmt.Printf("Exec:     sufy sandbox exec %s -- <command>\n", sb.ID())
		fmt.Printf("Kill:     sufy sandbox kill %s\n", sb.ID())
		return
	}

	PrintSuccess("Sandbox %s created, connecting...", sb.ID())

	defer func() {
		fmt.Printf("\nKilling sandbox %s...\n", sb.ID())
		if kErr := sb.Kill(context.Background()); kErr != nil {
			if !strings.Contains(kErr.Error(), "404") {
				PrintWarn("kill sandbox failed: %v", kErr)
			}
		}
	}()

	RunTerminalSession(ctx, sb)
}
