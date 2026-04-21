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
	"sync"

	sdk "github.com/sufy-dev/sufy/sandbox"
)

// PauseInfo holds the flags passed to PauseBatch.
type PauseInfo struct {
	SandboxIDs []string
	All        bool
	State      string
	Metadata   string
}

// Pause pauses the given sandbox so it can be resumed later.
func Pause(sandboxID string) {
	PauseBatch(PauseInfo{SandboxIDs: []string{sandboxID}})
}

// PauseBatch pauses one or more sandboxes, or all sandboxes matching filters.
func PauseBatch(info PauseInfo) {
	if !info.All && len(info.SandboxIDs) == 0 {
		PrintError("sandbox ID is required, or use --all")
		return
	}

	client := MustNewSandboxClient()
	ctx := context.Background()

	ids := info.SandboxIDs
	if info.All {
		stateFilter := info.State
		if stateFilter == "" {
			stateFilter = DefaultState
		}
		listed, err := listSandboxIDs(ctx, client, stateFilter, info.Metadata)
		if err != nil {
			PrintError("list sandboxes failed: %v", err)
			return
		}
		if len(listed) == 0 {
			fmt.Println("No sandboxes found matching the filter.")
			return
		}
		ids = listed
	}

	var wg sync.WaitGroup
	for _, id := range ids {
		wg.Add(1)
		go func(sandboxID string) {
			defer wg.Done()
			sb, err := client.Connect(ctx, sandboxID, sdk.ConnectParams{Timeout: 10})
			if err != nil {
				PrintError("connect sandbox %s failed: %v", sandboxID, err)
				return
			}
			if err := sb.Pause(ctx); err != nil {
				PrintError("pause sandbox %s failed: %v", sandboxID, err)
				return
			}
			PrintSuccess("Sandbox %s paused", sandboxID)
		}(id)
	}
	wg.Wait()
}
