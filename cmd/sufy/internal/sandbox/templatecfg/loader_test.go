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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestLoadMissingReturnsNil(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "missing.toml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg != nil {
		t.Fatalf("expected nil config for missing file, got %+v", cfg)
	}
}

func TestLoadParsesFieldsAndRecordsDefined(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, DefaultFileName)
	writeFile(t, p, `
template_id = "tmpl-abc"
name = "demo"
dockerfile = "./Dockerfile"
cpu_count = 4
memory_mb = 1024
no_cache = true
`)
	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.TemplateID != "tmpl-abc" || cfg.Name != "demo" || cfg.Dockerfile != "./Dockerfile" {
		t.Fatalf("string fields not parsed: %+v", cfg)
	}
	if cfg.CPUCount != 4 || cfg.MemoryMB != 1024 || !cfg.NoCache {
		t.Fatalf("numeric/bool fields not parsed: %+v", cfg)
	}
	if !cfg.defined["no_cache"] {
		t.Errorf("expected no_cache to be marked defined")
	}
	if cfg.defined["from_image"] {
		t.Errorf("from_image should not be marked defined when absent")
	}
	if cfg.SourcePath() == "" {
		t.Errorf("expected SourcePath to be set")
	}
}

func TestLoadParseErrorReturnsError(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bad.toml")
	writeFile(t, p, "this is = not valid = toml")
	if _, err := Load(p); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestFindInDirAndLoadFromCwd(t *testing.T) {
	dir := t.TempDir()

	// FindInDir returns "" when missing.
	got, err := FindInDir(dir)
	if err != nil {
		t.Fatalf("FindInDir: %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty path, got %q", got)
	}

	// Create the file and ensure both helpers return it.
	target := filepath.Join(dir, DefaultFileName)
	writeFile(t, target, `name = "x"`)
	got, err = FindInDir(dir)
	if err != nil {
		t.Fatalf("FindInDir: %v", err)
	}
	if !strings.HasSuffix(got, DefaultFileName) {
		t.Fatalf("FindInDir returned %q", got)
	}

	// LoadFromCwd works after chdir.
	origWD, _ := os.Getwd()
	defer os.Chdir(origWD)
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	cfg, err := LoadFromCwd()
	if err != nil {
		t.Fatalf("LoadFromCwd: %v", err)
	}
	if cfg == nil || cfg.Name != "x" {
		t.Fatalf("expected cfg.Name=x, got %+v", cfg)
	}
}

func TestLoadFromCwdNoFileReturnsNil(t *testing.T) {
	dir := t.TempDir()
	origWD, _ := os.Getwd()
	defer os.Chdir(origWD)
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	cfg, err := LoadFromCwd()
	if err != nil {
		t.Fatalf("LoadFromCwd: %v", err)
	}
	if cfg != nil {
		t.Fatalf("expected nil cfg, got %+v", cfg)
	}
}
