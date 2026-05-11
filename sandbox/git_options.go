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

import "time"

// GitOptions carries common knobs shared by individual git operations. It is
// designed to be embedded into per-operation option structs. The SDK always
// injects GIT_TERMINAL_PROMPT=0 to disable interactive prompts.
type GitOptions struct {
	// Envs supplies extra environment variables.
	Envs map[string]string
	// User selects the OS user executing the command. Defaults to DefaultUser.
	User string
	// Cwd selects the working directory.
	Cwd string
	// Timeout caps the command runtime.
	Timeout time.Duration
}

// CloneOptions configures a git clone.
type CloneOptions struct {
	GitOptions
	// Path is the clone destination path.
	Path string
	// Branch is the branch to check out. When set, --single-branch is appended.
	Branch string
	// Depth requests a shallow clone of the given depth.
	Depth int
	// Username is the HTTPS authentication username.
	Username string
	// Password is the HTTPS authentication password or token.
	Password string
	// DangerouslyStoreCredentials, when true, persists credentials in
	// .git/config. By default credentials are stripped via remote set-url
	// after the clone completes.
	DangerouslyStoreCredentials bool
}

// InitOptions configures a git init.
type InitOptions struct {
	GitOptions
	// Bare requests a bare repository.
	Bare bool
	// InitialBranch sets the initial branch name (for example "main").
	InitialBranch string
}

// RemoteAddOptions configures a git remote add.
type RemoteAddOptions struct {
	GitOptions
	// Fetch runs `git fetch` immediately after adding the remote.
	Fetch bool
	// Overwrite replaces an existing remote's URL when set.
	Overwrite bool
}

// CommitOptions configures a git commit.
type CommitOptions struct {
	GitOptions
	// AuthorName overrides the commit author name.
	AuthorName string
	// AuthorEmail overrides the commit author email.
	AuthorEmail string
	// AllowEmpty permits empty commits.
	AllowEmpty bool
}

// AddOptions configures a git add.
type AddOptions struct {
	GitOptions
	// Files lists the files to stage. When empty, All determines whether to
	// use `-A` or `.`.
	Files []string
	// All controls Files-empty behavior. When nil the SDK defaults to true
	// (aligned with E2B); explicit false uses `.` so only the current
	// directory is staged.
	All *bool
}

// DeleteBranchOptions configures a `git branch -d/-D`.
type DeleteBranchOptions struct {
	GitOptions
	// Force passes -D for forced deletion.
	Force bool
}

// GitResetMode selects the mode for `git reset`.
type GitResetMode string

const (
	// GitResetModeSoft corresponds to `git reset --soft`.
	GitResetModeSoft GitResetMode = "soft"
	// GitResetModeMixed corresponds to `git reset --mixed`.
	GitResetModeMixed GitResetMode = "mixed"
	// GitResetModeHard corresponds to `git reset --hard`.
	GitResetModeHard GitResetMode = "hard"
	// GitResetModeMerge corresponds to `git reset --merge`.
	GitResetModeMerge GitResetMode = "merge"
	// GitResetModeKeep corresponds to `git reset --keep`.
	GitResetModeKeep GitResetMode = "keep"
)

// ResetOptions configures a git reset.
type ResetOptions struct {
	GitOptions
	// Mode selects the reset mode. When empty no `--<mode>` flag is passed.
	Mode GitResetMode
	// Target is the commit, branch or ref to reset to. Defaults to HEAD.
	Target string
	// Paths lists the paths to reset.
	Paths []string
}

// RestoreOptions configures a git restore. When neither Staged nor Worktree is
// explicitly set, only the worktree is restored (Worktree=true).
type RestoreOptions struct {
	GitOptions
	// Paths lists the paths to restore. At least one is required.
	Paths []string
	// Staged, when non-nil, explicitly controls whether to restore the index.
	Staged *bool
	// Worktree, when non-nil, explicitly controls whether to restore the
	// worktree.
	Worktree *bool
	// Source restores from the given commit or ref.
	Source string
}

// PushOptions configures a git push.
type PushOptions struct {
	GitOptions
	// Remote names the remote (for example "origin").
	Remote string
	// Branch names the branch to push.
	Branch string
	// SetUpstream controls whether to append --set-upstream. When nil the SDK
	// defaults to true (aligned with E2B); set explicitly to false to disable.
	SetUpstream *bool
	// Username is the HTTPS authentication username.
	Username string
	// Password is the HTTPS authentication password or token.
	Password string
}

// PullOptions configures a git pull.
type PullOptions struct {
	GitOptions
	// Remote names the remote.
	Remote string
	// Branch names the branch to pull.
	Branch string
	// Username is the HTTPS authentication username.
	Username string
	// Password is the HTTPS authentication password or token.
	Password string
}

// ConfigOptions configures a git config command. When Scope is "local", Path
// is required.
type ConfigOptions struct {
	GitOptions
	// Scope selects the git config scope. Defaults to GitConfigScopeGlobal.
	Scope GitConfigScope
	// Path is the repository path, required when Scope is GitConfigScopeLocal.
	Path string
}

// AuthenticateOptions configures DangerouslyAuthenticate.
type AuthenticateOptions struct {
	GitOptions
	// Username is the HTTPS authentication username (required).
	Username string
	// Password is the HTTPS authentication password or token (required).
	Password string
	// Host is the host to authenticate against. Defaults to "github.com".
	Host string
	// Protocol is the protocol to authenticate against. Defaults to "https".
	// Only "https" is permitted; other values return InvalidArgumentError.
	Protocol string
}
