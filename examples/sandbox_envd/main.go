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

package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/sufy-dev/sufy/sandbox"
)

func main() {
	apiKey := os.Getenv("SUFY_API_KEY")
	if apiKey == "" {
		log.Fatal("SUFY_API_KEY environment variable is required")
	}

	c := sandbox.New(&sandbox.Config{
		APIKey:  apiKey,
		BaseURL: os.Getenv("SUFY_BASE_URL"),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// 1. Pick the first usable template.
	templates, err := c.ListTemplates(ctx, nil)
	if err != nil {
		log.Fatalf("ListTemplates failed: %v", err)
	}

	var templateID string
	for _, tmpl := range templates {
		if tmpl.BuildStatus == sandbox.BuildStatusReady || tmpl.BuildStatus == sandbox.BuildStatusUploaded {
			templateID = tmpl.TemplateID
			break
		}
	}
	if templateID == "" {
		log.Fatal("no successfully built template available")
	}
	fmt.Printf("using template: %s\n", templateID)

	// 2. Create the sandbox and wait for it to become ready, with Metadata and NetworkConfig.
	timeout := int32(120)
	meta := sandbox.Metadata{"env": "dev", "team": "backend"}
	network := sandbox.NetworkConfig{
		AllowPublicTraffic: boolPtr(true),
	}
	sb, info, err := c.CreateAndWait(ctx, sandbox.CreateParams{
		TemplateID: templateID,
		Timeout:    &timeout,
		Metadata:   &meta,
		Network:    &network,
	}, sandbox.WithPollInterval(2*time.Second))
	if err != nil {
		log.Fatalf("CreateAndWait failed: %v", err)
	}
	fmt.Printf("sandbox ready: %s\n", sb.ID())

	// Verify Metadata.
	if info.Metadata != nil {
		fmt.Printf("Metadata: %v\n", *info.Metadata)
	}

	defer func() {
		killCtx, killCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer killCancel()
		if err := sb.Kill(killCtx); err != nil {
			log.Printf("Kill failed: %v", err)
		} else {
			fmt.Printf("sandbox %s terminated\n", sb.ID())
		}
	}()

	// 3. Fetch the port host.
	host := sb.GetHost(8080)
	fmt.Printf("port 8080 host: %s\n", host)

	// 4. Filesystem operations.
	fmt.Println("\n--- filesystem operations ---")

	// Write file.
	_, err = sb.Files().Write(ctx, "/tmp/hello.txt", []byte("Hello from Go SDK!\n"))
	if err != nil {
		log.Fatalf("Write failed: %v", err)
	}
	fmt.Println("file written: /tmp/hello.txt")

	// Read file.
	content, err := sb.Files().Read(ctx, "/tmp/hello.txt")
	if err != nil {
		log.Fatalf("Read failed: %v", err)
	}
	fmt.Printf("file content: %s", string(content))

	// Create directory.
	_, err = sb.Files().MakeDir(ctx, "/tmp/mydir")
	if err != nil {
		log.Fatalf("MakeDir failed: %v", err)
	}
	fmt.Println("directory created: /tmp/mydir")

	// List directory.
	entries, err := sb.Files().List(ctx, "/tmp")
	if err != nil {
		log.Fatalf("List failed: %v", err)
	}
	fmt.Printf("/tmp directory contents (%d entries):\n", len(entries))
	for _, e := range entries {
		fmt.Printf("  %s %s (%s, %d bytes)\n", e.Type, e.Name, e.Permissions, e.Size)
	}

	// Batch write files.
	fmt.Println("\n--- batch write files ---")
	files := []sandbox.WriteEntry{
		{Path: "/tmp/batch-a.txt", Data: []byte("file A content")},
		{Path: "/tmp/batch-b.txt", Data: []byte("file B content")},
		{Path: "/tmp/batch-c.txt", Data: []byte("file C content")},
	}
	infos, err := sb.Files().WriteFiles(ctx, files)
	if err != nil {
		log.Fatalf("WriteFiles failed: %v", err)
	}
	for _, fi := range infos {
		fmt.Printf("written: %s (%d bytes)\n", fi.Path, fi.Size)
	}

	// ReadText — read a file as string.
	fmt.Println("\n--- ReadText ---")
	text, err := sb.Files().ReadText(ctx, "/tmp/batch-a.txt")
	if err != nil {
		log.Fatalf("ReadText failed: %v", err)
	}
	fmt.Printf("ReadText result: %q\n", text)

	// ReadStream — stream file content.
	fmt.Println("\n--- ReadStream ---")
	rc, err := sb.Files().ReadStream(ctx, "/tmp/batch-b.txt")
	if err != nil {
		log.Fatalf("ReadStream failed: %v", err)
	}
	streamData, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		log.Fatalf("stream read failed: %v", err)
	}
	fmt.Printf("ReadStream result: %q\n", string(streamData))

	// Exists — check whether a file exists.
	fmt.Println("\n--- Exists / GetInfo / Rename / Remove ---")
	exists, err := sb.Files().Exists(ctx, "/tmp/batch-a.txt")
	if err != nil {
		log.Fatalf("Exists failed: %v", err)
	}
	fmt.Printf("Exists(/tmp/batch-a.txt) = %v\n", exists)

	// GetInfo — fetch file metadata.
	fileInfo, err := sb.Files().GetInfo(ctx, "/tmp/batch-a.txt")
	if err != nil {
		log.Fatalf("GetInfo failed: %v", err)
	}
	fmt.Printf("GetInfo: name=%s, type=%s, size=%d, mode=%s\n",
		fileInfo.Name, fileInfo.Type, fileInfo.Size, fileInfo.Permissions)

	// Rename — rename a file.
	renamedInfo, err := sb.Files().Rename(ctx, "/tmp/batch-c.txt", "/tmp/batch-c-renamed.txt")
	if err != nil {
		log.Fatalf("Rename failed: %v", err)
	}
	fmt.Printf("Rename: %s -> %s\n", "/tmp/batch-c.txt", renamedInfo.Path)

	// Remove — delete a file.
	if err := sb.Files().Remove(ctx, "/tmp/batch-c-renamed.txt"); err != nil {
		log.Fatalf("Remove failed: %v", err)
	}
	exists, err = sb.Files().Exists(ctx, "/tmp/batch-c-renamed.txt")
	if err != nil {
		log.Fatalf("Exists failed: %v", err)
	}
	fmt.Printf("after Remove, Exists(/tmp/batch-c-renamed.txt) = %v\n", exists)

	// WatchDir — watch a directory for changes.
	fmt.Println("\n--- WatchDir ---")

	// Create the directory to watch.
	_, err = sb.Files().MakeDir(ctx, "/tmp/watch-test")
	if err != nil {
		log.Fatalf("MakeDir failed: %v", err)
	}

	// Start watching the directory.
	wh, err := sb.Files().WatchDir(ctx, "/tmp/watch-test", sandbox.WithRecursive(true))
	if err != nil {
		log.Fatalf("WatchDir failed: %v", err)
	}
	fmt.Println("watching /tmp/watch-test (recursive)")

	// Trigger a file change within the watched directory.
	_, err = sb.Files().Write(ctx, "/tmp/watch-test/watch-file.txt", []byte("watched content"))
	if err != nil {
		log.Fatalf("Write failed: %v", err)
	}
	fmt.Println("written: /tmp/watch-test/watch-file.txt")

	// Collect events (up to 3 seconds).
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()
	eventCount := 0
loop:
	for {
		select {
		case ev, ok := <-wh.Events():
			if !ok {
				break loop
			}
			eventCount++
			fmt.Printf("  event: type=%s, name=%s\n", ev.Type, ev.Name)
		case <-timer.C:
			break loop
		}
	}
	fmt.Printf("received %d events\n", eventCount)

	// Stop watching.
	wh.Stop()
	if err := wh.Err(); err != nil {
		fmt.Printf("watch error: %v\n", err)
	} else {
		fmt.Println("watch stopped")
	}

	// 5. Command execution.
	fmt.Println("\n--- command execution ---")

	result, err := sb.Commands().Run(ctx, "echo hello world")
	if err != nil {
		log.Fatalf("Run failed: %v", err)
	}
	fmt.Printf("command: echo hello world\n")
	fmt.Printf("exit code: %d\n", result.ExitCode)
	fmt.Printf("stdout: %s", result.Stdout)

	// Command with environment variables.
	result, err = sb.Commands().Run(ctx, "echo $MY_VAR",
		sandbox.WithEnvs(map[string]string{"MY_VAR": "sandbox-value"}),
	)
	if err != nil {
		log.Fatalf("Run failed: %v", err)
	}
	fmt.Printf("command: echo $MY_VAR (MY_VAR=sandbox-value)\n")
	fmt.Printf("stdout: %s", result.Stdout)

	// WithCwd — specify the working directory.
	result, err = sb.Commands().Run(ctx, "pwd", sandbox.WithCwd("/tmp"))
	if err != nil {
		log.Fatalf("Run failed: %v", err)
	}
	fmt.Printf("command: pwd (cwd=/tmp)\nstdout: %s", result.Stdout)

	// WithTimeout — command timeout.
	result, err = sb.Commands().Run(ctx, "echo fast", sandbox.WithTimeout(5*time.Second))
	if err != nil {
		log.Fatalf("Run failed: %v", err)
	}
	fmt.Printf("command: echo fast (timeout=5s)\nstdout: %s", result.Stdout)

	// WithOnStdout / WithOnStderr — realtime output callbacks.
	fmt.Println("\n--- realtime output callbacks ---")
	var stdoutChunks, stderrChunks int
	result, err = sb.Commands().Run(ctx, "echo out-line && echo err-line >&2",
		sandbox.WithOnStdout(func(data []byte) { stdoutChunks++ }),
		sandbox.WithOnStderr(func(data []byte) { stderrChunks++ }),
	)
	if err != nil {
		log.Fatalf("Run failed: %v", err)
	}
	fmt.Printf("stdout chunks: %d, stderr chunks: %d\n", stdoutChunks, stderrChunks)
	fmt.Printf("stdout: %sstderr: %s", result.Stdout, result.Stderr)

	// Start / List / Kill — background process management.
	fmt.Println("\n--- background commands (Start / List / Kill) ---")
	handle, err := sb.Commands().Start(ctx, "sleep 30", sandbox.WithTag("bg-sleep"))
	if err != nil {
		log.Fatalf("Start failed: %v", err)
	}
	// Wait for PID assignment.
	if _, err := handle.WaitPID(ctx); err != nil {
		log.Fatalf("WaitPID failed: %v", err)
	}
	fmt.Printf("background command started: PID=%d\n", handle.PID())

	// List — list running processes.
	processes, err := sb.Commands().List(ctx)
	if err != nil {
		log.Fatalf("List failed: %v", err)
	}
	fmt.Printf("running processes (%d):\n", len(processes))
	for _, p := range processes {
		tag := "<none>"
		if p.Tag != nil {
			tag = *p.Tag
		}
		fmt.Printf("  PID=%d, cmd=%s, tag=%s\n", p.PID, p.Cmd, tag)
	}

	// Kill — terminate the background process.
	if err := sb.Commands().Kill(ctx, handle.PID()); err != nil {
		log.Fatalf("Kill failed: %v", err)
	}
	fmt.Printf("process PID=%d terminated\n", handle.PID())

	// 6. Download / upload URLs.
	fmt.Println("\n--- file URLs ---")
	downloadURL := sb.DownloadURL("/tmp/hello.txt")
	fmt.Printf("download URL: %s\n", downloadURL)

	uploadURL := sb.UploadURL("/tmp/upload.txt")
	fmt.Printf("upload URL: %s\n", uploadURL)

	// Upload / download through Files().Write() / Files().Read().
	writeContent := []byte("uploaded via Files().Write()\n")
	if _, err := sb.Files().Write(ctx, "/tmp/upload-test.txt", writeContent); err != nil {
		log.Fatalf("Files().Write failed: %v", err)
	}
	fmt.Println("Files().Write succeeded: /tmp/upload-test.txt")

	readContent, err := sb.Files().Read(ctx, "/tmp/upload-test.txt")
	if err != nil {
		log.Fatalf("Files().Read failed: %v", err)
	}
	fmt.Printf("Files().Read result: %q\n", string(readContent))

	// 7. PTY terminal.
	fmt.Println("\n--- PTY terminal ---")

	// Create — create a PTY session.
	var ptyOutput []byte
	ptyHandle, err := sb.Pty().Create(ctx, sandbox.PtySize{Cols: 80, Rows: 24},
		sandbox.WithOnStdout(func(data []byte) {
			ptyOutput = append(ptyOutput, data...)
		}),
	)
	if err != nil {
		log.Fatalf("Pty.Create failed: %v", err)
	}
	if _, err := ptyHandle.WaitPID(ctx); err != nil {
		log.Fatalf("WaitPID failed: %v", err)
	}
	fmt.Printf("PTY created: PID=%d\n", ptyHandle.PID())

	// SendInput — send input to the PTY.
	if err := sb.Pty().SendInput(ctx, ptyHandle.PID(), []byte("echo pty-hello\n")); err != nil {
		log.Fatalf("Pty.SendInput failed: %v", err)
	}
	time.Sleep(500 * time.Millisecond)
	fmt.Printf("PTY output snippet: %q\n", truncate(string(ptyOutput), 200))

	// Resize — resize the terminal.
	if err := sb.Pty().Resize(ctx, ptyHandle.PID(), sandbox.PtySize{Cols: 120, Rows: 40}); err != nil {
		log.Fatalf("Pty.Resize failed: %v", err)
	}
	fmt.Println("PTY resized to 120x40")

	// Kill — terminate the PTY session.
	if err := sb.Pty().Kill(ctx, ptyHandle.PID()); err != nil {
		log.Fatalf("Pty.Kill failed: %v", err)
	}
	fmt.Printf("PTY PID=%d terminated\n", ptyHandle.PID())
}

// boolPtr returns a pointer to the given bool value.
func boolPtr(v bool) *bool { return &v }

// truncate shortens a string to at most maxLen runes.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
