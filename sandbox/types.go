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
	"time"

	"github.com/sufy-dev/sufy/sandbox/internal/apis"
)

// ---------------------------------------------------------------------------
// Sandbox-related SDK types
// ---------------------------------------------------------------------------

// Metadata is a set of user-defined key/value pairs attached to a sandbox.
type Metadata map[string]string

// NetworkConfig describes a sandbox's outbound networking policy.
type NetworkConfig struct {
	// AllowOut is an optional list of CIDRs allowed for outbound traffic.
	AllowOut *[]string

	// AllowPublicTraffic enables inbound public traffic to the sandbox URL.
	AllowPublicTraffic *bool

	// DenyOut is an optional list of CIDRs denied for outbound traffic.
	DenyOut *[]string

	// MaskRequestHost overrides the Host header attached to sandbox requests.
	MaskRequestHost *string
}

// SandboxState represents the lifecycle state of a sandbox.
type SandboxState string

// Sandbox state constants.
const (
	StateRunning SandboxState = "running"
	StatePaused  SandboxState = "paused"
)

// CreateParams holds parameters for creating a sandbox.
type CreateParams struct {
	// TemplateID is the template to spawn from. Required.
	TemplateID string

	// Timeout is the sandbox lifetime in seconds. Optional.
	Timeout *int32

	// AutoPause controls whether the sandbox is paused (rather than killed)
	// when the timeout elapses. Optional.
	AutoPause *bool

	// AllowInternetAccess toggles outbound Internet access. Optional.
	AllowInternetAccess *bool

	// Secure enables secure communication mode. Optional.
	Secure *bool

	// EnvVars is an optional map of environment variables for the sandbox.
	EnvVars *map[string]string

	// Metadata is an optional set of user-defined key/value pairs.
	Metadata *Metadata

	// Network configures outbound networking rules. Optional.
	Network *NetworkConfig
}

// ConnectParams holds parameters for connecting to an existing sandbox.
type ConnectParams struct {
	// Timeout is the new lifetime in seconds applied when the sandbox resumes.
	Timeout int32
}

// RefreshParams holds parameters for extending a sandbox's lifetime.
type RefreshParams struct {
	// Duration is the number of seconds to extend the sandbox by. Optional.
	Duration *int
}

// ListParams holds query parameters for listing sandboxes. Supports pagination
// and filtering by state.
type ListParams struct {
	// Metadata filters sandboxes by a metadata query string like "user=abc&app=prod".
	Metadata *string

	// State filters sandboxes by one or more lifecycle states.
	State *[]SandboxState

	// NextToken is the pagination cursor returned by a previous List call.
	NextToken *string

	// Limit caps the number of items returned per page.
	Limit *int32
}

// GetMetricsParams holds query parameters for fetching per-sandbox metrics.
type GetMetricsParams struct {
	// Start is the window start time, as a Unix timestamp in seconds.
	Start *int64

	// End is the window end time, as a Unix timestamp in seconds.
	End *int64
}

func (p *GetMetricsParams) toAPI() *apis.GetSandboxMetricsParams {
	if p == nil {
		return nil
	}
	return &apis.GetSandboxMetricsParams{
		Start: p.Start,
		End:   p.End,
	}
}

// GetLogsParams holds query parameters for fetching sandbox logs.
type GetLogsParams struct {
	// Start is the earliest log timestamp to include, in milliseconds.
	Start *int64

	// Limit caps the number of log entries returned.
	Limit *int32
}

func (p *GetLogsParams) toAPI() *apis.GetSandboxLogsParams {
	if p == nil {
		return nil
	}
	return &apis.GetSandboxLogsParams{
		Start: p.Start,
		Limit: p.Limit,
	}
}

// GetSandboxesMetricsParams holds query parameters for batch-fetching metrics
// for multiple sandboxes.
type GetSandboxesMetricsParams struct {
	// SandboxIds is the list of sandbox identifiers to fetch metrics for.
	SandboxIds []string
}

func (p *GetSandboxesMetricsParams) toAPI() *apis.GetSandboxesMetricsParams {
	if p == nil {
		return nil
	}
	return &apis.GetSandboxesMetricsParams{
		SandboxIds: p.SandboxIds,
	}
}

// SandboxInfo holds the full details of a sandbox.
type SandboxInfo struct {
	SandboxID   string
	TemplateID  string
	ClientID    string
	Alias       *string
	Domain      *string
	State       SandboxState
	CPUCount    int32
	MemoryMB    int32
	DiskSizeMB  int32
	EnvdVersion string
	StartedAt   time.Time
	EndAt       time.Time
	Metadata    *Metadata
}

// ListedSandbox is an entry returned by List.
type ListedSandbox struct {
	SandboxID   string
	TemplateID  string
	ClientID    string
	Alias       *string
	State       SandboxState
	CPUCount    int32
	MemoryMB    int32
	DiskSizeMB  int32
	EnvdVersion string
	StartedAt   time.Time
	EndAt       time.Time
	Metadata    *Metadata
}

// SandboxMetric captures a single resource-metric snapshot for a sandbox.
type SandboxMetric struct {
	CPUCount      int32
	CPUUsedPct    float32
	MemTotal      int64
	MemUsed       int64
	DiskTotal     int64
	DiskUsed      int64
	Timestamp     time.Time
	TimestampUnix int64
}

// SandboxLogs groups both raw and structured sandbox log entries.
type SandboxLogs struct {
	Logs       []SandboxLog
	LogEntries []SandboxLogEntry
}

// SandboxLog is a raw log line with a timestamp.
type SandboxLog struct {
	Line      string
	Timestamp time.Time
}

// SandboxLogEntry is a structured sandbox log record.
type SandboxLogEntry struct {
	Level     LogLevel
	Message   string
	Fields    map[string]string
	Timestamp time.Time
}

// SandboxesWithMetrics is a batch metric result keyed by sandbox ID.
type SandboxesWithMetrics struct {
	Sandboxes map[string]SandboxMetric
}

// ---------------------------------------------------------------------------
// Conversion helpers — apis → SDK
// ---------------------------------------------------------------------------

func sandboxInfoFromAPI(d *apis.SandboxDetail) *SandboxInfo {
	if d == nil {
		return nil
	}
	info := &SandboxInfo{
		SandboxID:   d.SandboxID,
		TemplateID:  d.TemplateID,
		ClientID:    d.ClientID,
		Alias:       d.Alias,
		Domain:      d.Domain,
		State:       SandboxState(d.State),
		CPUCount:    d.CPUCount,
		MemoryMB:    d.MemoryMB,
		DiskSizeMB:  d.DiskSizeMB,
		EnvdVersion: d.EnvdVersion,
		StartedAt:   d.StartedAt,
		EndAt:       d.EndAt,
	}
	if d.Metadata != nil {
		m := Metadata(*d.Metadata)
		info.Metadata = &m
	}
	return info
}

func listedSandboxFromAPI(a apis.ListedSandbox) ListedSandbox {
	ls := ListedSandbox{
		SandboxID:   a.SandboxID,
		TemplateID:  a.TemplateID,
		ClientID:    a.ClientID,
		Alias:       a.Alias,
		State:       SandboxState(a.State),
		CPUCount:    a.CPUCount,
		MemoryMB:    a.MemoryMB,
		DiskSizeMB:  a.DiskSizeMB,
		EnvdVersion: a.EnvdVersion,
		StartedAt:   a.StartedAt,
		EndAt:       a.EndAt,
	}
	if a.Metadata != nil {
		m := Metadata(*a.Metadata)
		ls.Metadata = &m
	}
	return ls
}

func listedSandboxesFromAPI(a []apis.ListedSandbox) []ListedSandbox {
	if a == nil {
		return nil
	}
	result := make([]ListedSandbox, len(a))
	for i, s := range a {
		result[i] = listedSandboxFromAPI(s)
	}
	return result
}

func sandboxMetricFromAPI(a apis.SandboxMetric) SandboxMetric {
	return SandboxMetric{
		CPUCount:      a.CPUCount,
		CPUUsedPct:    a.CPUUsedPct,
		MemTotal:      a.MemTotal,
		MemUsed:       a.MemUsed,
		DiskTotal:     a.DiskTotal,
		DiskUsed:      a.DiskUsed,
		Timestamp:     a.Timestamp,
		TimestampUnix: a.TimestampUnix,
	}
}

func sandboxMetricsFromAPI(a []apis.SandboxMetric) []SandboxMetric {
	if a == nil {
		return nil
	}
	result := make([]SandboxMetric, len(a))
	for i, m := range a {
		result[i] = sandboxMetricFromAPI(m)
	}
	return result
}

func sandboxLogsFromAPI(a *apis.SandboxLogs) *SandboxLogs {
	if a == nil {
		return nil
	}
	result := &SandboxLogs{
		Logs:       make([]SandboxLog, 0, len(a.Logs)),
		LogEntries: make([]SandboxLogEntry, 0, len(a.LogEntries)),
	}
	for _, l := range a.Logs {
		result.Logs = append(result.Logs, SandboxLog{Line: l.Line, Timestamp: l.Timestamp})
	}
	for _, e := range a.LogEntries {
		result.LogEntries = append(result.LogEntries, SandboxLogEntry{
			Level:     LogLevel(e.Level),
			Message:   e.Message,
			Fields:    e.Fields,
			Timestamp: e.Timestamp,
		})
	}
	return result
}

func sandboxesWithMetricsFromAPI(a *apis.SandboxesWithMetrics) *SandboxesWithMetrics {
	if a == nil {
		return nil
	}
	result := &SandboxesWithMetrics{Sandboxes: make(map[string]SandboxMetric, len(a.Sandboxes))}
	for k, v := range a.Sandboxes {
		result.Sandboxes[k] = sandboxMetricFromAPI(v)
	}
	return result
}

// ---------------------------------------------------------------------------
// Conversion helpers — SDK → apis
// ---------------------------------------------------------------------------

func (p *CreateParams) toAPI() apis.CreateSandboxJSONRequestBody {
	body := apis.CreateSandboxJSONRequestBody{
		TemplateID:          p.TemplateID,
		Timeout:             p.Timeout,
		AutoPause:           p.AutoPause,
		AllowInternetAccess: p.AllowInternetAccess,
		Secure:              p.Secure,
	}
	if p.EnvVars != nil {
		ev := apis.EnvVars(*p.EnvVars)
		body.EnvVars = &ev
	}
	if p.Metadata != nil {
		m := apis.SandboxMetadata(*p.Metadata)
		body.Metadata = &m
	}
	if p.Network != nil {
		body.Network = &apis.SandboxNetworkConfig{
			AllowOut:           p.Network.AllowOut,
			AllowPublicTraffic: p.Network.AllowPublicTraffic,
			DenyOut:            p.Network.DenyOut,
			MaskRequestHost:    p.Network.MaskRequestHost,
		}
	}
	return body
}

func (p *ConnectParams) toAPI() apis.ConnectSandboxJSONRequestBody {
	return apis.ConnectSandboxJSONRequestBody{
		Timeout: p.Timeout,
	}
}

func (p *RefreshParams) toAPI() apis.RefreshSandboxJSONRequestBody {
	return apis.RefreshSandboxJSONRequestBody{
		Duration: p.Duration,
	}
}

func (p *ListParams) toAPI() *apis.ListSandboxesV2Params {
	if p == nil {
		return nil
	}
	params := &apis.ListSandboxesV2Params{
		Metadata:  p.Metadata,
		NextToken: p.NextToken,
		Limit:     p.Limit,
	}
	if p.State != nil {
		states := make([]apis.SandboxState, len(*p.State))
		for i, s := range *p.State {
			states[i] = apis.SandboxState(s)
		}
		params.State = &states
	}
	return params
}
