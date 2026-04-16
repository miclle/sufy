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

	"connectrpc.com/connect"

	"github.com/sufy-dev/sufy/sandbox/internal/envdapi/process"
	"github.com/sufy-dev/sufy/sandbox/internal/envdapi/process/processconnect"
)

// PtySize describes the dimensions of a pseudo-terminal.
type PtySize struct {
	Cols uint32
	Rows uint32
}

// Pty exposes pseudo-terminal operations for a sandbox.
type Pty struct {
	sandbox *Sandbox
	rpc     processconnect.ProcessClient
}

// newPty constructs a Pty sub-module.
func newPty(s *Sandbox, rpc processconnect.ProcessClient) *Pty {
	return &Pty{sandbox: s, rpc: rpc}
}

// Create starts a PTY session. PTY output is delivered through the
// WithOnPtyData callback; when unset, WithOnStdout is used as a fallback.
func (p *Pty) Create(ctx context.Context, size PtySize, opts ...CommandOption) (*CommandHandle, error) {
	o := applyCommandOpts(opts)

	ptyCtx, ptyCancel := context.WithCancel(ctx)

	// Merge default PTY env vars with user-supplied ones.
	envs := map[string]string{
		"TERM":   "xterm",
		"LANG":   "C.UTF-8",
		"LC_ALL": "C.UTF-8",
	}
	for k, v := range o.envs {
		envs[k] = v
	}

	startReq := &process.StartRequest{
		Process: &process.ProcessConfig{
			Cmd:  "/bin/bash",
			Args: []string{"-i", "-l"},
			Envs: envs,
		},
		Pty: &process.PTY{
			Size: &process.PTY_Size{
				Cols: size.Cols,
				Rows: size.Rows,
			},
		},
	}
	if o.cwd != "" {
		startReq.Process.Cwd = &o.cwd
	}
	if o.tag != "" {
		startReq.Tag = &o.tag
	}

	req := connect.NewRequest(startReq)
	p.sandbox.setEnvdAuth(req, o.user)

	stream, err := p.rpc.Start(ptyCtx, req)
	if err != nil {
		ptyCancel()
		return nil, fmt.Errorf("create pty: %w", err)
	}

	commands := &Commands{sandbox: p.sandbox, rpc: p.rpc}

	// Prefer onPtyData; fall back to onStdout for compatibility.
	ptyDataFn := o.onPtyData
	if ptyDataFn == nil {
		ptyDataFn = o.onStdout
	}

	handle := &CommandHandle{
		commands:  commands,
		cancel:    ptyCancel,
		done:      make(chan struct{}),
		pidCh:     make(chan struct{}),
		onPtyData: ptyDataFn,
	}

	go processEventStream(stream, handle)

	return handle, nil
}

// Connect attaches to an existing PTY session by PID.
func (p *Pty) Connect(ctx context.Context, pid uint32) (*CommandHandle, error) {
	commands := &Commands{sandbox: p.sandbox, rpc: p.rpc}
	return commands.Connect(ctx, pid)
}

// SendInput writes bytes to the PTY.
func (p *Pty) SendInput(ctx context.Context, pid uint32, data []byte) error {
	req := connect.NewRequest(&process.SendInputRequest{
		Process: pidSelector(pid),
		Input: &process.ProcessInput{
			Input: &process.ProcessInput_Pty{Pty: data},
		},
	})
	p.sandbox.setEnvdAuth(req, DefaultUser)

	_, err := p.rpc.SendInput(ctx, req)
	if err != nil {
		return fmt.Errorf("send pty input: %w", err)
	}
	return nil
}

// Resize updates the PTY's dimensions.
func (p *Pty) Resize(ctx context.Context, pid uint32, size PtySize) error {
	req := connect.NewRequest(&process.UpdateRequest{
		Process: pidSelector(pid),
		Pty: &process.PTY{
			Size: &process.PTY_Size{
				Cols: size.Cols,
				Rows: size.Rows,
			},
		},
	})
	p.sandbox.setEnvdAuth(req, DefaultUser)

	_, err := p.rpc.Update(ctx, req)
	if err != nil {
		return fmt.Errorf("resize pty: %w", err)
	}
	return nil
}

// Kill terminates the PTY session.
func (p *Pty) Kill(ctx context.Context, pid uint32) error {
	commands := &Commands{sandbox: p.sandbox, rpc: p.rpc}
	return commands.Kill(ctx, pid)
}
