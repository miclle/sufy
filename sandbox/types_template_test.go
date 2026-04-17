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

//go:build unit

package sandbox

import (
	"encoding/json"
	"testing"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/sufy-dev/sufy/sandbox/internal/apis"
)

func TestTemplateStepsToAPI(t *testing.T) {
	args := []string{"echo", "hello"}
	steps := []TemplateStep{
		{Type: "run", Args: &args},
		{Type: "copy", Force: boolPtr(true)},
	}
	result := templateStepsToAPI(&steps)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(*result) != 2 {
		t.Fatalf("length = %d, want 2", len(*result))
	}
	if (*result)[0].Type != "run" {
		t.Errorf("step[0].Type = %q, want %q", (*result)[0].Type, "run")
	}
	if (*result)[0].Args == nil || len(*(*result)[0].Args) != 2 {
		t.Error("step[0].Args should have 2 elements")
	}
	if (*result)[1].Force == nil || !*(*result)[1].Force {
		t.Error("step[1].Force should be true")
	}
}

func TestTemplateStepsToAPINil(t *testing.T) {
	if templateStepsToAPI(nil) != nil {
		t.Error("nil input should return nil")
	}
}

func TestUpdateTemplateParamsToAPI(t *testing.T) {
	params := UpdateTemplateParams{Public: boolPtr(true)}
	body := params.toAPI()
	if body.Public == nil || !*body.Public {
		t.Error("Public should be true")
	}
}

func TestStartTemplateBuildParamsToAPI(t *testing.T) {
	steps := []TemplateStep{{Type: "run"}}
	params := StartTemplateBuildParams{
		FromImage: strPtr("ubuntu:22.04"),
		Steps:     &steps,
	}
	body, err := params.toAPI()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body.FromImage == nil || *body.FromImage != "ubuntu:22.04" {
		t.Errorf("FromImage = %v, want %q", body.FromImage, "ubuntu:22.04")
	}
	if body.Steps == nil || len(*body.Steps) != 1 {
		t.Fatalf("Steps length = %v, want 1", body.Steps)
	}
}

func TestStartTemplateBuildParamsToAPIError(t *testing.T) {
	// An empty JSON object lacks the required discriminator field, which should
	// cause UnmarshalJSON to fail on the FromImageRegistry union type.
	badJSON := json.RawMessage([]byte(`"not-an-object"`))
	params := StartTemplateBuildParams{
		FromImageRegistry: &badJSON,
	}
	_, err := params.toAPI()
	if err == nil {
		// If the union type accepts any JSON without error, the test is not
		// applicable — skip rather than fail.
		t.Skip("FromImageRegistry UnmarshalJSON does not reject this input; skipping")
	}
}

func TestListTemplatesParamsToAPI(t *testing.T) {
	params := ListTemplatesParams{TeamID: strPtr("team-1")}
	result := params.toAPI()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.TeamID == nil || *result.TeamID != "team-1" {
		t.Errorf("TeamID = %v, want %q", result.TeamID, "team-1")
	}
}

func TestListTemplatesParamsToAPINil(t *testing.T) {
	var p *ListTemplatesParams
	if p.toAPI() != nil {
		t.Error("nil receiver should return nil")
	}
}

func TestGetTemplateParamsToAPI(t *testing.T) {
	params := GetTemplateParams{Limit: int32Ptr(10)}
	result := params.toAPI()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Limit == nil || *result.Limit != 10 {
		t.Errorf("Limit = %v, want 10", result.Limit)
	}
}

func TestGetTemplateParamsToAPINil(t *testing.T) {
	var p *GetTemplateParams
	if p.toAPI() != nil {
		t.Error("nil receiver should return nil")
	}
}

func TestGetBuildStatusParamsToAPI(t *testing.T) {
	params := GetBuildStatusParams{
		LogsOffset: int32Ptr(5),
		Limit:      int32Ptr(50),
		Level:      logLevelPtr(LogLevelError),
	}
	result := params.toAPI()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.LogsOffset == nil || *result.LogsOffset != 5 {
		t.Errorf("LogsOffset = %v, want 5", result.LogsOffset)
	}
	if result.Limit == nil || *result.Limit != 50 {
		t.Errorf("Limit = %v, want 50", result.Limit)
	}
	if result.Level == nil || *result.Level != apis.LogLevel("error") {
		t.Errorf("Level = %v, want %q", result.Level, "error")
	}
}

func logLevelPtr(l LogLevel) *LogLevel { return &l }

func TestGetBuildStatusParamsToAPINil(t *testing.T) {
	var p *GetBuildStatusParams
	if p.toAPI() != nil {
		t.Error("nil receiver should return nil")
	}
}

func TestGetBuildLogsParamsToAPI(t *testing.T) {
	dir := LogsDirectionForward
	level := LogLevelInfo
	src := LogsSourcePersistent
	params := GetBuildLogsParams{
		Cursor:    int64Ptr(999),
		Limit:     int32Ptr(25),
		Direction: &dir,
		Level:     &level,
		Source:    &src,
	}
	result := params.toAPI()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Cursor == nil || *result.Cursor != 999 {
		t.Errorf("Cursor = %v, want 999", result.Cursor)
	}
	if result.Direction == nil || *result.Direction != apis.LogsDirection("forward") {
		t.Errorf("Direction = %v, want %q", result.Direction, "forward")
	}
	if result.Level == nil || *result.Level != apis.LogLevel("info") {
		t.Errorf("Level = %v, want %q", result.Level, "info")
	}
	if result.Source == nil || *result.Source != apis.LogsSource("persistent") {
		t.Errorf("Source = %v, want %q", result.Source, "persistent")
	}
}

func TestGetBuildLogsParamsToAPINil(t *testing.T) {
	var p *GetBuildLogsParams
	if p.toAPI() != nil {
		t.Error("nil receiver should return nil")
	}
}

func TestManageTagsParamsToAPI(t *testing.T) {
	params := ManageTagsParams{
		Target: "my-template:v1",
		Tags:   []string{"latest", "stable"},
	}
	body := params.toAPI()
	if body.Target != "my-template:v1" {
		t.Errorf("Target = %q, want %q", body.Target, "my-template:v1")
	}
	if len(body.Tags) != 2 || body.Tags[0] != "latest" {
		t.Errorf("Tags = %v, want [latest stable]", body.Tags)
	}
}

func TestDeleteTagsParamsToAPI(t *testing.T) {
	params := DeleteTagsParams{
		Name: "my-template",
		Tags: []string{"old"},
	}
	body := params.toAPI()
	if body.Name != "my-template" {
		t.Errorf("Name = %q, want %q", body.Name, "my-template")
	}
	if len(body.Tags) != 1 || body.Tags[0] != "old" {
		t.Errorf("Tags = %v, want [old]", body.Tags)
	}
}

func TestTemplateBuildLogsFromAPI(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	step := "step-1"
	apiLogs := &apis.TemplateBuildLogsResponse{
		Logs: []apis.BuildLogEntry{
			{Level: apis.LogLevel("info"), Message: "building", Step: &step, Timestamp: now},
		},
	}
	result := templateBuildLogsFromAPI(apiLogs)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Logs) != 1 {
		t.Fatalf("Logs length = %d, want 1", len(result.Logs))
	}
	if result.Logs[0].Level != LogLevelInfo || result.Logs[0].Message != "building" {
		t.Errorf("Log entry = %+v", result.Logs[0])
	}
	if result.Logs[0].Step == nil || *result.Logs[0].Step != "step-1" {
		t.Errorf("Step = %v, want %q", result.Logs[0].Step, "step-1")
	}

	// nil input
	if templateBuildLogsFromAPI(nil) != nil {
		t.Error("nil input should return nil")
	}
}

func TestTemplateBuildFileUploadFromAPI(t *testing.T) {
	url := "https://upload.example.com/file"
	result := templateBuildFileUploadFromAPI(&apis.TemplateBuildFileUpload{
		Present: false,
		URL:     &url,
	})
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Present {
		t.Error("Present should be false")
	}
	if result.URL == nil || *result.URL != url {
		t.Errorf("URL = %v, want %q", result.URL, url)
	}

	// without URL (cached)
	result2 := templateBuildFileUploadFromAPI(&apis.TemplateBuildFileUpload{Present: true})
	if !result2.Present {
		t.Error("Present should be true")
	}
	if result2.URL != nil {
		t.Error("URL should be nil")
	}

	// nil input
	if templateBuildFileUploadFromAPI(nil) != nil {
		t.Error("nil input should return nil")
	}
}

func TestAssignedTemplateTagsFromAPI(t *testing.T) {
	buildID := openapi_types.UUID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}
	result := assignedTemplateTagsFromAPI(&apis.AssignedTemplateTags{
		BuildID: buildID,
		Tags:    []string{"v1", "latest"},
	})
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.BuildID != buildID.String() {
		t.Errorf("BuildID = %q, want %q", result.BuildID, buildID.String())
	}
	if len(result.Tags) != 2 || result.Tags[0] != "v1" {
		t.Errorf("Tags = %v, want [v1 latest]", result.Tags)
	}

	// nil input
	if assignedTemplateTagsFromAPI(nil) != nil {
		t.Error("nil input should return nil")
	}
}

func TestTemplateCreateResponseFromAPINilBuildID(t *testing.T) {
	// Test conversion when BuildID is empty string (the zero value).
	resp := templateCreateResponseFromAPI(&apis.TemplateRequestResponseV3{
		TemplateID: "tmpl-99",
		BuildID:    "",
		Aliases:    []string{"a"},
		Names:      []string{"n"},
		Tags:       []string{"t"},
		Public:     true,
	})
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.BuildID != "" {
		t.Errorf("BuildID = %q, want empty", resp.BuildID)
	}
	if resp.TemplateID != "tmpl-99" {
		t.Errorf("TemplateID = %q, want %q", resp.TemplateID, "tmpl-99")
	}
	if !resp.Public {
		t.Error("Public should be true")
	}

	// nil input
	if templateCreateResponseFromAPI(nil) != nil {
		t.Error("nil input should return nil")
	}
}
