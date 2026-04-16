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

//go:build integration

package sandbox

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// testClient builds a client for integration tests from environment variables.
func testClient(t *testing.T) *Client {
	t.Helper()

	apiKey := os.Getenv("SUFY_API_KEY")
	apiURL := os.Getenv("SUFY_BASE_URL")
	if apiKey == "" {
		t.Fatal("SUFY_API_KEY environment variable is required")
	}

	return New(&Config{
		APIKey:  apiKey,
		BaseURL: apiURL,
	})
}

func TestIntegrationListTemplates(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	templates, err := c.ListTemplates(ctx, nil)
	if err != nil {
		t.Fatalf("ListTemplates failed: %v", err)
	}
	t.Logf("total templates: %d", len(templates))
	for _, tmpl := range templates {
		t.Logf("  - %s (buildStatus=%s)", tmpl.TemplateID, tmpl.BuildStatus)
	}
}

func TestIntegrationListSandboxes(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sandboxes, err := c.List(ctx, nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	t.Logf("total sandboxes: %d", len(sandboxes))
	for _, sb := range sandboxes {
		t.Logf("  - %s (template=%s)", sb.SandboxID, sb.TemplateID)
	}
}

func TestIntegrationSandboxLifecycle(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// 1. fetch available templates
	templates, err := c.ListTemplates(ctx, nil)
	if err != nil {
		t.Fatalf("ListTemplates failed: %v", err)
	}

	var templateID string
	for _, tmpl := range templates {
		if tmpl.BuildStatus == BuildStatusReady || tmpl.BuildStatus == BuildStatusUploaded {
			templateID = tmpl.TemplateID
			break
		}
	}
	if templateID == "" {
		t.Skip("no available template, skipping lifecycle test")
	}
	t.Logf("using template: %s", templateID)

	// 2. create the sandbox and wait until ready
	timeout := int32(60)
	sb, info, err := c.CreateAndWait(ctx, CreateParams{
		TemplateID: templateID,
		Timeout:    &timeout,
	}, WithPollInterval(2*time.Second))
	if err != nil {
		t.Fatalf("CreateAndWait failed: %v", err)
	}
	t.Logf("sandbox created: %s (state=%s)", sb.ID(), info.State)

	// ensure the sandbox is cleaned up at the end of the test
	killed := false
	defer func() {
		if killed {
			return
		}
		killCtx, killCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer killCancel()
		if err := sb.Kill(killCtx); err != nil {
			t.Logf("cleanup sandbox %s failed: %v", sb.ID(), err)
		} else {
			t.Logf("sandbox %s cleaned up", sb.ID())
		}
	}()

	// 3. check running status
	running, err := sb.IsRunning(ctx)
	if err != nil {
		t.Fatalf("IsRunning failed: %v", err)
	}
	if !running {
		t.Fatal("sandbox should be running")
	}

	// 4. fetch detailed info
	detail, err := sb.GetInfo(ctx)
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}
	t.Logf("sandbox detail: state=%s, templateID=%s, cpuCount=%d, memoryMB=%d",
		detail.State, detail.TemplateID, detail.CPUCount, detail.MemoryMB)

	// 5. update the timeout
	if err := sb.SetTimeout(ctx, 120*time.Second); err != nil {
		t.Fatalf("SetTimeout failed: %v", err)
	}
	t.Log("timeout updated to 120s")

	// 6. confirm the new sandbox is visible in the list
	sandboxes, err := c.List(ctx, nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	found := false
	for _, s := range sandboxes {
		if s.SandboxID == sb.ID() {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("newly created sandbox is missing from the list")
	}

	// 7. terminate the sandbox
	if err := sb.Kill(ctx); err != nil {
		t.Fatalf("Kill failed: %v", err)
	}
	killed = true
	t.Log("sandbox terminated")
}

// createTestSandbox creates a sandbox for envd integration tests and waits until it is ready.
func createTestSandbox(t *testing.T, c *Client, ctx context.Context) *Sandbox {
	t.Helper()

	templates, err := c.ListTemplates(ctx, nil)
	if err != nil {
		t.Fatalf("ListTemplates failed: %v", err)
	}

	var templateID string
	for _, tmpl := range templates {
		if tmpl.BuildStatus == BuildStatusReady || tmpl.BuildStatus == BuildStatusUploaded {
			templateID = tmpl.TemplateID
			break
		}
	}
	if templateID == "" {
		t.Skip("no available template, skipping test")
	}

	timeout := int32(120)
	sb, _, err := c.CreateAndWait(ctx, CreateParams{
		TemplateID: templateID,
		Timeout:    &timeout,
	}, WithPollInterval(2*time.Second))
	if err != nil {
		t.Fatalf("CreateAndWait failed: %v", err)
	}

	t.Cleanup(func() {
		killCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := sb.Kill(killCtx); err != nil {
			t.Logf("cleanup sandbox %s failed: %v", sb.ID(), err)
		}
	})

	return sb
}

func TestIntegrationFilesWriteRead(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sb := createTestSandbox(t, c, ctx)
	t.Logf("sandbox: %s", sb.ID())

	// write file
	content := []byte("hello sandbox\n")
	_, err := sb.Files().Write(ctx, "/tmp/test-file.txt", content)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	t.Log("file written")

	// read file
	got, err := sb.Files().Read(ctx, "/tmp/test-file.txt")
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("file content mismatch: got %q, want %q", string(got), string(content))
	}
	t.Log("file read content matches")
}

func TestIntegrationCommandsRun(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sb := createTestSandbox(t, c, ctx)
	t.Logf("sandbox: %s", sb.ID())

	// run a simple command
	result, err := sb.Commands().Run(ctx, "echo hello world")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	t.Logf("command result: exitCode=%d, stdout=%q, stderr=%q", result.ExitCode, result.Stdout, result.Stderr)
	if result.ExitCode != 0 {
		t.Fatalf("command exit code %d, want 0", result.ExitCode)
	}
	if result.Stdout != "hello world\n" {
		t.Fatalf("stdout = %q, want %q", result.Stdout, "hello world\n")
	}
}

func TestIntegrationGetHost(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sb := createTestSandbox(t, c, ctx)

	host := sb.GetHost(8080)
	if host == "" {
		t.Fatal("GetHost returned empty string")
	}
	t.Logf("GetHost(8080) = %s", host)

	// verify format: {port}-{sandboxID}.{domain}
	expected := "8080-" + sb.ID()
	if len(host) < len(expected) || host[:len(expected)] != expected {
		t.Fatalf("GetHost format mismatch: got %q, want prefix %q", host, expected)
	}
}

func TestIntegrationFilesystemOperations(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sb := createTestSandbox(t, c, ctx)
	t.Logf("sandbox: %s", sb.ID())

	fs := sb.Files()

	// create directory
	dirInfo, err := fs.MakeDir(ctx, "/tmp/test-dir")
	if err != nil {
		t.Fatalf("MakeDir failed: %v", err)
	}
	t.Logf("directory created: %s (type=%s)", dirInfo.Path, dirInfo.Type)

	// write file
	_, err = fs.Write(ctx, "/tmp/test-dir/hello.txt", []byte("hello"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// list directory
	entries, err := fs.List(ctx, "/tmp/test-dir")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	t.Logf("directory entries: %d", len(entries))

	// file existence
	exists, err := fs.Exists(ctx, "/tmp/test-dir/hello.txt")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Fatal("file should exist")
	}

	// fetch file info
	info, err := fs.GetInfo(ctx, "/tmp/test-dir/hello.txt")
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}
	t.Logf("file info: name=%s, size=%d, type=%s", info.Name, info.Size, info.Type)

	// rename
	newInfo, err := fs.Rename(ctx, "/tmp/test-dir/hello.txt", "/tmp/test-dir/renamed.txt")
	if err != nil {
		t.Fatalf("Rename failed: %v", err)
	}
	t.Logf("renamed: %s -> %s", "/tmp/test-dir/hello.txt", newInfo.Path)

	// delete
	if err := fs.Remove(ctx, "/tmp/test-dir/renamed.txt"); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
	t.Log("file deleted")

	// verify file no longer exists
	exists, err = fs.Exists(ctx, "/tmp/test-dir/renamed.txt")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Fatal("file should no longer exist")
	}
}

func TestIntegrationUploadDownload(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sb := createTestSandbox(t, c, ctx)
	t.Logf("sandbox: %s", sb.ID())

	// write file via Files().Write()
	content := []byte("upload test content\n")
	_, err := sb.Files().Write(ctx, "/tmp/uploaded.txt", content)
	if err != nil {
		t.Fatalf("Files().Write failed: %v", err)
	}
	t.Log("file written")

	// read file via Files().Read()
	got, err := sb.Files().Read(ctx, "/tmp/uploaded.txt")
	if err != nil {
		t.Fatalf("Files().Read failed: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("file content mismatch: got %q, want %q", string(got), string(content))
	}
	t.Log("file read content matches")
}

func TestIntegrationWriteFiles(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sb := createTestSandbox(t, c, ctx)
	t.Logf("sandbox: %s", sb.ID())

	files := []WriteEntry{
		{Path: "/tmp/batch-1.txt", Data: []byte("content one")},
		{Path: "/tmp/batch-2.txt", Data: []byte("content two")},
		{Path: "/tmp/batch-3.txt", Data: []byte("content three")},
	}

	infos, err := sb.Files().WriteFiles(ctx, files)
	if err != nil {
		t.Fatalf("WriteFiles failed: %v", err)
	}
	if len(infos) != 3 {
		t.Fatalf("WriteFiles returned %d results, want 3", len(infos))
	}

	// read back each file and verify content
	for i, f := range files {
		got, err := sb.Files().Read(ctx, f.Path)
		if err != nil {
			t.Fatalf("Read %s failed: %v", f.Path, err)
		}
		if string(got) != string(f.Data) {
			t.Fatalf("file %s content mismatch: got %q, want %q", f.Path, string(got), string(f.Data))
		}
		t.Logf("file %d (%s) verified: name=%s, size=%d", i, f.Path, infos[i].Name, infos[i].Size)
	}
}

func TestIntegrationReadText(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sb := createTestSandbox(t, c, ctx)
	t.Logf("sandbox: %s", sb.ID())

	content := "hello read text\n"
	_, err := sb.Files().Write(ctx, "/tmp/readtext.txt", []byte(content))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	got, err := sb.Files().ReadText(ctx, "/tmp/readtext.txt")
	if err != nil {
		t.Fatalf("ReadText failed: %v", err)
	}
	if got != content {
		t.Fatalf("ReadText content mismatch: got %q, want %q", got, content)
	}
	t.Log("ReadText verified")
}

func TestIntegrationReadStream(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sb := createTestSandbox(t, c, ctx)
	t.Logf("sandbox: %s", sb.ID())

	content := []byte("hello read stream\n")
	_, err := sb.Files().Write(ctx, "/tmp/readstream.txt", content)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	rc, err := sb.Files().ReadStream(ctx, "/tmp/readstream.txt")
	if err != nil {
		t.Fatalf("ReadStream failed: %v", err)
	}
	defer rc.Close()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read stream failed: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("ReadStream content mismatch: got %q, want %q", string(got), string(content))
	}
	t.Log("ReadStream verified")
}

// --- Commands async execution and process management ---

func TestIntegrationCommandsStartWait(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sb := createTestSandbox(t, c, ctx)
	t.Logf("sandbox: %s", sb.ID())

	handle, err := sb.Commands().Start(ctx, "sleep 1 && echo done")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	result, err := handle.Wait()
	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}

	if handle.PID() == 0 {
		t.Fatal("PID should be greater than 0")
	}
	if result.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", result.ExitCode)
	}
	if !strings.Contains(result.Stdout, "done") {
		t.Fatalf("Stdout = %q, want to contain 'done'", result.Stdout)
	}
	t.Logf("Start/Wait verified: PID=%d, ExitCode=%d, Stdout=%q", handle.PID(), result.ExitCode, result.Stdout)
}

func TestIntegrationCommandsKill(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sb := createTestSandbox(t, c, ctx)
	t.Logf("sandbox: %s", sb.ID())

	handle, err := sb.Commands().Start(ctx, "sleep 300")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// wait for PID to be assigned
	if _, err := handle.WaitPID(ctx); err != nil {
		t.Fatalf("WaitPID failed: %v", err)
	}
	t.Logf("process started: PID=%d", handle.PID())

	// kill process
	if err := sb.Commands().Kill(ctx, handle.PID()); err != nil {
		t.Fatalf("Kill failed: %v", err)
	}

	result, err := handle.Wait()
	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}

	if result.ExitCode == 0 {
		t.Fatal("killed process ExitCode should not be 0")
	}
	t.Logf("Kill verified: ExitCode=%d", result.ExitCode)
}

func TestIntegrationCommandsList(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sb := createTestSandbox(t, c, ctx)
	t.Logf("sandbox: %s", sb.ID())

	handle, err := sb.Commands().Start(ctx, "sleep 300")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// wait for PID to be assigned
	if _, err := handle.WaitPID(ctx); err != nil {
		t.Fatalf("WaitPID failed: %v", err)
	}
	t.Logf("process started: PID=%d", handle.PID())

	// list processes
	infos, err := sb.Commands().List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	found := false
	for _, info := range infos {
		if info.PID == handle.PID() {
			found = true
			t.Logf("found process: PID=%d, Cmd=%s, Args=%v", info.PID, info.Cmd, info.Args)
			break
		}
	}
	if !found {
		t.Fatalf("PID=%d not found in process list, total %d processes", handle.PID(), len(infos))
	}

	// cleanup
	_ = handle.Kill(ctx)
	_, _ = handle.Wait()
	t.Log("List verified")
}

func TestIntegrationCommandsSendStdin(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sb := createTestSandbox(t, c, ctx)
	t.Logf("sandbox: %s", sb.ID())

	// start a long-running sleep (stdin is disabled by default; SendStdin just verifies the RPC call succeeds)
	handle, err := sb.Commands().Start(ctx, "sleep 300")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// wait for PID to be assigned
	if _, err := handle.WaitPID(ctx); err != nil {
		t.Fatalf("WaitPID failed: %v", err)
	}

	// send stdin (stdin is disabled, the server returns an error; verify the error message is as expected)
	err = sb.Commands().SendStdin(ctx, handle.PID(), []byte("hello\n"))
	if err != nil {
		// when stdin is disabled, the server returns a "stdin not enabled" error as expected
		if strings.Contains(err.Error(), "stdin not enabled") {
			t.Logf("SendStdin returned expected error: %v", err)
		} else {
			t.Fatalf("SendStdin failed (unexpected error): %v", err)
		}
	} else {
		t.Log("SendStdin RPC call succeeded (data may have been discarded)")
	}

	// cleanup
	_ = handle.Kill(ctx)
	_, _ = handle.Wait()
	t.Log("SendStdin verified")
}

func TestIntegrationCommandsWithCallbacks(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sb := createTestSandbox(t, c, ctx)
	t.Logf("sandbox: %s", sb.ID())

	var (
		mu        sync.Mutex
		gotStdout []byte
		gotStderr []byte
	)

	result, err := sb.Commands().Run(ctx, "echo out && echo err >&2",
		WithOnStdout(func(data []byte) {
			mu.Lock()
			defer mu.Unlock()
			gotStdout = append(gotStdout, data...)
		}),
		WithOnStderr(func(data []byte) {
			mu.Lock()
			defer mu.Unlock()
			gotStderr = append(gotStderr, data...)
		}),
	)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", result.ExitCode)
	}

	mu.Lock()
	stdoutStr := string(gotStdout)
	stderrStr := string(gotStderr)
	mu.Unlock()

	if !strings.Contains(stdoutStr, "out") {
		t.Fatalf("Stdout callback did not receive expected data: %q", stdoutStr)
	}
	if !strings.Contains(stderrStr, "err") {
		t.Fatalf("Stderr callback did not receive expected data: %q", stderrStr)
	}
	t.Logf("callbacks verified: stdout=%q, stderr=%q", stdoutStr, stderrStr)
}

func TestIntegrationCommandsWithOptions(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sb := createTestSandbox(t, c, ctx)
	t.Logf("sandbox: %s", sb.ID())

	// test WithCwd
	result, err := sb.Commands().Run(ctx, "pwd", WithCwd("/tmp"))
	if err != nil {
		t.Fatalf("Run with WithCwd failed: %v", err)
	}
	if !strings.Contains(result.Stdout, "/tmp") {
		t.Fatalf("WithCwd: Stdout = %q, want to contain '/tmp'", result.Stdout)
	}
	t.Logf("WithCwd verified: Stdout=%q", result.Stdout)

	// test WithEnvs
	result, err = sb.Commands().Run(ctx, "echo $FOO", WithEnvs(map[string]string{"FOO": "BAR"}))
	if err != nil {
		t.Fatalf("Run with WithEnvs failed: %v", err)
	}
	if !strings.Contains(result.Stdout, "BAR") {
		t.Fatalf("WithEnvs: Stdout = %q, want to contain 'BAR'", result.Stdout)
	}
	t.Logf("WithEnvs verified: Stdout=%q", result.Stdout)
}

// --- PTY interaction ---

func TestIntegrationPtyCreateAndKill(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sb := createTestSandbox(t, c, ctx)
	t.Logf("sandbox: %s", sb.ID())

	var (
		mu     sync.Mutex
		output []byte
	)

	handle, err := sb.Pty().Create(ctx, PtySize{Cols: 80, Rows: 24},
		WithOnStdout(func(data []byte) {
			mu.Lock()
			defer mu.Unlock()
			output = append(output, data...)
		}),
	)
	if err != nil {
		t.Fatalf("Pty.Create failed: %v", err)
	}

	// wait for PID to be assigned
	if _, err := handle.WaitPID(ctx); err != nil {
		t.Fatalf("WaitPID failed: %v", err)
	}
	t.Logf("PTY created: PID=%d", handle.PID())

	// wait for some PTY output
	time.Sleep(2 * time.Second)

	// Kill PTY
	if err := sb.Pty().Kill(ctx, handle.PID()); err != nil {
		t.Fatalf("Pty.Kill failed: %v", err)
	}

	result, err := handle.Wait()
	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}

	mu.Lock()
	outputStr := string(output)
	mu.Unlock()

	t.Logf("PTY output length: %d bytes, ExitCode=%d", len(outputStr), result.ExitCode)
	if len(outputStr) == 0 {
		t.Log("warning: no PTY output received (the bash prompt may be empty in some environments)")
	}
	t.Log("Pty.Create/Kill verified")
}

func TestIntegrationPtySendInput(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sb := createTestSandbox(t, c, ctx)
	t.Logf("sandbox: %s", sb.ID())

	var (
		mu     sync.Mutex
		output []byte
	)

	handle, err := sb.Pty().Create(ctx, PtySize{Cols: 80, Rows: 24},
		WithOnStdout(func(data []byte) {
			mu.Lock()
			defer mu.Unlock()
			output = append(output, data...)
		}),
	)
	if err != nil {
		t.Fatalf("Pty.Create failed: %v", err)
	}

	// wait for PID to be assigned and let the shell initialize
	if _, err := handle.WaitPID(ctx); err != nil {
		t.Fatalf("WaitPID failed: %v", err)
	}
	time.Sleep(2 * time.Second)

	// send input
	if err := sb.Pty().SendInput(ctx, handle.PID(), []byte("echo pty-test\n")); err != nil {
		t.Fatalf("Pty.SendInput failed: %v", err)
	}

	// wait for output
	deadline := time.After(10 * time.Second)
	for {
		mu.Lock()
		has := strings.Contains(string(output), "pty-test")
		mu.Unlock()
		if has {
			break
		}
		select {
		case <-deadline:
			mu.Lock()
			t.Fatalf("timed out waiting for PTY output, received: %q", string(output))
			mu.Unlock()
		default:
			time.Sleep(200 * time.Millisecond)
		}
	}

	// cleanup
	_ = sb.Pty().Kill(ctx, handle.PID())
	_, _ = handle.Wait()

	mu.Lock()
	t.Logf("PTY output: %q", string(output))
	mu.Unlock()
	t.Log("Pty.SendInput verified")
}

func TestIntegrationPtyResize(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sb := createTestSandbox(t, c, ctx)
	t.Logf("sandbox: %s", sb.ID())

	handle, err := sb.Pty().Create(ctx, PtySize{Cols: 80, Rows: 24})
	if err != nil {
		t.Fatalf("Pty.Create failed: %v", err)
	}

	// wait for PID to be assigned
	if _, err := handle.WaitPID(ctx); err != nil {
		t.Fatalf("WaitPID failed: %v", err)
	}

	// Resize
	if err := sb.Pty().Resize(ctx, handle.PID(), PtySize{Cols: 200, Rows: 50}); err != nil {
		t.Fatalf("Pty.Resize failed: %v", err)
	}
	t.Log("Resize called successfully")

	// cleanup
	_ = sb.Pty().Kill(ctx, handle.PID())
	_, _ = handle.Wait()
	t.Log("Pty.Resize verified")
}

// --- Filesystem.WatchDir ---

func TestIntegrationWatchDir(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sb := createTestSandbox(t, c, ctx)
	t.Logf("sandbox: %s", sb.ID())

	// create the directory to watch
	watchPath := "/tmp/watch-test-" + fmt.Sprintf("%d", time.Now().UnixNano())
	_, err := sb.Files().MakeDir(ctx, watchPath)
	if err != nil {
		t.Fatalf("MakeDir failed: %v", err)
	}

	// start WatchDir
	watcher, err := sb.Files().WatchDir(ctx, watchPath, WithRecursive(true))
	if err != nil {
		t.Fatalf("WatchDir failed: %v", err)
	}
	defer watcher.Stop()

	// wait for the watcher to become ready
	time.Sleep(1 * time.Second)

	// write a file in another goroutine to trigger an event
	go func() {
		_, _ = sb.Files().Write(ctx, watchPath+"/event-file.txt", []byte("watch me"))
	}()

	// collect events
	var events []FilesystemEvent
	deadline := time.After(15 * time.Second)
	for {
		select {
		case ev, ok := <-watcher.Events():
			if !ok {
				goto done
			}
			events = append(events, ev)
			t.Logf("event received: Name=%s, Type=%s", ev.Name, ev.Type)
			// at least one event is enough
			if len(events) >= 1 {
				goto done
			}
		case <-deadline:
			goto done
		}
	}
done:

	if len(events) == 0 {
		t.Fatal("no filesystem event received")
	}
	t.Logf("WatchDir verified: %d events received", len(events))
}

func TestIntegrationMetadata(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	templates, err := c.ListTemplates(ctx, nil)
	if err != nil {
		t.Fatalf("ListTemplates failed: %v", err)
	}

	var templateID string
	for _, tmpl := range templates {
		if tmpl.BuildStatus == BuildStatusReady || tmpl.BuildStatus == BuildStatusUploaded {
			templateID = tmpl.TemplateID
			break
		}
	}
	if templateID == "" {
		t.Skip("no available template, skipping test")
	}

	timeout := int32(60)
	meta := Metadata{"env": "test", "team": "backend"}
	sb, _, err := c.CreateAndWait(ctx, CreateParams{
		TemplateID: templateID,
		Timeout:    &timeout,
		Metadata:   &meta,
	}, WithPollInterval(2*time.Second))
	if err != nil {
		t.Fatalf("CreateAndWait failed: %v", err)
	}
	t.Cleanup(func() {
		killCtx, killCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer killCancel()
		if err := sb.Kill(killCtx); err != nil {
			t.Logf("cleanup sandbox %s failed: %v", sb.ID(), err)
		}
	})

	info, err := sb.GetInfo(ctx)
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}

	if info.Metadata == nil {
		t.Fatal("Metadata should not be nil")
	}
	got := *info.Metadata
	if got["env"] != "test" {
		t.Errorf("Metadata[env] = %q, want %q", got["env"], "test")
	}
	if got["team"] != "backend" {
		t.Errorf("Metadata[team] = %q, want %q", got["team"], "backend")
	}
	t.Logf("Metadata verified: %v", got)
}
