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

//go:build windows

package sandbox

import (
	"context"
	"time"
)

// windowsResizePollInterval is the polling interval used on Windows in the
// absence of a SIGWINCH equivalent. Matches the e2b CLI default.
const windowsResizePollInterval = 200 * time.Millisecond

// notifyTerminalResize periodically signals a potential resize on Windows so
// the caller can re-query the terminal size.
func notifyTerminalResize(ctx context.Context, resizeEvents chan<- struct{}) {
	go func() {
		ticker := time.NewTicker(windowsResizePollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				select {
				case resizeEvents <- struct{}{}:
				default:
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}
