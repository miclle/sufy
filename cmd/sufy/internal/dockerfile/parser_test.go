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

package dockerfile

import (
	"reflect"
	"strings"
	"testing"
)

func TestParse_BasicInstructions(t *testing.T) {
	content := "FROM alpine:3.19\nRUN echo hello\nWORKDIR /app\n"
	result, err := Parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Instructions) != 3 {
		t.Fatalf("expected 3 instructions, got %d", len(result.Instructions))
	}
	if result.Instructions[0].Name != "FROM" || result.Instructions[0].Args != "alpine:3.19" {
		t.Errorf("unexpected FROM: %+v", result.Instructions[0])
	}
	if result.Instructions[1].Name != "RUN" || result.Instructions[1].Args != "echo hello" {
		t.Errorf("unexpected RUN: %+v", result.Instructions[1])
	}
	if result.EscapeToken != '\\' {
		t.Errorf("expected default escape '\\', got %q", result.EscapeToken)
	}
}

func TestParse_SkipsBlankAndComments(t *testing.T) {
	content := "\n# comment\nFROM a\n\n# another\nRUN b\n"
	result, err := Parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Instructions) != 2 {
		t.Fatalf("expected 2 instructions, got %d", len(result.Instructions))
	}
}

func TestParse_EscapeDirectiveBacktick(t *testing.T) {
	content := "# escape=`\nFROM a\nRUN one `\n  && two\n"
	result, err := Parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.EscapeToken != '`' {
		t.Errorf("expected backtick escape, got %q", result.EscapeToken)
	}
	// RUN should have both parts joined.
	var found bool
	for _, inst := range result.Instructions {
		if inst.Name == "RUN" && strings.Contains(inst.Args, "one") && strings.Contains(inst.Args, "two") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected joined RUN instruction, got %+v", result.Instructions)
	}
}

func TestParse_EscapeDirectiveBackslashDefault(t *testing.T) {
	content := "# syntax=docker/dockerfile:1\nFROM a\n"
	result, err := Parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.EscapeToken != '\\' {
		t.Errorf("expected default backslash escape, got %q", result.EscapeToken)
	}
}

func TestParse_StripsBOM(t *testing.T) {
	content := "\xef\xbb\xbfFROM alpine\n"
	result, err := Parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Instructions) != 1 || result.Instructions[0].Name != "FROM" {
		t.Errorf("BOM not stripped: %+v", result.Instructions)
	}
}

func TestParse_LineContinuation(t *testing.T) {
	content := "FROM a\nRUN echo hello \\\n  && echo world\n"
	result, err := Parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var run string
	for _, inst := range result.Instructions {
		if inst.Name == "RUN" {
			run = inst.Args
		}
	}
	if !strings.Contains(run, "hello") || !strings.Contains(run, "world") {
		t.Errorf("continuation not joined: %q", run)
	}
}

func TestParse_ContinuationSkipsCommentsInside(t *testing.T) {
	content := "FROM a\nRUN echo a \\\n# inline comment\n  && echo b\n"
	result, err := Parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var run string
	for _, inst := range result.Instructions {
		if inst.Name == "RUN" {
			run = inst.Args
		}
	}
	if strings.Contains(run, "inline comment") {
		t.Errorf("comment not skipped inside continuation: %q", run)
	}
}

func TestParse_CopyFlagsExtracted(t *testing.T) {
	content := "FROM a\nCOPY --chown=user:group --chmod=755 src dest\n"
	result, err := Parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	copyInst := result.Instructions[1]
	if copyInst.Flags["chown"] != "user:group" {
		t.Errorf("expected chown flag, got %v", copyInst.Flags)
	}
	if copyInst.Flags["chmod"] != "755" {
		t.Errorf("expected chmod flag, got %v", copyInst.Flags)
	}
	if copyInst.Args != "src dest" {
		t.Errorf("expected cleaned args, got %q", copyInst.Args)
	}
}

func TestParse_Heredoc(t *testing.T) {
	content := "FROM a\nRUN <<EOF\necho one\necho two\nEOF\n"
	result, err := Parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	runInst := result.Instructions[1]
	if runInst.Name != "RUN" {
		t.Fatalf("expected RUN, got %s", runInst.Name)
	}
	if runInst.Heredoc == "" {
		t.Errorf("expected heredoc body, got empty")
	}
	if !strings.Contains(runInst.Args, "echo one") || !strings.Contains(runInst.Args, "echo two") {
		t.Errorf("unexpected RUN args: %q", runInst.Args)
	}
}

func TestParse_HeredocUnterminated(t *testing.T) {
	content := "FROM a\nRUN <<EOF\necho hi\n"
	_, err := Parse(content)
	if err == nil {
		t.Fatalf("expected error for unterminated heredoc")
	}
	if !strings.Contains(err.Error(), "unterminated heredoc") {
		t.Errorf("expected unterminated heredoc error, got %v", err)
	}
}

func TestParse_UnknownInstructionWarns(t *testing.T) {
	content := "FROM a\nFOO bar baz\n"
	result, err := Parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Warnings) == 0 {
		t.Fatalf("expected warnings")
	}
	if !strings.Contains(result.Warnings[0], "unknown instruction") {
		t.Errorf("expected unknown instruction warning, got %q", result.Warnings[0])
	}
}

func TestParse_ONBUILDWarns(t *testing.T) {
	content := "FROM a\nONBUILD RUN echo hi\n"
	result, err := Parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Warnings) == 0 || !strings.Contains(result.Warnings[0], "ONBUILD") {
		t.Errorf("expected ONBUILD warning, got %v", result.Warnings)
	}
}

func TestParseEnvValues_KeyValueFormat(t *testing.T) {
	kvs, err := ParseEnvValues(`A=1 B="two words" C='three'`, '\\')
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{"A", "1", "B", "two words", "C", "three"}
	if !reflect.DeepEqual(kvs, expected) {
		t.Errorf("got %v, want %v", kvs, expected)
	}
}

func TestParseEnvValues_LegacyFormat(t *testing.T) {
	kvs, err := ParseEnvValues("FOO bar baz", '\\')
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(kvs, []string{"FOO", "bar baz"}) {
		t.Errorf("got %v", kvs)
	}
}

func TestParseEnvValues_Empty(t *testing.T) {
	if _, err := ParseEnvValues("", '\\'); err == nil {
		t.Errorf("expected error for empty ENV")
	}
}

func TestParseEnvValues_EscapeInDoubleQuote(t *testing.T) {
	kvs, err := ParseEnvValues(`A="hello\"world"`, '\\')
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kvs[1] != `hello"world` {
		t.Errorf("expected escaped quote, got %q", kvs[1])
	}
}

func TestParseCommand_ExecForm(t *testing.T) {
	got := ParseCommand(`["/bin/sh", "-c", "echo hi"]`)
	want := "/bin/sh -c 'echo hi'"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestParseCommand_ShellForm(t *testing.T) {
	got := ParseCommand("echo hello world")
	if got != "echo hello world" {
		t.Errorf("got %q", got)
	}
}

func TestParseCommand_Empty(t *testing.T) {
	if got := ParseCommand(""); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestStripHeredocMarkers(t *testing.T) {
	got := StripHeredocMarkers("src <<EOF dest")
	if got != "src dest" {
		t.Errorf("got %q", got)
	}
}

func TestStripHeredocMarkers_NoMarker(t *testing.T) {
	got := StripHeredocMarkers("src dest")
	if got != "src dest" {
		t.Errorf("got %q", got)
	}
}

func TestIsEscapedEscape(t *testing.T) {
	if !isEscapedEscape("a\\\\", '\\') == false {
		// "a\\\\" is literal a\\ (two backslashes) -> even count -> not escaped
	}
	// One trailing backslash consumed -> string before is "a\" -> odd count -> escaped
	if !isEscapedEscape("a\\", '\\') {
		t.Errorf("expected escaped, got false for single backslash")
	}
	// Two trailing backslashes consumed -> not escaped
	if isEscapedEscape("", '\\') {
		t.Errorf("expected not escaped for empty")
	}
}
