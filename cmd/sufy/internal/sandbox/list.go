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
	"os"

	sdk "github.com/sufy-dev/sufy/sandbox"
)

// ListInfo holds the flags passed to List.
type ListInfo struct {
	State    string
	Metadata string
	Limit    int32
	Format   string
}

// List prints sandboxes matching the given filter.
func List(info ListInfo) {
	client := MustNewSandboxClient()
	ctx := context.Background()

	params := &sdk.ListParams{}
	stateFilter := info.State
	if stateFilter == "" {
		stateFilter = DefaultState
	}
	states := ParseStates(stateFilter)
	if len(states) > 0 {
		params.State = &states
	}
	if info.Metadata != "" {
		md := ParseMetadataQuery(info.Metadata)
		if md != "" {
			params.Metadata = &md
		}
	}
	if info.Limit > 0 {
		params.Limit = &info.Limit
	}

	sandboxes, err := client.List(ctx, params)
	if err != nil {
		PrintError("list sandboxes failed: %v", err)
		return
	}

	format := info.Format
	if format == "" {
		format = FormatPretty
	}

	if format == FormatJSON {
		PrintJSON(sandboxes)
		return
	}

	if len(sandboxes) == 0 {
		fmt.Println("No sandboxes found.")
		return
	}
	tw := NewTable(os.Stdout)
	fmt.Fprintln(tw, "SANDBOX ID\tTEMPLATE\tSTATE\tCPU\tMEMORY\tSTARTED\tEND\tMETADATA")
	for _, s := range sandboxes {
		md := map[string]string{}
		if s.Metadata != nil {
			md = map[string]string(*s.Metadata)
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%d MiB\t%s\t%s\t%s\n",
			s.SandboxID,
			s.TemplateID,
			string(s.State),
			s.CPUCount,
			s.MemoryMB,
			FormatTimestamp(s.StartedAt),
			FormatTimestamp(s.EndAt),
			FormatMetadata(md),
		)
	}
	_ = tw.Flush()
}

// listSandboxIDs fetches sandbox IDs matching the given state and metadata
// filters. Shared by the Kill, Pause, and Resume batch paths.
func listSandboxIDs(ctx context.Context, client *sdk.Client, stateFilter, metadata string) ([]string, error) {
	params := &sdk.ListParams{}
	states := ParseStates(stateFilter)
	if len(states) > 0 {
		params.State = &states
	}
	if metadata != "" {
		md := ParseMetadataQuery(metadata)
		if md != "" {
			params.Metadata = &md
		}
	}
	sandboxes, err := client.List(ctx, params)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(sandboxes))
	for i, s := range sandboxes {
		ids[i] = s.SandboxID
	}
	return ids, nil
}
