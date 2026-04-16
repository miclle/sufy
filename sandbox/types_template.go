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
	"encoding/json"
	"fmt"
	"time"

	"github.com/sufy-dev/sufy/sandbox/internal/apis"
)

// ---------------------------------------------------------------------------
// Template-related SDK types
// ---------------------------------------------------------------------------

// TemplateBuildStatus is the build status of a template.
type TemplateBuildStatus string

// Template build status constants.
const (
	BuildStatusReady    TemplateBuildStatus = "ready"
	BuildStatusError    TemplateBuildStatus = "error"
	BuildStatusBuilding TemplateBuildStatus = "building"
	BuildStatusWaiting  TemplateBuildStatus = "waiting"
	BuildStatusUploaded TemplateBuildStatus = "uploaded"
)

// LogLevel is a log-entry severity.
type LogLevel string

// Log level constants.
const (
	LogLevelDebug LogLevel = "debug"
	LogLevelError LogLevel = "error"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
)

// LogsDirection selects the chronological direction for fetching logs.
type LogsDirection string

// Logs direction constants.
const (
	LogsDirectionBackward LogsDirection = "backward"
	LogsDirectionForward  LogsDirection = "forward"
)

// LogsSource selects the storage source for fetching logs.
type LogsSource string

// Logs source constants.
const (
	LogsSourcePersistent LogsSource = "persistent"
	LogsSourceTemporary  LogsSource = "temporary"
)

// TemplateStep describes one build step of a template.
type TemplateStep struct {
	// Args holds the step arguments.
	Args *[]string

	// FilesHash is a hash of files used by the step.
	FilesHash *string

	// Force disables build caching for this step.
	Force *bool

	// Type identifies the step type.
	Type string
}

func templateStepsToAPI(steps *[]TemplateStep) *[]apis.TemplateStep {
	if steps == nil {
		return nil
	}
	result := make([]apis.TemplateStep, len(*steps))
	for i, s := range *steps {
		result[i] = apis.TemplateStep{
			Args:      s.Args,
			FilesHash: s.FilesHash,
			Force:     s.Force,
			Type:      s.Type,
		}
	}
	return &result
}

// FromImageRegistry is a registry-authentication payload. It is a JSON union
// (AWS / GCP / General) whose raw form is delegated to the server for parsing.
type FromImageRegistry = json.RawMessage

// CreateTemplateParams holds parameters for creating a template.
type CreateTemplateParams struct {
	// Alias is the template alias. Deprecated: use Name.
	Alias *string

	// CPUCount sets the sandbox CPU count.
	CPUCount *int32

	// MemoryMB sets the sandbox memory (in MiB).
	MemoryMB *int32

	// Name is the template name, optionally with a tag ("my-template" or
	// "my-template:v1").
	Name *string

	// Tags lists the tags assigned to this template build.
	Tags *[]string

	// TeamID is the team identifier. Deprecated.
	TeamID *string
}

func (p *CreateTemplateParams) toAPI() apis.CreateTemplateV3JSONRequestBody {
	return apis.CreateTemplateV3JSONRequestBody{
		Alias:    p.Alias,
		CPUCount: p.CPUCount,
		MemoryMB: p.MemoryMB,
		Name:     p.Name,
		Tags:     p.Tags,
		TeamID:   p.TeamID,
	}
}

// UpdateTemplateParams holds parameters for updating a template.
type UpdateTemplateParams struct {
	// Public toggles the template's visibility.
	Public *bool
}

func (p *UpdateTemplateParams) toAPI() apis.UpdateTemplateJSONRequestBody {
	return apis.UpdateTemplateJSONRequestBody{
		Public: p.Public,
	}
}

// StartTemplateBuildParams holds parameters for starting a template build.
type StartTemplateBuildParams struct {
	// Force forces a full rebuild, skipping the cache.
	Force *bool

	// FromImage is the base image used as the build root.
	FromImage *string

	// FromImageRegistry carries registry-authentication details.
	FromImageRegistry *FromImageRegistry

	// FromTemplate is an existing template used as the build root.
	FromTemplate *string

	// ReadyCmd is a command used to probe readiness after build.
	ReadyCmd *string

	// StartCmd is the command executed on sandbox startup.
	StartCmd *string

	// Steps is an ordered list of build steps.
	Steps *[]TemplateStep
}

func (p *StartTemplateBuildParams) toAPI() (apis.StartTemplateBuildV2JSONRequestBody, error) {
	body := apis.StartTemplateBuildV2JSONRequestBody{
		Force:        p.Force,
		FromImage:    p.FromImage,
		FromTemplate: p.FromTemplate,
		ReadyCmd:     p.ReadyCmd,
		StartCmd:     p.StartCmd,
		Steps:        templateStepsToAPI(p.Steps),
	}
	if p.FromImageRegistry != nil {
		reg := apis.FromImageRegistry{}
		if err := reg.UnmarshalJSON(*p.FromImageRegistry); err != nil {
			return body, fmt.Errorf("unmarshal from_image_registry: %w", err)
		}
		body.FromImageRegistry = &reg
	}
	return body, nil
}

// ListTemplatesParams holds query parameters for listing templates.
type ListTemplatesParams struct {
	// TeamID filters by team. Optional.
	TeamID *string
}

func (p *ListTemplatesParams) toAPI() *apis.ListTemplatesParams {
	if p == nil {
		return nil
	}
	return &apis.ListTemplatesParams{
		TeamID: p.TeamID,
	}
}

// GetTemplateParams holds query parameters for fetching template details.
type GetTemplateParams struct {
	// NextToken is the pagination cursor.
	NextToken *string

	// Limit caps the number of builds returned.
	Limit *int32
}

func (p *GetTemplateParams) toAPI() *apis.GetTemplateParams {
	if p == nil {
		return nil
	}
	return &apis.GetTemplateParams{
		NextToken: p.NextToken,
		Limit:     p.Limit,
	}
}

// GetBuildStatusParams holds query parameters for fetching build status.
type GetBuildStatusParams struct {
	// LogsOffset is the index of the first log line to return.
	LogsOffset *int32

	// Limit caps the number of log lines returned.
	Limit *int32

	// Level filters logs by severity.
	Level *LogLevel
}

func (p *GetBuildStatusParams) toAPI() *apis.GetTemplateBuildStatusParams {
	if p == nil {
		return nil
	}
	params := &apis.GetTemplateBuildStatusParams{
		LogsOffset: p.LogsOffset,
		Limit:      p.Limit,
	}
	if p.Level != nil {
		level := apis.LogLevel(*p.Level)
		params.Level = &level
	}
	return params
}

// GetBuildLogsParams holds query parameters for fetching build logs.
type GetBuildLogsParams struct {
	// Cursor is the starting timestamp in milliseconds.
	Cursor *int64

	// Limit caps the number of log lines returned.
	Limit *int32

	// Direction selects the chronological direction.
	Direction *LogsDirection

	// Level filters logs by severity.
	Level *LogLevel

	// Source selects the storage source.
	Source *LogsSource
}

func (p *GetBuildLogsParams) toAPI() *apis.GetTemplateBuildLogsParams {
	if p == nil {
		return nil
	}
	params := &apis.GetTemplateBuildLogsParams{
		Cursor: p.Cursor,
		Limit:  p.Limit,
	}
	if p.Direction != nil {
		dir := apis.LogsDirection(*p.Direction)
		params.Direction = &dir
	}
	if p.Level != nil {
		level := apis.LogLevel(*p.Level)
		params.Level = &level
	}
	if p.Source != nil {
		src := apis.LogsSource(*p.Source)
		params.Source = &src
	}
	return params
}

// ManageTagsParams holds parameters for assigning tags to a template build.
type ManageTagsParams struct {
	// Tags is the list of tag names to assign.
	Tags []string

	// Target is the template target in "name:tag" form.
	Target string
}

func (p *ManageTagsParams) toAPI() apis.AssignTemplateTagsJSONRequestBody {
	return apis.AssignTemplateTagsJSONRequestBody{
		Tags:   p.Tags,
		Target: p.Target,
	}
}

// DeleteTagsParams holds parameters for deleting tags from a template.
type DeleteTagsParams struct {
	// Name is the template name.
	Name string

	// Tags is the list of tag names to delete.
	Tags []string
}

func (p *DeleteTagsParams) toAPI() apis.DeleteTemplateTagsJSONRequestBody {
	return apis.DeleteTemplateTagsJSONRequestBody{
		Name: p.Name,
		Tags: p.Tags,
	}
}

// Template represents a template's primary attributes.
type Template struct {
	TemplateID    string
	Aliases       []string
	BuildID       string
	BuildStatus   TemplateBuildStatus
	BuildCount    int32
	CPUCount      int32
	MemoryMB      int32
	DiskSizeMB    int32
	EnvdVersion   string
	Public        bool
	SpawnCount    int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
	LastSpawnedAt *time.Time
}

// TemplateBuild represents a single build of a template.
type TemplateBuild struct {
	BuildID     string
	Status      TemplateBuildStatus
	CPUCount    int32
	MemoryMB    int32
	DiskSizeMB  *int32
	EnvdVersion *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	FinishedAt  *time.Time
}

// TemplateWithBuilds represents a template together with its build history.
type TemplateWithBuilds struct {
	TemplateID    string
	Aliases       []string
	Public        bool
	SpawnCount    int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
	LastSpawnedAt *time.Time
	Builds        []TemplateBuild
}

// TemplateBuildInfo is a snapshot of a build's state and logs.
type TemplateBuildInfo struct {
	TemplateID string
	BuildID    string
	Status     TemplateBuildStatus
	Logs       []string
}

// TemplateBuildLogs is a list of build log entries.
type TemplateBuildLogs struct {
	Logs []BuildLogEntry
}

// BuildLogEntry is a structured build log record.
type BuildLogEntry struct {
	Level     LogLevel
	Message   string
	Step      *string
	Timestamp time.Time
}

// TemplateCreateResponse is the response body returned by CreateTemplate.
type TemplateCreateResponse struct {
	TemplateID string
	BuildID    string
	Aliases    []string
	Names      []string
	Tags       []string
	Public     bool
}

// TemplateBuildFileUpload holds upload information for a template build file.
type TemplateBuildFileUpload struct {
	// Present indicates the file is already cached server-side.
	Present bool
	// URL is the upload target when the file is not cached.
	URL *string
}

// TemplateAliasResponse resolves a template alias.
type TemplateAliasResponse struct {
	TemplateID string
	Public     bool
}

// AssignedTemplateTags is the response returned by AssignTemplateTags.
type AssignedTemplateTags struct {
	BuildID string
	Tags    []string
}

// ---------------------------------------------------------------------------
// Conversion helpers — apis → SDK
// ---------------------------------------------------------------------------

func templateFromAPI(a apis.Template) Template {
	return Template{
		TemplateID:    a.TemplateID,
		Aliases:       a.Aliases,
		BuildID:       a.BuildID,
		BuildStatus:   TemplateBuildStatus(a.BuildStatus),
		BuildCount:    a.BuildCount,
		CPUCount:      a.CPUCount,
		MemoryMB:      a.MemoryMB,
		DiskSizeMB:    a.DiskSizeMB,
		EnvdVersion:   a.EnvdVersion,
		Public:        a.Public,
		SpawnCount:    a.SpawnCount,
		CreatedAt:     a.CreatedAt,
		UpdatedAt:     a.UpdatedAt,
		LastSpawnedAt: a.LastSpawnedAt,
	}
}

func templatesFromAPI(a []apis.Template) []Template {
	if a == nil {
		return nil
	}
	result := make([]Template, len(a))
	for i, t := range a {
		result[i] = templateFromAPI(t)
	}
	return result
}

func templateBuildFromAPI(a apis.TemplateBuild) TemplateBuild {
	return TemplateBuild{
		BuildID:     a.BuildID.String(),
		Status:      TemplateBuildStatus(a.Status),
		CPUCount:    a.CPUCount,
		MemoryMB:    a.MemoryMB,
		DiskSizeMB:  a.DiskSizeMB,
		EnvdVersion: a.EnvdVersion,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
		FinishedAt:  a.FinishedAt,
	}
}

func templateWithBuildsFromAPI(a *apis.TemplateWithBuilds) *TemplateWithBuilds {
	if a == nil {
		return nil
	}
	result := &TemplateWithBuilds{
		TemplateID:    a.TemplateID,
		Aliases:       a.Aliases,
		Public:        a.Public,
		SpawnCount:    a.SpawnCount,
		CreatedAt:     a.CreatedAt,
		UpdatedAt:     a.UpdatedAt,
		LastSpawnedAt: a.LastSpawnedAt,
		Builds:        make([]TemplateBuild, 0, len(a.Builds)),
	}
	for _, b := range a.Builds {
		result.Builds = append(result.Builds, templateBuildFromAPI(b))
	}
	return result
}

func templateBuildInfoFromAPI(a *apis.TemplateBuildInfo) *TemplateBuildInfo {
	if a == nil {
		return nil
	}
	return &TemplateBuildInfo{
		TemplateID: a.TemplateID,
		BuildID:    a.BuildID,
		Status:     TemplateBuildStatus(a.Status),
		Logs:       a.Logs,
	}
}

func templateBuildLogsFromAPI(a *apis.TemplateBuildLogsResponse) *TemplateBuildLogs {
	if a == nil {
		return nil
	}
	result := &TemplateBuildLogs{Logs: make([]BuildLogEntry, 0, len(a.Logs))}
	for _, e := range a.Logs {
		result.Logs = append(result.Logs, BuildLogEntry{
			Level:     LogLevel(e.Level),
			Message:   e.Message,
			Step:      e.Step,
			Timestamp: e.Timestamp,
		})
	}
	return result
}

func templateCreateResponseFromAPI(a *apis.TemplateRequestResponseV3) *TemplateCreateResponse {
	if a == nil {
		return nil
	}
	return &TemplateCreateResponse{
		TemplateID: a.TemplateID,
		BuildID:    a.BuildID,
		Aliases:    a.Aliases,
		Names:      a.Names,
		Tags:       a.Tags,
		Public:     a.Public,
	}
}

func templateBuildFileUploadFromAPI(a *apis.TemplateBuildFileUpload) *TemplateBuildFileUpload {
	if a == nil {
		return nil
	}
	return &TemplateBuildFileUpload{
		Present: a.Present,
		URL:     a.URL,
	}
}

func templateAliasResponseFromAPI(a *apis.TemplateAliasResponse) *TemplateAliasResponse {
	if a == nil {
		return nil
	}
	return &TemplateAliasResponse{
		TemplateID: a.TemplateID,
		Public:     a.Public,
	}
}

func assignedTemplateTagsFromAPI(a *apis.AssignedTemplateTags) *AssignedTemplateTags {
	if a == nil {
		return nil
	}
	return &AssignedTemplateTags{
		BuildID: a.BuildID.String(),
		Tags:    a.Tags,
	}
}
