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

// Package sandbox provides the Go SDK for the SUFY Sandbox service, used to
// manage securely isolated cloud sandbox environments.
//
// The Sandbox service is a runtime infrastructure designed for AI Agent
// scenarios. It provides securely isolated cloud environments for running
// AI-generated code. System-level isolation ensures code execution cannot
// illegally access or tamper with the host. Sandboxes boot in under 200ms,
// live for 5 minutes by default (up to 1 hour), and can be paused or resumed
// to persist filesystem and memory state.
//
// # Core concepts
//
//   - Sandbox: an isolated cloud execution environment (lightweight
//     virtualization) with three states: running, paused and killed.
//   - Template: a prebuilt sandbox definition containing base image,
//     dependencies, files and start command, enabling sub-second boot.
//   - envd: the agent daemon running inside a sandbox, exposing process
//     management, filesystem operations and PTY terminal services over
//     ConnectRPC.
//
// # Quick start
//
// Create a client and launch a sandbox:
//
//	c := sandbox.New(&sandbox.Config{
//	    APIKey: os.Getenv("SUFY_API_KEY"),
//	})
//
//	timeout := int32(120)
//	sb, _, err := c.CreateAndWait(ctx, sandbox.CreateParams{
//	    TemplateID: "base",
//	    Timeout:    &timeout,
//	}, sandbox.WithPollInterval(2*time.Second))
//
//	defer sb.Kill(ctx)
//
// # Sandbox lifecycle
//
// Client provides sandbox creation, connection and listing:
//
//   - [Client.Create] / [Client.CreateAndWait]: create a sandbox (the latter
//     polls until the sandbox is ready).
//   - [Client.Connect]: connect to an existing sandbox, resuming it if paused.
//   - [Client.List]: list sandboxes, with optional state and metadata filters.
//
// Sandbox instances expose lifecycle management:
//
//   - [Sandbox.Kill]: terminate the sandbox.
//   - [Sandbox.Pause]: pause the sandbox (preserves filesystem and memory).
//   - [Sandbox.SetTimeout]: update the timeout.
//   - [Sandbox.Refresh]: extend the sandbox lifetime.
//   - [Sandbox.GetInfo]: query detailed sandbox state.
//   - [Sandbox.IsRunning]: check availability through the envd /health endpoint.
//   - [Sandbox.GetMetrics]: fetch CPU, memory and disk metrics.
//   - [Sandbox.GetLogs]: fetch sandbox logs.
//   - [Sandbox.WaitForReady]: poll until the sandbox enters the running state.
//
// # Command execution
//
// Run terminal commands inside the sandbox via [Sandbox.Commands]:
//
//	// synchronous
//	result, err := sb.Commands().Run(ctx, "echo hello",
//	    sandbox.WithEnvs(map[string]string{"MY_VAR": "value"}),
//	    sandbox.WithCwd("/tmp"),
//	    sandbox.WithTimeout(5*time.Second),
//	)
//	fmt.Println(result.Stdout)
//
//	// asynchronous (background command)
//	handle, err := sb.Commands().Start(ctx, "sleep 30", sandbox.WithTag("bg"))
//	handle.WaitPID(ctx)
//	sb.Commands().Kill(ctx, handle.PID())
//
// Commands support realtime output callbacks ([WithOnStdout] / [WithOnStderr]),
// background process management ([Commands.Start] / [Commands.List] /
// [Commands.Kill]), and stdin forwarding ([Commands.SendStdin]).
//
// # Filesystem operations
//
// Read and write files via [Sandbox.Files]:
//
//	// write and read
//	sb.Files().Write(ctx, "/tmp/hello.txt", []byte("Hello!"))
//	content, err := sb.Files().Read(ctx, "/tmp/hello.txt")
//
//	// batch write
//	sb.Files().WriteFiles(ctx, []sandbox.WriteEntry{
//	    {Path: "/tmp/a.txt", Data: []byte("content A")},
//	    {Path: "/tmp/b.txt", Data: []byte("content B")},
//	})
//
//	// directory operations
//	sb.Files().MakeDir(ctx, "/tmp/mydir")
//	entries, err := sb.Files().List(ctx, "/tmp")
//
//	// watch a directory
//	wh, err := sb.Files().WatchDir(ctx, "/tmp/watch", sandbox.WithRecursive(true))
//	for ev := range wh.Events() {
//	    fmt.Printf("event: %s %s\n", ev.Type, ev.Name)
//	}
//
// Additional helpers include [Filesystem.ReadText], [Filesystem.ReadStream],
// [Filesystem.Exists], [Filesystem.GetInfo], [Filesystem.Rename] and
// [Filesystem.Remove].
//
// # PTY terminal
//
// Create and manage pseudo-terminal sessions via [Sandbox.Pty]:
//
//	ptyHandle, err := sb.Pty().Create(ctx, sandbox.PtySize{Cols: 80, Rows: 24},
//	    sandbox.WithOnStdout(func(data []byte) { fmt.Print(string(data)) }),
//	)
//	sb.Pty().SendInput(ctx, ptyHandle.PID(), []byte("ls -la\n"))
//	sb.Pty().Resize(ctx, ptyHandle.PID(), sandbox.PtySize{Cols: 120, Rows: 40})
//	sb.Pty().Kill(ctx, ptyHandle.PID())
//
// # Template management
//
// Client provides full template lifecycle management:
//
//   - [Client.ListTemplates] / [Client.GetTemplate]: list and query templates.
//   - [Client.CreateTemplate]: create a template (returns templateID and buildID).
//   - [Client.UpdateTemplate] / [Client.DeleteTemplate]: update or delete a template.
//   - [Client.StartTemplateBuild] / [Client.WaitForBuild]: start a build and wait for completion.
//   - [Client.GetTemplateBuildStatus] / [Client.GetTemplateBuildLogs]: query build status and logs.
//   - [Client.AssignTemplateTags] / [Client.DeleteTemplateTags]: manage template tags.
//   - [Client.GetTemplateByAlias]: look up a template by alias.
//
// # Network access
//
// Sandboxes allow outbound Internet access by default. Outbound traffic rules
// can be configured through the Network field of [CreateParams]. Use
// [Sandbox.GetHost] to obtain the external domain that serves a specific
// sandbox port.
//
// # Poll options
//
// [Client.CreateAndWait], [Sandbox.WaitForReady] and [Client.WaitForBuild]
// accept [PollOption] values to customize polling behavior:
//
//   - [WithPollInterval]: set the poll interval.
//   - [WithBackoff]: enable exponential backoff.
//   - [WithOnPoll]: register a poll callback (useful for logging or progress).
package sandbox
