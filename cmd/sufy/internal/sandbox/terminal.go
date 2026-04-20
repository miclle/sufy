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
	"os"
	"sync"
	"time"

	"golang.org/x/term"

	sdk "github.com/sufy-dev/sufy/sandbox"
)

// batchedWriter buffers stdin bytes and flushes them on an interval so the
// PTY receives fewer, larger SendInput RPCs. The 10ms interval matches the
// e2b CLI's BatchedQueue default.
type batchedWriter struct {
	mu     sync.Mutex
	buf    []byte
	sendFn func(ctx context.Context, data []byte) error
	ctx    context.Context
	done   chan struct{}
}

type terminalSize struct {
	width, height int
}

func newBatchedWriter(ctx context.Context, interval time.Duration, sendFn func(ctx context.Context, data []byte) error) *batchedWriter {
	bw := &batchedWriter{sendFn: sendFn, ctx: ctx, done: make(chan struct{})}
	go bw.flushLoop(interval)
	return bw
}

func (bw *batchedWriter) Write(data []byte) {
	bw.mu.Lock()
	bw.buf = append(bw.buf, data...)
	bw.mu.Unlock()
}

func (bw *batchedWriter) flush() {
	bw.mu.Lock()
	if len(bw.buf) == 0 {
		bw.mu.Unlock()
		return
	}
	data := bw.buf
	bw.buf = nil
	bw.mu.Unlock()

	_ = bw.sendFn(bw.ctx, data)
}

func (bw *batchedWriter) flushLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			bw.flush()
		case <-bw.done:
			bw.flush()
			return
		}
	}
}

func (bw *batchedWriter) stop() { close(bw.done) }

func detectResize(previous terminalSize, width, height int, err error) (terminalSize, bool) {
	if err != nil {
		return previous, false
	}
	current := terminalSize{width: width, height: height}
	if current == previous {
		return previous, false
	}
	return current, true
}

// RunTerminalSession bridges the local TTY with a PTY running in the sandbox.
// It sets raw mode, forwards resize events, refreshes the sandbox timeout, and
// proxies stdin through a batched writer.
func RunTerminalSession(ctx context.Context, sb *sdk.Sandbox) {
	width, height, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		width, height = 80, 24
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		PrintError("failed to set raw mode: %v", err)
		return
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	ptyCtx, ptyCancel := context.WithCancel(ctx)
	defer ptyCancel()

	handle, err := sb.Pty().Create(ptyCtx, sdk.PtySize{
		Cols: uint32(width),
		Rows: uint32(height),
	}, sdk.WithOnPtyData(func(data []byte) {
		os.Stdout.Write(data)
	}))
	if err != nil {
		PrintError("create PTY failed: %v", err)
		return
	}

	pid, err := handle.WaitPID(ptyCtx)
	if err != nil {
		PrintError("wait for PTY PID failed: %v", err)
		return
	}

	resizeEvents := make(chan struct{}, 1)
	notifyTerminalResize(ptyCtx, resizeEvents)

	startResizeMonitor(ptyCtx, resizeEvents, terminalSize{width: width, height: height},
		func() (int, int, error) { return term.GetSize(int(os.Stdin.Fd())) },
		func(w, h int) {
			_ = sb.Pty().Resize(ptyCtx, pid, sdk.PtySize{Cols: uint32(w), Rows: uint32(h)})
		})

	// Keep-alive loop: matches e2b CLI (5s interval, 30s extension).
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		_ = sb.SetTimeout(ptyCtx, 30*time.Second)
		for {
			select {
			case <-ticker.C:
				_ = sb.SetTimeout(ptyCtx, 30*time.Second)
			case <-ptyCtx.Done():
				return
			}
		}
	}()

	writer := newBatchedWriter(ptyCtx, 10*time.Millisecond, func(ctx context.Context, data []byte) error {
		return sb.Pty().SendInput(ctx, pid, data)
	})
	defer writer.stop()

	go func() {
		buf := make([]byte, 1024)
		for {
			n, rErr := os.Stdin.Read(buf)
			if rErr != nil {
				ptyCancel()
				return
			}
			if n > 0 {
				writer.Write(buf[:n])
			}
		}
	}()

	handle.Wait()
}

func startResizeMonitor(
	ctx context.Context,
	resizeEvents <-chan struct{},
	initial terminalSize,
	getSize func() (int, int, error),
	resize func(width, height int),
) {
	go func() {
		size := initial
		for {
			select {
			case <-resizeEvents:
				w, h, err := getSize()
				next, changed := detectResize(size, w, h, err)
				if changed {
					size = next
					resize(w, h)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}
