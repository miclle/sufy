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
	"io"
	"math"
	"os"
	"strings"
	"testing"
)

// captureStdout redirects os.Stdout during fn and returns what was written.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()

	fn()
	_ = w.Close()
	return <-done
}

// captureStderr mirrors captureStdout but targets os.Stderr.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w
	defer func() { os.Stderr = orig }()

	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()

	fn()
	_ = w.Close()
	return <-done
}

func TestPrintJSON_Object(t *testing.T) {
	out := captureStdout(t, func() {
		PrintJSON(map[string]any{"name": "alice", "age": 30})
	})
	out = strings.TrimSpace(out)

	var decoded map[string]any
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}
	if decoded["name"] != "alice" {
		t.Errorf("unexpected name: %v", decoded["name"])
	}
	// Indented output: expect at least one two-space indent.
	if !strings.Contains(out, "\n  ") {
		t.Errorf("expected indented JSON, got %q", out)
	}
}

func TestPrintJSON_Slice(t *testing.T) {
	out := captureStdout(t, func() {
		PrintJSON([]int{1, 2, 3})
	})
	out = strings.TrimSpace(out)
	var decoded []int
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(decoded) != 3 || decoded[0] != 1 {
		t.Errorf("unexpected output: %v", decoded)
	}
}

func TestPrintJSON_MarshalFailureWritesStderr(t *testing.T) {
	// math.Inf cannot be encoded by encoding/json.
	var stdoutBuf string
	stderrBuf := captureStderr(t, func() {
		stdoutBuf = captureStdout(t, func() {
			PrintJSON(math.Inf(1))
		})
	})
	if !strings.Contains(stderrBuf, "Error: marshal JSON failed") {
		t.Errorf("expected marshal error on stderr, got %q", stderrBuf)
	}
	if strings.TrimSpace(stdoutBuf) != "" {
		t.Errorf("expected no stdout output on error, got %q", stdoutBuf)
	}
}
