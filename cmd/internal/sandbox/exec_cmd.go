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
	"strings"

	sdk "github.com/sufy-dev/sufy/sandbox"
)

// ExecInfo holds the flags passed to Exec.
type ExecInfo struct {
	SandboxID  string
	Command    []string
	Background bool
	Cwd        string
	User       string
	EnvVars    []string
}

// Exec runs a command inside an existing sandbox.
func Exec(info ExecInfo) {
	if info.SandboxID == "" {
		PrintError("sandbox ID is required")
		return
	}
	if len(info.Command) == 0 {
		PrintError("command is required")
		return
	}

	client := MustNewSandboxClient()
	ctx := context.Background()

	sb, err := client.Connect(ctx, info.SandboxID, sdk.ConnectParams{Timeout: 10})
	if err != nil {
		PrintError("connect to sandbox %s failed: %v", info.SandboxID, err)
		return
	}

	cmd := strings.Join(info.Command, " ")

	var opts []sdk.CommandOption
	if info.Cwd != "" {
		opts = append(opts, sdk.WithCwd(info.Cwd))
	}
	if info.User != "" {
		opts = append(opts, sdk.WithCommandUser(info.User))
	}
	if env := ParseEnvPairs(info.EnvVars); len(env) > 0 {
		opts = append(opts, sdk.WithEnvs(env))
	}
	if IsPipedStdin() {
		opts = append(opts, sdk.WithStdin())
	}

	if info.Background {
		ExecBackground(ctx, sb, cmd, opts)
	} else {
		ExecForeground(ctx, sb, cmd, opts)
	}
}
