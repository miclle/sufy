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
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// GitAuthError signals that a git operation aborted because of failed
// authentication.
type GitAuthError struct {
	// Msg is the error message.
	Msg string
	// Err is the underlying error (possibly joined via errors.Join) for
	// errors.Is/As unwrapping.
	Err error
}

// Error implements the error interface.
func (e *GitAuthError) Error() string { return e.Msg }

// Unwrap returns the underlying error so callers can use errors.Is/As to dig
// further.
func (e *GitAuthError) Unwrap() error { return e.Err }

// GitUpstreamError signals that a git operation aborted because the upstream
// tracking branch is missing.
type GitUpstreamError struct {
	// Msg is the error message.
	Msg string
	// Err is the underlying error (possibly joined via errors.Join) for
	// errors.Is/As unwrapping.
	Err error
}

// Error implements the error interface.
func (e *GitUpstreamError) Error() string { return e.Msg }

// Unwrap returns the underlying error so callers can use errors.Is/As to dig
// further.
func (e *GitUpstreamError) Unwrap() error { return e.Err }

// InvalidArgumentError signals that an argument is invalid.
type InvalidArgumentError struct {
	// Msg is the error message.
	Msg string
}

// Error implements the error interface.
func (e *InvalidArgumentError) Error() string { return e.Msg }

// withCredentials embeds HTTPS credentials into a git repository URL. Only the
// https scheme is allowed; when both username and password are empty the URL
// is returned unchanged.
//
// http is deliberately rejected — putting a token or password into a plaintext
// URL exposes credentials over non-TLS links, which is incompatible with the
// public contract of "HTTPS + username/password only".
func withCredentials(rawURL, username, password string) (string, error) {
	if username == "" && password == "" {
		return rawURL, nil
	}
	if username == "" || password == "" {
		return "", &InvalidArgumentError{Msg: "Both username and password are required when using Git credentials."}
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse git url: %w", err)
	}
	if u.Scheme != "https" {
		return "", &InvalidArgumentError{Msg: "Only https Git URLs support username/password credentials."}
	}

	u.User = url.UserPassword(username, password)
	return u.String(), nil
}

// stripCredentials removes embedded credentials from a git URL. The input is
// returned unchanged when parsing fails or when the URL is not http(s).
func stripCredentials(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return rawURL
	}
	if u.User == nil {
		return rawURL
	}
	u.User = nil
	return u.String()
}

// requireHTTPSIfHasCredentials enforces that a URL with embedded credentials
// uses https. Parse failures and credential-free URLs return nil so the
// downstream command can produce its own diagnostics.
func requireHTTPSIfHasCredentials(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil || u.User == nil {
		return nil
	}
	if u.Scheme != "https" {
		return &InvalidArgumentError{Msg: "Only https Git URLs support inline username/password credentials."}
	}
	return nil
}

// authFailureSnippets lists common git auth-failure markers (lowercase match).
//
// Only credential-specific markers are listed. Generic "permission denied"
// messages — caused by missing write permissions on the worktree, broken
// .git/config permissions, or lock-file ownership issues — are excluded to
// avoid steering callers toward the wrong diagnosis.
var authFailureSnippets = []string{
	"authentication failed",
	"terminal prompts disabled",
	"could not read username",
	"could not read password",
	"invalid username or password",
	"bad credentials",
	"requested url returned error: 401",
	"requested url returned error: 403",
	"http basic: access denied",
}

// upstreamFailureSnippets lists common missing-upstream markers (lowercase match).
var upstreamFailureSnippets = []string{
	"has no upstream branch",
	"no upstream branch",
	"no upstream configured",
	"no tracking information for the current branch",
	"no tracking information",
	"set the remote as upstream",
	"set the upstream branch",
	"please specify which branch you want to merge with",
}

// gitCommandError is the internal error returned by git commands. It carries
// stdout/stderr so later matching can classify the failure.
type gitCommandError struct {
	// Cmd is the failed git subcommand name (e.g. "clone", "push").
	Cmd string
	// Result is the underlying command result with ExitCode/Stdout/Stderr.
	Result *CommandResult
}

// Error implements the error interface.
func (e *gitCommandError) Error() string {
	if e.Result == nil {
		return fmt.Sprintf("git %s failed", e.Cmd)
	}
	if stderr := strings.TrimSpace(e.Result.Stderr); stderr != "" {
		return fmt.Sprintf("git %s failed (exit %d): %s", e.Cmd, e.Result.ExitCode, stderr)
	}
	return fmt.Sprintf("git %s failed (exit %d)", e.Cmd, e.Result.ExitCode)
}

// matchSnippets searches the stdout+stderr of a git command error for the
// given snippets (lowercase match).
func matchSnippets(err error, snippets []string) bool {
	var ge *gitCommandError
	if !errors.As(err, &ge) || ge.Result == nil {
		return false
	}
	message := strings.ToLower(ge.Result.Stderr + "\n" + ge.Result.Stdout)
	for _, s := range snippets {
		if strings.Contains(message, s) {
			return true
		}
	}
	return false
}

// isAuthFailure reports whether the error originated from a git auth failure.
func isAuthFailure(err error) bool { return matchSnippets(err, authFailureSnippets) }

// isMissingUpstream reports whether the error originated from a missing
// upstream tracking branch.
func isMissingUpstream(err error) bool { return matchSnippets(err, upstreamFailureSnippets) }

// buildAuthErrorMessage builds an auth-error message tailored to the action.
func buildAuthErrorMessage(action string, missingPassword bool) string {
	if missingPassword {
		return fmt.Sprintf("Git %s requires a password/token for private repositories.", action)
	}
	return fmt.Sprintf("Git %s requires credentials for private repositories.", action)
}

// buildUpstreamErrorMessage builds a missing-upstream message tailored to the
// action.
func buildUpstreamErrorMessage(action string) string {
	if action == "push" {
		return "Git push failed because no upstream branch is configured. " +
			"Set upstream once with SetUpstream=true (and optional Remote/Branch), " +
			"or pass Remote and Branch explicitly."
	}
	return "Git pull failed because no upstream branch is configured. " +
		"Pass Remote and Branch explicitly, or set upstream once (push with " +
		"SetUpstream=true or run: git branch --set-upstream-to=origin/<branch> <branch>)."
}
