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

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read %s: %v", p, err)
	}
	return string(b)
}

func TestWriteTemplateIDReplacesExistingLine(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, DefaultFileName)
	const input = `# comment header
name = "demo"
   template_id  =   "old"
cpu_count = 2
`
	if err := os.WriteFile(p, []byte(input), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := WriteTemplateID(p, "tmpl-new"); err != nil {
		t.Fatalf("WriteTemplateID: %v", err)
	}
	got := read(t, p)
	if !strings.Contains(got, `   template_id = "tmpl-new"`) {
		t.Errorf("replacement did not preserve indent or set new value:\n%s", got)
	}
	// Comments and other fields preserved.
	if !strings.Contains(got, "# comment header") || !strings.Contains(got, `cpu_count = 2`) {
		t.Errorf("other content not preserved:\n%s", got)
	}
	// Old value gone.
	if strings.Contains(got, `"old"`) {
		t.Errorf("old value still present:\n%s", got)
	}
}

func TestWriteTemplateIDInsertsWhenAbsent(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, DefaultFileName)
	const input = `name = "demo"
cpu_count = 2
`
	if err := os.WriteFile(p, []byte(input), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := WriteTemplateID(p, "tmpl-x"); err != nil {
		t.Fatalf("WriteTemplateID: %v", err)
	}
	got := read(t, p)
	if !strings.HasPrefix(got, `template_id = "tmpl-x"`) {
		t.Errorf("expected template_id at top, got:\n%s", got)
	}
	if !strings.Contains(got, `name = "demo"`) {
		t.Errorf("existing content lost:\n%s", got)
	}
}

func TestWriteTemplateIDIgnoresCommentedLine(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, DefaultFileName)
	const input = `# template_id = "commented"
name = "demo"
`
	if err := os.WriteFile(p, []byte(input), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := WriteTemplateID(p, "tmpl-y"); err != nil {
		t.Fatalf("WriteTemplateID: %v", err)
	}
	got := read(t, p)
	if !strings.HasPrefix(got, `template_id = "tmpl-y"`) {
		t.Errorf("expected new line inserted at top, got:\n%s", got)
	}
	if !strings.Contains(got, `# template_id = "commented"`) {
		t.Errorf("commented line should be preserved, got:\n%s", got)
	}
}

func TestWriteTemplateIDPreservesCRLF(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, DefaultFileName)
	input := "name = \"demo\"\r\ntemplate_id = \"old\"\r\n"
	if err := os.WriteFile(p, []byte(input), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := WriteTemplateID(p, "tmpl-crlf"); err != nil {
		t.Fatalf("WriteTemplateID: %v", err)
	}
	got := read(t, p)
	if !strings.Contains(got, "\r\n") {
		t.Errorf("CRLF line endings lost: %q", got)
	}
	if strings.Contains(got, "\"old\"") {
		t.Errorf("old value still present: %q", got)
	}
	if !strings.Contains(got, "\"tmpl-crlf\"") {
		t.Errorf("new value not written: %q", got)
	}
}

func TestWriteTemplateIDStopsAtNonRootTable(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, DefaultFileName)
	const input = `name = "demo"
[some.section]
template_id = "should-not-touch"
`
	if err := os.WriteFile(p, []byte(input), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := WriteTemplateID(p, "tmpl-z"); err != nil {
		t.Fatalf("WriteTemplateID: %v", err)
	}
	got := read(t, p)
	// The line inside the section must stay untouched (no replacement happened).
	if !strings.Contains(got, `template_id = "should-not-touch"`) {
		t.Errorf("section-scoped template_id should not be replaced:\n%s", got)
	}
	// And a new top-level line must be inserted.
	if !strings.HasPrefix(got, `template_id = "tmpl-z"`) {
		t.Errorf("expected new top-level template_id, got:\n%s", got)
	}
}

func TestWriteTemplateIDMissingFileErrors(t *testing.T) {
	err := WriteTemplateID(filepath.Join(t.TempDir(), "missing.toml"), "x")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestIsCommentLine(t *testing.T) {
	cases := map[string]bool{
		"# foo":       true,
		"   # foo":    true,
		"\t#foo":      true,
		"foo = 1":     false,
		"  foo = #1": false,
		"":            false,
	}
	for in, want := range cases {
		if got := isCommentLine(in); got != want {
			t.Errorf("isCommentLine(%q) = %v, want %v", in, got, want)
		}
	}
}
