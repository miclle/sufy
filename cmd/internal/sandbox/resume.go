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

// ResumeInfo holds the flags passed to ResumeBatch.
type ResumeInfo struct {
	SandboxIDs []string
	All        bool
	Metadata   string
	Timeout    int32
}

// Resume resumes a paused sandbox. Timeout sets the new lifetime in seconds.
func Resume(sandboxID string, timeout int32) {
	ResumeBatch(ResumeInfo{SandboxIDs: []string{sandboxID}, Timeout: timeout})
}

// ResumeBatch resumes one or more paused sandboxes, or all paused sandboxes.
func ResumeBatch(info ResumeInfo) {
	if !info.All && len(info.SandboxIDs) == 0 {
		PrintError("sandbox ID is required, or use --all")
		return
	}

	client := MustNewSandboxClient()
	ctx := context.Background()

	ids := info.SandboxIDs
	if info.All {
		listed, err := listSandboxIDs(ctx, client, "paused", info.Metadata)
		if err != nil {
			PrintError("list sandboxes failed: %v", err)
			return
		}
		if len(listed) == 0 {
			fmt.Println("No paused sandboxes found.")
			return
		}
		ids = listed
	}

	connectTimeout := info.Timeout
	if connectTimeout <= 0 {
		connectTimeout = 300
	}

	var wg sync.WaitGroup
	for _, id := range ids {
		wg.Add(1)
		go func(sandboxID string) {
			defer wg.Done()
			_, err := client.Connect(ctx, sandboxID, sdk.ConnectParams{Timeout: connectTimeout})
			if err != nil {
				PrintError("resume sandbox %s failed: %v", sandboxID, err)
				return
			}
			PrintSuccess("Sandbox %s resumed", sandboxID)
		}(id)
	}
	wg.Wait()
}
