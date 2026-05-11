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
	"sync"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"

	"github.com/sufy-dev/sufy/sandbox/internal/envdapi/process"
	"github.com/sufy-dev/sufy/sandbox/internal/envdapi/process/processconnect"
)

// CommandResult holds the outcome of a command execution.
type CommandResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Error    string
}

// CommandHandle is a handle to a running background command.
type CommandHandle struct {
	// pid is set by the event-stream goroutine and read by PID()/Kill() concurrently.
	pid atomic.Uint32

	commands *Commands
	cancel   context.CancelFunc
	done     chan struct{}
	pidCh    chan struct{}
	pidOnce  sync.Once
	result   *CommandResult

	mu        sync.Mutex
	onStdout  func(data []byte)
	onStderr  func(data []byte)
	onPtyData func(data []byte)
}

// markPIDReady stores pid and closes pidCh exactly once. Both the Start event
// handler and Connect's caller-supplied PID may race to set it.
func (h *CommandHandle) markPIDReady(pid uint32) {
	h.pidOnce.Do(func() {
		h.pid.Store(pid)
		close(h.pidCh)
	})
}

// PID returns the process identifier.
func (h *CommandHandle) PID() uint32 {
	return h.pid.Load()
}

// Wait blocks until the command completes and returns the result.
func (h *CommandHandle) Wait() (*CommandResult, error) {
	<-h.done
	if h.result == nil {
		return nil, fmt.Errorf("command terminated without result")
	}
	return h.result, nil
}

// Kill terminates the command.
func (h *CommandHandle) Kill(ctx context.Context) error {
	return h.commands.Kill(ctx, h.pid.Load())
}

// WaitPID blocks until the PID has been assigned or the context is cancelled.
func (h *CommandHandle) WaitPID(ctx context.Context) (uint32, error) {
	select {
	case <-h.pidCh:
		return h.pid.Load(), nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

// ProcessInfo describes a running sandbox process.
type ProcessInfo struct {
	PID  uint32
	Tag  *string
	Cmd  string
	Args []string
	Envs map[string]string
	Cwd  *string
}

// CommandOption configures a command execution.
type CommandOption func(*commandOpts)

type commandOpts struct {
	envs      map[string]string
	cwd       string
	user      string
	tag       string
	onStdout  func(data []byte)
	onStderr  func(data []byte)
	onPtyData func(data []byte)
	timeout   time.Duration
	stdin     bool
}

// WithEnvs sets environment variables for the command.
func WithEnvs(envs map[string]string) CommandOption {
	return func(o *commandOpts) { o.envs = envs }
}

// WithCwd sets the command's working directory.
func WithCwd(cwd string) CommandOption {
	return func(o *commandOpts) { o.cwd = cwd }
}

// WithCommandUser sets the OS user used to execute the command.
func WithCommandUser(user string) CommandOption {
	return func(o *commandOpts) { o.user = user }
}

// WithTag attaches a tag to the process so it can be re-identified later.
func WithTag(tag string) CommandOption {
	return func(o *commandOpts) { o.tag = tag }
}

// WithOnStdout sets a streaming callback for stdout. Applies to standard
// commands only; PTY sessions should use WithOnPtyData.
func WithOnStdout(fn func(data []byte)) CommandOption {
	return func(o *commandOpts) { o.onStdout = fn }
}

// WithOnStderr sets a streaming callback for stderr.
func WithOnStderr(fn func(data []byte)) CommandOption {
	return func(o *commandOpts) { o.onStderr = fn }
}

// WithOnPtyData sets the streaming callback for PTY output. When unset,
// Pty.Create falls back to the WithOnStdout callback for compatibility.
func WithOnPtyData(fn func(data []byte)) CommandOption {
	return func(o *commandOpts) { o.onPtyData = fn }
}

// WithTimeout sets an upper bound on command execution.
func WithTimeout(timeout time.Duration) CommandOption {
	return func(o *commandOpts) { o.timeout = timeout }
}

// WithStdin enables stdin for the spawned process. After Start, data can be
// written through Commands.SendStdin and EOF delivered via Commands.CloseStdin.
func WithStdin() CommandOption {
	return func(o *commandOpts) { o.stdin = true }
}

func applyCommandOpts(opts []CommandOption) *commandOpts {
	o := &commandOpts{user: DefaultUser}
	for _, fn := range opts {
		fn(o)
	}
	return o
}

// Commands exposes command execution for a sandbox.
type Commands struct {
	sandbox *Sandbox
	rpc     processconnect.ProcessClient
}

// newCommands constructs a Commands sub-module.
func newCommands(s *Sandbox, rpc processconnect.ProcessClient) *Commands {
	return &Commands{sandbox: s, rpc: rpc}
}

// pidSelector returns a ProcessSelector that targets a process by PID.
func pidSelector(pid uint32) *process.ProcessSelector {
	return &process.ProcessSelector{Selector: &process.ProcessSelector_Pid{Pid: pid}}
}

// Run executes a command inside the sandbox and blocks until it completes.
//
// NOTE: stdout and stderr are accumulated in memory. For long-running commands
// or commands with large output, prefer Start + WithOnStdout/WithOnStderr to
// stream output incrementally.
func (c *Commands) Run(ctx context.Context, cmd string, opts ...CommandOption) (*CommandResult, error) {
	handle, err := c.Start(ctx, cmd, opts...)
	if err != nil {
		return nil, err
	}
	return handle.Wait()
}

// Start launches a background command. The command runs as /bin/bash -l -c
// <cmd>, which enables shell syntax (pipes, redirections) and loads the login
// shell environment.
func (c *Commands) Start(ctx context.Context, cmd string, opts ...CommandOption) (*CommandHandle, error) {
	o := applyCommandOpts(opts)

	var cmdCtx context.Context
	var cmdCancel context.CancelFunc
	if o.timeout > 0 {
		cmdCtx, cmdCancel = context.WithTimeout(ctx, o.timeout)
	} else {
		cmdCtx, cmdCancel = context.WithCancel(ctx)
	}

	startReq := &process.StartRequest{
		Process: &process.ProcessConfig{
			Cmd:  "/bin/bash",
			Args: []string{"-l", "-c", cmd},
			Envs: o.envs,
		},
	}
	if o.cwd != "" {
		startReq.Process.Cwd = &o.cwd
	}
	if o.tag != "" {
		startReq.Tag = &o.tag
	}
	startReq.Stdin = &o.stdin

	req := connect.NewRequest(startReq)
	c.sandbox.setEnvdAuth(req, o.user)

	stream, err := c.rpc.Start(cmdCtx, req)
	if err != nil {
		cmdCancel()
		return nil, fmt.Errorf("start command: %w", err)
	}

	handle := &CommandHandle{
		commands: c,
		cancel:   cmdCancel,
		done:     make(chan struct{}),
		pidCh:    make(chan struct{}),
		onStdout: o.onStdout,
		onStderr: o.onStderr,
	}

	go processEventStream(stream, handle)

	return handle, nil
}

// eventMessage is the common interface satisfied by StartResponse and
// ConnectResponse.
type eventMessage interface {
	GetEvent() *process.ProcessEvent
}

// streamReceiver abstracts the server-streaming read surface of ConnectRPC.
type streamReceiver[T eventMessage] interface {
	Receive() bool
	Msg() T
	Err() error
}

// processEventStream drains a process event stream, populating the handle's
// result and forwarding output via the registered callbacks.
func processEventStream[T eventMessage](stream streamReceiver[T], handle *CommandHandle) {
	defer close(handle.done)

	var stdout, stderr []byte
	for stream.Receive() {
		event := stream.Msg().GetEvent()
		if event == nil {
			continue
		}
		switch ev := event.Event.(type) {
		case *process.ProcessEvent_Start:
			handle.markPIDReady(ev.Start.Pid)
		case *process.ProcessEvent_Data:
			if data := ev.Data.GetStdout(); len(data) > 0 {
				stdout = append(stdout, data...)
				handle.mu.Lock()
				fn := handle.onStdout
				handle.mu.Unlock()
				if fn != nil {
					fn(data)
				}
			}
			if data := ev.Data.GetStderr(); len(data) > 0 {
				stderr = append(stderr, data...)
				handle.mu.Lock()
				fn := handle.onStderr
				handle.mu.Unlock()
				if fn != nil {
					fn(data)
				}
			}
			if data := ev.Data.GetPty(); len(data) > 0 {
				handle.mu.Lock()
				fn := handle.onPtyData
				handle.mu.Unlock()
				if fn != nil {
					fn(data)
				}
			}
		case *process.ProcessEvent_End:
			handle.result = &CommandResult{
				ExitCode: int(ev.End.ExitCode),
				Stdout:   string(stdout),
				Stderr:   string(stderr),
			}
			if ev.End.Error != nil {
				handle.result.Error = *ev.End.Error
			}
		}
	}

	// If the stream ended without an End event, synthesize an error result.
	if handle.result == nil {
		errMsg := ""
		if err := stream.Err(); err != nil {
			errMsg = err.Error()
		}
		handle.result = &CommandResult{
			ExitCode: -1,
			Stdout:   string(stdout),
			Stderr:   string(stderr),
			Error:    errMsg,
		}
	}
}

// Connect attaches to a running process by PID.
func (c *Commands) Connect(ctx context.Context, pid uint32) (*CommandHandle, error) {
	connectCtx, connectCancel := context.WithCancel(ctx)

	req := connect.NewRequest(&process.ConnectRequest{
		Process: pidSelector(pid),
	})
	c.sandbox.setEnvdAuth(req, DefaultUser)

	stream, err := c.rpc.Connect(connectCtx, req)
	if err != nil {
		connectCancel()
		return nil, fmt.Errorf("connect to process: %w", err)
	}

	handle := &CommandHandle{
		commands: c,
		cancel:   connectCancel,
		done:     make(chan struct{}),
		pidCh:    make(chan struct{}),
	}
	handle.markPIDReady(pid)

	go processEventStream(stream, handle)

	return handle, nil
}

// List returns all running processes.
func (c *Commands) List(ctx context.Context) ([]ProcessInfo, error) {
	req := connect.NewRequest(&process.ListRequest{})
	c.sandbox.setEnvdAuth(req, DefaultUser)

	resp, err := c.rpc.List(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("list processes: %w", err)
	}

	var infos []ProcessInfo
	for _, p := range resp.Msg.Processes {
		info := ProcessInfo{
			PID: p.Pid,
			Tag: p.Tag,
		}
		if p.Config != nil {
			info.Cmd = p.Config.Cmd
			info.Args = p.Config.Args
			info.Envs = p.Config.Envs
			info.Cwd = p.Config.Cwd
		}
		infos = append(infos, info)
	}
	return infos, nil
}

// SendStdin writes data to a process's stdin stream.
func (c *Commands) SendStdin(ctx context.Context, pid uint32, data []byte) error {
	req := connect.NewRequest(&process.SendInputRequest{
		Process: pidSelector(pid),
		Input: &process.ProcessInput{
			Input: &process.ProcessInput_Stdin{Stdin: data},
		},
	})
	c.sandbox.setEnvdAuth(req, DefaultUser)

	_, err := c.rpc.SendInput(ctx, req)
	if err != nil {
		return fmt.Errorf("send stdin: %w", err)
	}
	return nil
}

// CloseStdin closes stdin for a process, delivering EOF. Intended for non-PTY
// processes; PTY processes should emit Ctrl+D (0x04) via SendStdin instead.
// If the server does not implement this RPC the Unimplemented error can be
// safely ignored.
func (c *Commands) CloseStdin(ctx context.Context, pid uint32) error {
	req := connect.NewRequest(&process.CloseStdinRequest{
		Process: pidSelector(pid),
	})
	c.sandbox.setEnvdAuth(req, DefaultUser)

	_, err := c.rpc.CloseStdin(ctx, req)
	if err != nil {
		return fmt.Errorf("close stdin: %w", err)
	}
	return nil
}

// Kill sends SIGKILL to the target process.
func (c *Commands) Kill(ctx context.Context, pid uint32) error {
	req := connect.NewRequest(&process.SendSignalRequest{
		Process: pidSelector(pid),
		Signal:  process.Signal_SIGNAL_SIGKILL,
	})
	c.sandbox.setEnvdAuth(req, DefaultUser)

	_, err := c.rpc.SendSignal(ctx, req)
	if err != nil {
		return fmt.Errorf("kill process: %w", err)
	}
	return nil
}
