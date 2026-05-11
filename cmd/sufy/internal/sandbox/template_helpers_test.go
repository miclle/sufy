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
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/sufy-dev/sufy/cmd/sufy/internal/dockerfile"
	sdk "github.com/sufy-dev/sufy/sandbox"
)

// --- validateBuildSourceSelection ---

func TestValidateBuildSourceSelection(t *testing.T) {
	cases := []struct {
		name    string
		info    BuildInfo
		wantErr string
	}{
		{
			name:    "both from-image and from-template",
			info:    BuildInfo{FromImage: "ubuntu", FromTemplate: "base"},
			wantErr: "cannot specify both",
		},
		{
			name:    "none of the three",
			info:    BuildInfo{},
			wantErr: "is required",
		},
		{
			name: "from-image only",
			info: BuildInfo{FromImage: "ubuntu"},
		},
		{
			name: "dockerfile only",
			info: BuildInfo{Dockerfile: "./Dockerfile"},
		},
		{
			name: "from-template only",
			info: BuildInfo{FromTemplate: "base"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateBuildSourceSelection(tc.info)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

// --- validateRebuildSourceSelection ---

func TestValidateRebuildSourceSelection(t *testing.T) {
	// Not in rebuild path (no template ID) → no error.
	if err := validateRebuildSourceSelection(BuildInfo{}, true, true); err != nil {
		t.Errorf("expected no error without TemplateID, got %v", err)
	}

	// Rebuild + CLI flag → error.
	err := validateRebuildSourceSelection(BuildInfo{TemplateID: "tmpl"}, true, false)
	if err == nil {
		t.Error("expected error for rebuild + cliFromImage")
	}
	err = validateRebuildSourceSelection(BuildInfo{TemplateID: "tmpl"}, false, true)
	if err == nil {
		t.Error("expected error for rebuild + cliFromTemplate")
	}

	// Rebuild without CLI flags (from_template may come from toml) → OK.
	if err := validateRebuildSourceSelection(BuildInfo{TemplateID: "tmpl", FromTemplate: "base"}, false, false); err != nil {
		t.Errorf("expected no error when from-* comes from toml, got %v", err)
	}
}

// --- buildParamsFromDockerfileResult ---

func TestBuildParamsFromDockerfileResultPrecedence(t *testing.T) {
	result := &dockerfile.ConvertResult{
		BaseImage: "ubuntu:22.04",
		Steps:     []sdk.TemplateStep{},
	}

	// FromTemplate wins.
	p := buildParamsFromDockerfileResult(result, BuildInfo{FromTemplate: "base", FromImage: "ignored"})
	if p.FromTemplate == nil || *p.FromTemplate != "base" {
		t.Errorf("expected FromTemplate=base, got %v", p.FromTemplate)
	}
	if p.FromImage != nil {
		t.Errorf("expected FromImage nil when FromTemplate set, got %v", *p.FromImage)
	}

	// FromImage wins over Dockerfile FROM.
	p = buildParamsFromDockerfileResult(result, BuildInfo{FromImage: "alpine:3"})
	if p.FromImage == nil || *p.FromImage != "alpine:3" {
		t.Errorf("expected FromImage=alpine:3, got %v", p.FromImage)
	}

	// Falls back to Dockerfile FROM.
	p = buildParamsFromDockerfileResult(result, BuildInfo{})
	if p.FromImage == nil || *p.FromImage != "ubuntu:22.04" {
		t.Errorf("expected FromImage=ubuntu:22.04, got %v", p.FromImage)
	}
}

// --- isTemplateAliasNotFound ---

func TestIsTemplateAliasNotFound(t *testing.T) {
	if isTemplateAliasNotFound(nil) {
		t.Error("nil error should not be treated as not-found")
	}
	if isTemplateAliasNotFound(errors.New("network down")) {
		t.Error("plain error should not be treated as not-found")
	}
	if !isTemplateAliasNotFound(&sdk.APIError{StatusCode: http.StatusNotFound}) {
		t.Error("404 APIError should be treated as not-found")
	}
	if isTemplateAliasNotFound(&sdk.APIError{StatusCode: http.StatusInternalServerError}) {
		t.Error("500 APIError should not be treated as not-found")
	}
	// Wrapped error should still be detected.
	wrapped := errors.Join(errors.New("ctx"), &sdk.APIError{StatusCode: 404})
	if !isTemplateAliasNotFound(wrapped) {
		t.Error("wrapped 404 should be detected via errors.As")
	}
}
