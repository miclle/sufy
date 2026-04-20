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
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	sdk "github.com/sufy-dev/sufy/sandbox"
)

// stdinChunkSize bounds how much stdin is forwarded per SendStdin RPC.
const stdinChunkSize = 64 * 1024

// ExecForeground runs a command in the foreground, forwarding stdout, stderr,
// stdin, SIGINT/SIGTERM, and the exit code. It calls os.Exit on completion.
func ExecForeground(ctx context.Context, sb *sdk.Sandbox, cmd string, opts []sdk.CommandOption) {
	opts = append(opts,
		sdk.WithOnStdout(func(data []byte) { os.Stdout.Write(data) }),
		sdk.WithOnStderr(func(data []byte) { os.Stderr.Write(data) }),
	)

	handle, err := sb.Commands().Start(ctx, cmd, opts...)
	if err != nil {
		PrintError("exec failed: %v", err)
		os.Exit(1)
	}

	pid, err := handle.WaitPID(ctx)
	if err != nil {
		PrintError("waiting for process start: %v", err)
		os.Exit(1)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for range sigCh {
			_ = sb.Commands().Kill(ctx, pid)
		}
	}()

	if IsPipedStdin() {
		go sendStdinToSandbox(ctx, sb, pid)
	}

	result, err := handle.Wait()
	signal.Stop(sigCh)
	if err != nil {
		PrintError("exec failed: %v", err)
		os.Exit(1)
	}
	if result.Error != "" {
		PrintError("%s", result.Error)
	}
	os.Exit(result.ExitCode)
}

// ExecBackground starts a command detached and prints its PID. When stdin is
// piped it forwards stdin to the remote process before returning.
func ExecBackground(ctx context.Context, sb *sdk.Sandbox, cmd string, opts []sdk.CommandOption) {
	handle, err := sb.Commands().Start(ctx, cmd, opts...)
	if err != nil {
		PrintError("exec failed: %v", err)
		return
	}

	pid, err := handle.WaitPID(ctx)
	if err != nil {
		PrintError("waiting for process start: %v", err)
		return
	}
	fmt.Printf("PID: %d\n", pid)

	if IsPipedStdin() {
		sendStdinToSandbox(ctx, sb, pid)
	}
}

// IsPipedStdin reports whether stdin is a pipe/redirect rather than an
// interactive terminal.
func IsPipedStdin() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice == 0
}

func sendStdinToSandbox(ctx context.Context, sb *sdk.Sandbox, pid uint32) {
	buf := make([]byte, stdinChunkSize)
	for {
		n, err := os.Stdin.Read(buf)
		if n > 0 {
			if sendErr := sb.Commands().SendStdin(ctx, pid, buf[:n]); sendErr != nil {
				return
			}
		}
		if err != nil {
			break
		}
	}
	_ = sb.Commands().CloseStdin(ctx, pid)
}
