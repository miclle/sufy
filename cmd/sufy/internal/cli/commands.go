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

package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/sufy-dev/sufy/sandbox"
)

// KillInfo holds the flags passed to KillBatch.
type KillInfo struct {
	SandboxIDs []string
	All        bool
	State      string
	Metadata   string
}

// Kill terminates the given sandbox.
func Kill(sandboxID string) {
	KillBatch(KillInfo{SandboxIDs: []string{sandboxID}})
}

// KillBatch terminates one or more sandboxes, or all sandboxes matching filters.
func KillBatch(info KillInfo) {
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
			sb, err := client.Connect(ctx, sandboxID, sandbox.ConnectParams{Timeout: 10})
			if err != nil {
				PrintError("connect sandbox %s failed: %v", sandboxID, err)
				return
			}
			if err := sb.Kill(ctx); err != nil {
				if strings.Contains(err.Error(), "404") {
					PrintWarn("sandbox %s not found (already killed?)", sandboxID)
					return
				}
				PrintError("kill sandbox %s failed: %v", sandboxID, err)
				return
			}
			PrintSuccess("Sandbox %s killed", sandboxID)
		}(id)
	}
	wg.Wait()
}

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
			sb, err := client.Connect(ctx, sandboxID, sandbox.ConnectParams{Timeout: 10})
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
			_, err := client.Connect(ctx, sandboxID, sandbox.ConnectParams{Timeout: connectTimeout})
			if err != nil {
				PrintError("resume sandbox %s failed: %v", sandboxID, err)
				return
			}
			PrintSuccess("Sandbox %s resumed", sandboxID)
		}(id)
	}
	wg.Wait()
}

// listSandboxIDs fetches sandbox IDs matching the given state and metadata filters.
func listSandboxIDs(ctx context.Context, client *sandbox.Client, stateFilter, metadata string) ([]string, error) {
	params := &sandbox.ListParams{}
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

// Connect opens an interactive PTY session to an existing sandbox. The sandbox
// is left alive after the session ends.
func Connect(sandboxID string) {
	if sandboxID == "" {
		PrintError("sandbox ID is required")
		return
	}
	client := MustNewSandboxClient()
	ctx := context.Background()
	sb, err := client.Connect(ctx, sandboxID, sandbox.ConnectParams{Timeout: 300})
	if err != nil {
		PrintError("connect sandbox failed: %v", err)
		return
	}
	fmt.Printf("Connecting to sandbox %s...\n", sb.ID())
	RunTerminalSession(ctx, sb)
	fmt.Printf("\nDisconnected from sandbox %s.\n", sb.ID())
}

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

	params := &sandbox.ListParams{}
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

	params := sandbox.CreateParams{TemplateID: info.TemplateID}
	if info.Timeout > 0 {
		params.Timeout = &info.Timeout
	}
	if info.Metadata != "" {
		meta := sandbox.Metadata(ParseKeyValueMap(info.Metadata))
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

	sb, err := client.Connect(ctx, info.SandboxID, sandbox.ConnectParams{Timeout: 10})
	if err != nil {
		PrintError("connect to sandbox %s failed: %v", info.SandboxID, err)
		return
	}

	cmd := strings.Join(info.Command, " ")

	var opts []sandbox.CommandOption
	if info.Cwd != "" {
		opts = append(opts, sandbox.WithCwd(info.Cwd))
	}
	if info.User != "" {
		opts = append(opts, sandbox.WithCommandUser(info.User))
	}
	if env := ParseEnvPairs(info.EnvVars); len(env) > 0 {
		opts = append(opts, sandbox.WithEnvs(env))
	}
	if IsPipedStdin() {
		opts = append(opts, sandbox.WithStdin())
	}

	if info.Background {
		ExecBackground(ctx, sb, cmd, opts)
	} else {
		ExecForeground(ctx, sb, cmd, opts)
	}
}

// LogsInfo holds the flags passed to Logs.
type LogsInfo struct {
	SandboxID string
	Level     string
	Limit     int32
	Format    string
	Follow    bool
	Loggers   string
}

// Logs streams or prints sandbox logs.
func Logs(info LogsInfo) {
	if info.SandboxID == "" {
		PrintError("sandbox ID is required")
		return
	}

	client := MustNewSandboxClient()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signalNotify(sigCh)
	go func() {
		<-sigCh
		cancel()
	}()
	defer signalStop(sigCh)

	sb, err := client.Connect(ctx, info.SandboxID, sandbox.ConnectParams{Timeout: 10})
	if err != nil {
		PrintError("connect to sandbox %s failed: %v", info.SandboxID, err)
		return
	}

	level := info.Level
	if level == "" {
		level = "info"
	}
	loggerPrefixes := SplitCSV(info.Loggers)

	sandboxDone := make(chan struct{})
	if info.Follow {
		go WatchSandboxRunning(ctx, sb, sandboxDone)
	}

	var start *int64
	for {
		params := &sandbox.GetLogsParams{Start: start}
		if info.Limit > 0 && start == nil {
			params.Limit = &info.Limit
		}

		logs, lErr := sb.GetLogs(ctx, params)
		if lErr != nil {
			PrintError("get sandbox logs failed: %v", lErr)
			return
		}

		if info.Format == FormatJSON {
			if !info.Follow {
				PrintJSON(logs)
				return
			}
			if len(logs.Logs) > 0 || len(logs.LogEntries) > 0 {
				PrintJSON(logs)
			}
		} else {
			PrintLogEntries(logs, level, loggerPrefixes)
		}

		if !info.Follow {
			if info.Format != FormatJSON && len(logs.Logs) == 0 && len(logs.LogEntries) == 0 {
				fmt.Println("No logs found")
			}
			return
		}

		if len(logs.Logs) > 0 {
			ts := logs.Logs[len(logs.Logs)-1].Timestamp.UnixMilli() + 1
			start = &ts
		} else if len(logs.LogEntries) > 0 {
			ts := logs.LogEntries[len(logs.LogEntries)-1].Timestamp.UnixMilli() + 1
			start = &ts
		}

		select {
		case <-sandboxDone:
			if info.Format != FormatJSON {
				fmt.Println("\nStopped printing logs — sandbox is closed")
			}
			return
		case <-ctx.Done():
			return
		default:
		}

		time.Sleep(400 * time.Millisecond)
	}
}

// MetricsInfo holds the flags passed to Metrics.
type MetricsInfo struct {
	SandboxID string
	Format    string
	Follow    bool
}

// Metrics prints resource-usage metrics for the given sandbox.
func Metrics(info MetricsInfo) {
	if info.SandboxID == "" {
		PrintError("sandbox ID is required")
		return
	}

	client := MustNewSandboxClient()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signalNotify(sigCh)
	go func() {
		<-sigCh
		cancel()
	}()
	defer signalStop(sigCh)

	sb, err := client.Connect(ctx, info.SandboxID, sandbox.ConnectParams{Timeout: 10})
	if err != nil {
		PrintError("connect to sandbox %s failed: %v", info.SandboxID, err)
		return
	}

	sandboxDone := make(chan struct{})
	if info.Follow {
		go WatchSandboxRunning(ctx, sb, sandboxDone)
	}

	var lastTimestamp *time.Time

	for {
		params := &sandbox.GetMetricsParams{}
		if lastTimestamp != nil {
			start := lastTimestamp.Unix()
			params.Start = &start
		}

		metrics, mErr := sb.GetMetrics(ctx, params)
		if mErr != nil {
			PrintError("get sandbox metrics failed: %v", mErr)
			return
		}

		if info.Format == FormatJSON {
			if !info.Follow {
				PrintJSON(metrics)
				return
			}
			if len(metrics) > 0 {
				PrintJSON(metrics)
			}
		} else {
			if !info.Follow && len(metrics) == 0 {
				fmt.Println("No metrics available")
				return
			}
			for _, m := range metrics {
				if lastTimestamp != nil && !m.Timestamp.After(*lastTimestamp) {
					continue
				}
				fmt.Printf("[%s]  %s  %.1f%% / %d Cores | %s  %s / %s | %s  %s / %s\n",
					m.Timestamp.Format(time.RFC3339),
					ColorInfo.Sprint("CPU:"),
					m.CPUUsedPct,
					m.CPUCount,
					ColorInfo.Sprint("Memory:"),
					FormatBytes(m.MemUsed),
					FormatBytes(m.MemTotal),
					ColorInfo.Sprint("Disk:"),
					FormatBytes(m.DiskUsed),
					FormatBytes(m.DiskTotal),
				)
				ts := m.Timestamp
				lastTimestamp = &ts
			}
		}

		if !info.Follow {
			return
		}

		select {
		case <-sandboxDone:
			if info.Format != FormatJSON {
				fmt.Println("\nStopped printing metrics — sandbox is closed")
			}
			return
		case <-ctx.Done():
			return
		default:
		}

		time.Sleep(400 * time.Millisecond)
	}
}
