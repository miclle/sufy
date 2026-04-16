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
	"time"
)

// PollOption configures polling behavior for wait-style APIs.
type PollOption func(*pollOpts)

type pollOpts struct {
	interval    time.Duration
	maxInterval time.Duration
	backoff     float64 // multiplier, 1.0 means no back-off
	onPoll      func(attempt int)
}

func defaultPollOpts(defaultInterval time.Duration) *pollOpts {
	return &pollOpts{
		interval:    defaultInterval,
		maxInterval: 0,
		backoff:     1.0,
	}
}

// WithPollInterval sets the initial polling interval.
func WithPollInterval(d time.Duration) PollOption {
	return func(o *pollOpts) { o.interval = d }
}

// WithBackoff enables exponential back-off. multiplier is applied to the
// interval after each poll; maxInterval caps the growth (0 means unbounded).
func WithBackoff(multiplier float64, maxInterval time.Duration) PollOption {
	return func(o *pollOpts) {
		o.backoff = multiplier
		o.maxInterval = maxInterval
	}
}

// WithOnPoll registers a callback that is invoked before every poll attempt.
// The attempt counter starts at 1.
func WithOnPoll(fn func(attempt int)) PollOption {
	return func(o *pollOpts) { o.onPoll = fn }
}

// pollLoop is the shared polling loop used by WaitForReady and WaitForBuild.
// pollFn is called on every iteration and returns (done, result, error).
func pollLoop[T any](ctx context.Context, opts *pollOpts, pollFn func() (bool, T, error)) (T, error) {
	if opts.interval <= 0 {
		opts.interval = time.Second
	}

	interval := opts.interval
	var timer *time.Timer
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()

	attempt := 0
	for {
		attempt++
		if opts.onPoll != nil {
			opts.onPoll(attempt)
		}

		done, result, err := pollFn()
		if err != nil {
			return result, err
		}
		if done {
			return result, nil
		}

		if opts.backoff > 1.0 {
			interval = time.Duration(float64(interval) * opts.backoff)
			if opts.maxInterval > 0 && interval > opts.maxInterval {
				interval = opts.maxInterval
			}
		}

		if timer == nil {
			timer = time.NewTimer(interval)
		} else {
			timer.Reset(interval)
		}
		select {
		case <-ctx.Done():
			var zero T
			return zero, ctx.Err()
		case <-timer.C:
		}
	}
}
