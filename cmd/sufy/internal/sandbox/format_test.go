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
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/fatih/color"
)

func init() {
	// Disable ANSI color codes for deterministic test output.
	color.NoColor = true
}

func TestLogLevelBadge_KnownLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error"}
	for _, lv := range levels {
		got := LogLevelBadge(lv)
		want := fmt.Sprintf("%-5s", strings.ToUpper(lv))
		if got != want {
			t.Errorf("LogLevelBadge(%q) = %q, want %q", lv, got, want)
		}
	}
}

func TestLogLevelBadge_UnknownLevel(t *testing.T) {
	got := LogLevelBadge("trace")
	want := fmt.Sprintf("%-5s", "TRACE")
	if got != want {
		t.Errorf("LogLevelBadge(trace) = %q, want %q", got, want)
	}
}

func TestLogLevelBadge_CaseInsensitive(t *testing.T) {
	got := LogLevelBadge("INFO")
	want := fmt.Sprintf("%-5s", "INFO")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestNewTable_WritesTabAligned(t *testing.T) {
	var buf bytes.Buffer
	w := NewTable(&buf)
	fmt.Fprintln(w, "A\tB\tC")
	fmt.Fprintln(w, "123\t4\t5")
	if err := w.Flush(); err != nil {
		t.Fatalf("flush error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "A") || !strings.Contains(out, "B") || !strings.Contains(out, "C") {
		t.Errorf("unexpected output: %q", out)
	}
	// tabwriter with padding 2 should insert spaces between columns.
	if !strings.Contains(out, "  ") {
		t.Errorf("expected padding in tabwriter output: %q", out)
	}
}

func TestFormatTimestamp_Zero(t *testing.T) {
	if got := FormatTimestamp(time.Time{}); got != "-" {
		t.Errorf("expected '-' for zero time, got %q", got)
	}
}

func TestFormatTimestamp_NonZero(t *testing.T) {
	tm := time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)
	got := FormatTimestamp(tm)
	if got != "2026-04-21T12:00:00Z" {
		t.Errorf("got %q", got)
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		b    int64
		want string
	}{
		{0, "0 MiB"},
		{1024 * 1024, "1 MiB"},
		{1024 * 1024 * 5, "5 MiB"},
		{1024*1024 + 512*1024, "1.5 MiB"},
	}
	for _, tt := range tests {
		if got := FormatBytes(tt.b); got != tt.want {
			t.Errorf("FormatBytes(%d) = %q, want %q", tt.b, got, tt.want)
		}
	}
}

func TestFormatMetadata_Empty(t *testing.T) {
	if got := FormatMetadata(nil); got != "-" {
		t.Errorf("expected '-' for nil, got %q", got)
	}
	if got := FormatMetadata(map[string]string{}); got != "-" {
		t.Errorf("expected '-' for empty, got %q", got)
	}
}

func TestFormatMetadata_NonEmpty(t *testing.T) {
	// Order is not guaranteed; check both possibilities.
	got := FormatMetadata(map[string]string{"a": "1", "b": "2"})
	if got != "a=1, b=2" && got != "b=2, a=1" {
		t.Errorf("unexpected metadata format: %q", got)
	}
}

func TestFormatOptionalString(t *testing.T) {
	if got := FormatOptionalString(nil); got != "-" {
		t.Errorf("expected '-' for nil pointer, got %q", got)
	}
	empty := ""
	if got := FormatOptionalString(&empty); got != "-" {
		t.Errorf("expected '-' for empty string, got %q", got)
	}
	val := "hello"
	if got := FormatOptionalString(&val); got != "hello" {
		t.Errorf("got %q", got)
	}
}

func TestPrintError_WritesToStderr(t *testing.T) {
	// We cannot easily intercept os.Stderr without patching, but verify the
	// helper does not panic for common format strings.
	PrintError("something %s", "bad")
}

func TestPrintSuccessAndWarn(t *testing.T) {
	PrintSuccess("ok %d", 1)
	PrintWarn("careful %s", "now")
}
