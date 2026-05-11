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

package templatecfg

import (
	"reflect"
	"sort"
	"testing"
)

// newCfg builds a FileConfig with the given defined keys for tests.
func newCfg(c FileConfig, defined ...string) *FileConfig {
	c.defined = map[string]bool{}
	for _, k := range defined {
		c.defined[k] = true
	}
	return &c
}

func TestApplyToFillsZeroValuesAndReportsOverrides(t *testing.T) {
	cfg := newCfg(FileConfig{
		TemplateID:   "tmpl-from-file",
		Name:         "file-name",
		Dockerfile:   "./Dockerfile",
		FromImage:    "ubuntu:22.04",
		FromTemplate: "base",
		StartCmd:     "/start",
		ReadyCmd:     "/ready",
		Path:         "ctx",
		CPUCount:     2,
		MemoryMB:     1024,
	}, "template_id", "name", "dockerfile", "from_image",
		"from_template", "start_cmd", "ready_cmd", "path",
		"cpu_count", "memory_mb")

	// CLI gives: TemplateID + FromImage (different value, must report override).
	dst := BuildFields{
		TemplateID: "cli-id",
		FromImage:  "alpine:3",
	}
	overrides := cfg.ApplyTo(&dst)

	// CLI-supplied fields keep their value; zero fields take the file value.
	if dst.TemplateID != "cli-id" {
		t.Errorf("TemplateID overwritten: %s", dst.TemplateID)
	}
	if dst.FromImage != "alpine:3" {
		t.Errorf("FromImage overwritten: %s", dst.FromImage)
	}
	if dst.Name != "file-name" || dst.Dockerfile != "./Dockerfile" || dst.Path != "ctx" {
		t.Errorf("zero fields not filled: %+v", dst)
	}
	if dst.CPUCount != 2 || dst.MemoryMB != 1024 {
		t.Errorf("numeric zeros not filled: %+v", dst)
	}

	sort.Strings(overrides)
	want := []string{"from_image", "template_id"}
	if !reflect.DeepEqual(overrides, want) {
		t.Errorf("overrides = %v, want %v", overrides, want)
	}
}

func TestApplyToCLISameAsFileNoOverride(t *testing.T) {
	cfg := newCfg(FileConfig{TemplateID: "same"}, "template_id")
	dst := BuildFields{TemplateID: "same"}
	overrides := cfg.ApplyTo(&dst)
	if len(overrides) != 0 {
		t.Errorf("expected no overrides when CLI == file, got %v", overrides)
	}
}

func TestApplyBoolRespectsNoCacheChanged(t *testing.T) {
	// File: no_cache=true, CLI: did NOT change --no-cache (so default false).
	// Expectation: file wins (CLI default is ignored).
	cfg := newCfg(FileConfig{NoCache: true}, "no_cache")
	dst := BuildFields{NoCache: false, NoCacheChanged: false}
	overrides := cfg.ApplyTo(&dst)
	if !dst.NoCache {
		t.Errorf("expected NoCache=true from file, got false")
	}
	if len(overrides) != 0 {
		t.Errorf("unexpected overrides: %v", overrides)
	}

	// File: no_cache=true, CLI: explicitly set --no-cache=false. CLI wins,
	// override is reported.
	dst2 := BuildFields{NoCache: false, NoCacheChanged: true}
	overrides2 := cfg.ApplyTo(&dst2)
	if dst2.NoCache {
		t.Errorf("expected NoCache=false from CLI, got true")
	}
	if len(overrides2) != 1 || overrides2[0] != "no_cache" {
		t.Errorf("expected no_cache override, got %v", overrides2)
	}
}

func TestApplyBoolFileUndefinedRespectsCLI(t *testing.T) {
	cfg := newCfg(FileConfig{NoCache: false}) // no_cache not in defined
	dst := BuildFields{NoCache: true, NoCacheChanged: true}
	overrides := cfg.ApplyTo(&dst)
	if !dst.NoCache {
		t.Errorf("expected CLI value preserved when file does not define no_cache")
	}
	if len(overrides) != 0 {
		t.Errorf("unexpected overrides: %v", overrides)
	}
}

func TestApplyToEmptyFileNoChanges(t *testing.T) {
	cfg := newCfg(FileConfig{})
	dst := BuildFields{Name: "cli", CPUCount: 1}
	overrides := cfg.ApplyTo(&dst)
	if dst.Name != "cli" || dst.CPUCount != 1 {
		t.Errorf("dst mutated by empty file: %+v", dst)
	}
	if len(overrides) != 0 {
		t.Errorf("unexpected overrides: %v", overrides)
	}
}
