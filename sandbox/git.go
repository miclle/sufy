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

// Package sandbox's git.go provides high-level git operations inside the
// sandbox.
//
// Git commands run via Commands.Run against the git binary preinstalled in the
// sandbox. Only HTTPS + username/password (token) authentication is supported;
// SSH keys are not.

package sandbox

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strings"
	"time"
)

// defaultGitEnv lists the environment variables injected into every git
// command. GIT_TERMINAL_PROMPT=0 disables interactive prompts.
var defaultGitEnv = map[string]string{
	"GIT_TERMINAL_PROMPT": "0",
}

// credentialCleanupTimeout caps credential-cleanup steps (strip / restore
// origin URL). The cleanup deliberately runs on a fresh context so it still
// executes after the main ctx is cancelled, but a bounded timeout prevents an
// unresponsive sandbox or network glitch from blocking the caller's goroutine.
const credentialCleanupTimeout = 10 * time.Second

// Git is the sandbox-side git interface.
//
// Only HTTPS + username/password (token) authentication is supported; SSH
// keys are not. GIT_TERMINAL_PROMPT=0 is always injected so git never blocks
// on interactive input.
type Git struct {
	commands *Commands
}

// newGit constructs a Git sub-module.
func newGit(c *Commands) *Git {
	return &Git{commands: c}
}

// validateCredentialPair ensures username and password are either both
// provided or both omitted. verb is used to format the error message
// ("clone" / "push" / "pull").
func validateCredentialPair(verb, username, password string) error {
	if (username == "") == (password == "") {
		return nil
	}
	return &InvalidArgumentError{Msg: fmt.Sprintf("Username and password must be provided together for git %s, or not at all.", verb)}
}

// Clone clones a remote repository into the sandbox.
//
// When opts.Username/Password is provided and DangerouslyStoreCredentials is
// false, the SDK strips the credentials from origin's URL via
// `git remote set-url` after the clone completes.
func (g *Git) Clone(ctx context.Context, repoURL string, opts *CloneOptions) (*CommandResult, error) {
	if opts == nil {
		opts = &CloneOptions{}
	}
	if err := validateCredentialPair("clone", opts.Username, opts.Password); err != nil {
		return nil, err
	}

	plan, err := buildClonePlan(repoURL, opts.Path, opts.Branch, opts.Depth, opts.Username, opts.Password, opts.DangerouslyStoreCredentials)
	if err != nil {
		return nil, err
	}

	result, err := g.runGit(ctx, "clone", plan.args, "", &opts.GitOptions)
	if err != nil {
		if isAuthFailure(err) {
			return nil, &GitAuthError{Msg: buildAuthErrorMessage("clone", opts.Username != "" && opts.Password == ""), Err: err}
		}
		return nil, err
	}

	if plan.shouldStrip {
		// Strip credentials even if ctx has been cancelled, so the
		// credential-bearing URL never lingers in .git/config. Use an
		// independent timeout-bounded context so a sandbox glitch cannot
		// block forever.
		cleanupCtx, cancel := context.WithTimeout(context.Background(), credentialCleanupTimeout)
		defer cancel()
		_, serr := g.runGit(cleanupCtx, "remote", buildRemoteSetURLArgs("origin", plan.sanitizedURL), plan.repoPath, &opts.GitOptions)
		if serr != nil {
			return result, fmt.Errorf("clone succeeded but failed to strip credentials: %w", serr)
		}
	}
	return result, nil
}

// Init initializes a new git repository at the given path.
func (g *Git) Init(ctx context.Context, path string, opts *InitOptions) (*CommandResult, error) {
	if opts == nil {
		opts = &InitOptions{}
	}
	args := []string{"init"}
	if opts.InitialBranch != "" {
		args = append(args, "--initial-branch", opts.InitialBranch)
	}
	if opts.Bare {
		args = append(args, "--bare")
	}
	args = append(args, path)
	return g.runGit(ctx, "init", args, "", &opts.GitOptions)
}

// Status returns the overall status of the repository.
func (g *Git) Status(ctx context.Context, path string, opts *GitOptions) (*GitStatus, error) {
	result, err := g.runGit(ctx, "status", buildStatusArgs(), path, opts)
	if err != nil {
		return nil, err
	}
	return parseGitStatus(result.Stdout), nil
}

// Branches returns the branch listing of the repository.
//
// On an unborn repo (after `git init` but before the first commit),
// `git branch` produces no output even though HEAD already points at the
// initial branch. In that case we fall back to
// `git symbolic-ref --short HEAD` so CurrentBranch is still populated, in
// line with the contract that CurrentBranch is empty only on detached HEAD.
func (g *Git) Branches(ctx context.Context, path string, opts *GitOptions) (*GitBranches, error) {
	result, err := g.runGit(ctx, "branch", buildBranchesArgs(), path, opts)
	if err != nil {
		return nil, err
	}
	branches := parseGitBranches(result.Stdout)
	if branches.CurrentBranch == "" && len(branches.Branches) == 0 {
		if name, ok := g.unbornCurrentBranch(ctx, path, opts); ok {
			branches.CurrentBranch = name
		}
	}
	return branches, nil
}

// unbornCurrentBranch tries to read the current branch of an unborn repo via
// symbolic-ref. Returns ok=false when symbolic-ref fails — e.g. on detached
// HEAD or when the path is not a git directory.
func (g *Git) unbornCurrentBranch(ctx context.Context, path string, opts *GitOptions) (string, bool) {
	result, err := g.runGit(ctx, "symbolic-ref", []string{"symbolic-ref", "--short", "HEAD"}, path, opts)
	if err != nil {
		return "", false
	}
	name := strings.TrimSpace(result.Stdout)
	if name == "" {
		return "", false
	}
	return name, true
}

// CreateBranch creates a new branch and switches to it.
func (g *Git) CreateBranch(ctx context.Context, path, branch string, opts *GitOptions) (*CommandResult, error) {
	if branch == "" {
		return nil, &InvalidArgumentError{Msg: "Branch name is required."}
	}
	return g.runGit(ctx, "checkout", buildCreateBranchArgs(branch), path, opts)
}

// CheckoutBranch switches to an existing branch.
func (g *Git) CheckoutBranch(ctx context.Context, path, branch string, opts *GitOptions) (*CommandResult, error) {
	if branch == "" {
		return nil, &InvalidArgumentError{Msg: "Branch name is required."}
	}
	return g.runGit(ctx, "checkout", buildCheckoutBranchArgs(branch), path, opts)
}

// DeleteBranch deletes a branch.
func (g *Git) DeleteBranch(ctx context.Context, path, branch string, opts *DeleteBranchOptions) (*CommandResult, error) {
	if branch == "" {
		return nil, &InvalidArgumentError{Msg: "Branch name is required."}
	}
	if opts == nil {
		opts = &DeleteBranchOptions{}
	}
	return g.runGit(ctx, "branch", buildDeleteBranchArgs(branch, opts.Force), path, &opts.GitOptions)
}

// Add stages the specified files. When no files are provided it stages all
// changes by default.
func (g *Git) Add(ctx context.Context, path string, opts *AddOptions) (*CommandResult, error) {
	if opts == nil {
		opts = &AddOptions{}
	}
	all := true
	if opts.All != nil {
		all = *opts.All
	}
	return g.runGit(ctx, "add", buildAddArgs(opts.Files, all), path, &opts.GitOptions)
}

// Commit creates a commit in the repository.
func (g *Git) Commit(ctx context.Context, path, message string, opts *CommitOptions) (*CommandResult, error) {
	if message == "" {
		return nil, &InvalidArgumentError{Msg: "Commit message is required."}
	}
	if opts == nil {
		opts = &CommitOptions{}
	}
	return g.runGit(ctx, "commit", buildCommitArgs(message, opts.AuthorName, opts.AuthorEmail, opts.AllowEmpty), path, &opts.GitOptions)
}

// Reset resets HEAD to the given state.
func (g *Git) Reset(ctx context.Context, path string, opts *ResetOptions) (*CommandResult, error) {
	if opts == nil {
		opts = &ResetOptions{}
	}
	args, err := buildResetArgs(opts.Mode, opts.Target, opts.Paths)
	if err != nil {
		return nil, err
	}
	return g.runGit(ctx, "reset", args, path, &opts.GitOptions)
}

// Restore restores worktree files or unstages paths.
func (g *Git) Restore(ctx context.Context, path string, opts *RestoreOptions) (*CommandResult, error) {
	if opts == nil {
		return nil, &InvalidArgumentError{Msg: "Restore options are required."}
	}
	args, err := buildRestoreArgs(opts.Paths, opts.Staged, opts.Worktree, opts.Source)
	if err != nil {
		return nil, err
	}
	return g.runGit(ctx, "restore", args, path, &opts.GitOptions)
}

// Push pushes commits to a remote.
//
// When opts.Username/Password is provided the SDK temporarily writes the
// credentials into the target remote URL and restores the original URL after
// the command completes.
//
// When Remote is unset, the SDK tries to auto-select the repository's sole
// remote (aligned with the "auto-select single remote" contract). With
// multiple or no remotes it falls back to git's native behavior or returns an
// explicit error when credentials are required.
//
// When SetUpstream=true and Branch is unset, the SDK only resolves the
// current branch (via `git rev-parse`) when a target remote has been
// determined. Without that, `git push --set-upstream <remote>` without a
// branch name would be rejected by git. When target is undetermined (multiple
// or zero remotes), we keep the bare `git push` form and let git emit its own
// error rather than misinterpreting the branch name as a remote.
func (g *Git) Push(ctx context.Context, path string, opts *PushOptions) (*CommandResult, error) {
	if opts == nil {
		opts = &PushOptions{}
	}
	if err := validateCredentialPair("push", opts.Username, opts.Password); err != nil {
		return nil, err
	}
	setUpstream := true
	if opts.SetUpstream != nil {
		setUpstream = *opts.SetUpstream
	}

	buildArgs := func(target string) ([]string, error) {
		branch := opts.Branch
		if target == "" {
			if branch != "" {
				return nil, &InvalidArgumentError{Msg: "Remote is required when Branch is specified and the repository does not have a single remote to auto-select."}
			}
			return buildPushArgs("", "", false), nil
		}
		if setUpstream && branch == "" {
			name, err := g.currentBranch(ctx, path, &opts.GitOptions)
			if err != nil {
				return nil, err
			}
			branch = name
		}
		return buildPushArgs(target, branch, setUpstream), nil
	}
	return g.runWithOptionalCredentials(ctx, "push", path, opts.Remote, opts.Username, opts.Password, &opts.GitOptions, buildArgs)
}

// Pull pulls changes from a remote.
//
// When opts.Username/Password is provided the SDK temporarily writes the
// credentials into the target remote URL and restores the original URL after
// the command completes.
//
// When Remote is unset, the SDK tries to auto-select the repository's sole
// remote. When both Remote and Branch are unset, the SDK first checks whether
// the current branch has an upstream configured and returns GitUpstreamError
// instead of letting an ambiguous error reach the caller.
func (g *Git) Pull(ctx context.Context, path string, opts *PullOptions) (*CommandResult, error) {
	if opts == nil {
		opts = &PullOptions{}
	}
	if err := validateCredentialPair("pull", opts.Username, opts.Password); err != nil {
		return nil, err
	}

	if opts.Remote == "" && opts.Branch == "" {
		ok, err := g.hasUpstream(ctx, path, &opts.GitOptions)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, &GitUpstreamError{Msg: buildUpstreamErrorMessage("pull")}
		}
	}
	return g.runWithOptionalCredentials(ctx, "pull", path, opts.Remote, opts.Username, opts.Password, &opts.GitOptions,
		func(target string) ([]string, error) {
			// When target is empty but Branch was explicit, fail upfront:
			// `git pull <branch>` is parsed as `git pull <repository>`, which
			// would conflict with our public contract.
			if target == "" && opts.Branch != "" {
				return nil, &InvalidArgumentError{Msg: "Remote is required when Branch is specified and the repository does not have a single remote to auto-select."}
			}
			return buildPullArgs(target, opts.Branch), nil
		})
}

// currentBranch returns the name of the branch HEAD currently points at via
// `git rev-parse`. On detached HEAD rev-parse returns "HEAD"; we turn that
// into an explicit error so the literal "HEAD" never ends up in a push
// command.
func (g *Git) currentBranch(ctx context.Context, path string, opts *GitOptions) (string, error) {
	result, err := g.runGit(ctx, "rev-parse", []string{"rev-parse", "--abbrev-ref", "HEAD"}, path, opts)
	if err != nil {
		return "", err
	}
	name := strings.TrimSpace(result.Stdout)
	if name == "" || name == "HEAD" {
		return "", &InvalidArgumentError{Msg: "Cannot push with SetUpstream=true on a detached HEAD; specify Branch explicitly or set SetUpstream=false."}
	}
	return name, nil
}

// runWithOptionalCredentials runs push/pull-style remote-sync commands with
// optional credential injection. buildArgs receives the resolved target
// remote name (possibly empty) and returns the final argument list.
//
// Remote resolution is deliberately decoupled from the credential branch so
// the "with credentials" and "without credentials" call paths share the same
// semantics for single-remote repos: with no explicit Remote, a repository
// that happens to have exactly one remote auto-selects it.
func (g *Git) runWithOptionalCredentials(
	ctx context.Context,
	sub, path, remote, username, password string,
	opts *GitOptions,
	buildArgs func(target string) ([]string, error),
) (*CommandResult, error) {
	withCreds := username != "" && password != ""

	// Without credentials, use any explicit remote as is. With no explicit
	// remote, try to auto-select the single one; with zero or multiple
	// remotes fall back to git's default behavior so git's own error
	// message reaches the caller.
	if !withCreds {
		target := remote
		if target == "" {
			name, err := g.autoSelectRemote(ctx, path, opts)
			if err != nil {
				// If `git remote` itself fails (the path is not a repo,
				// etc.) return the raw error; otherwise buildArgs would
				// turn it into a misleading InvalidArgumentError.
				return nil, err
			}
			target = name
		}
		args, err := buildArgs(target)
		if err != nil {
			return nil, err
		}
		result, err := g.runGit(ctx, sub, args, path, opts)
		if err != nil {
			return nil, mapPushPullError(err, sub, username, password)
		}
		return result, nil
	}

	// With credentials we must resolve a concrete remote name (with URL) so
	// we can inject and restore credentials.
	remoteName, originalURL, err := g.resolveRemoteName(ctx, path, remote, opts)
	if err != nil {
		return nil, err
	}
	args, err := buildArgs(remoteName)
	if err != nil {
		return nil, err
	}
	var result *CommandResult
	err = g.withRemoteCredentials(ctx, path, remoteName, originalURL, username, password, opts, func() error {
		r, runErr := g.runGit(ctx, sub, args, path, opts)
		result = r
		return runErr
	})
	if err != nil {
		return nil, mapPushPullError(err, sub, username, password)
	}
	return result, nil
}

// autoSelectRemote returns (name, nil) when the repo has exactly one remote;
// returns ("", nil) for zero or multiple remotes so the caller can decide
// whether to fall back to git's default. When `git remote` itself fails
// (path is not a repo / unreachable / git crashes) it returns ("", err) so
// the real error surfaces.
func (g *Git) autoSelectRemote(ctx context.Context, path string, opts *GitOptions) (string, error) {
	remotes, err := g.listRemotes(ctx, path, opts)
	if err != nil {
		return "", err
	}
	if len(remotes) == 1 {
		return remotes[0], nil
	}
	return "", nil
}

// listRemotes returns the repository's currently configured remote names
// (blank lines and surrounding whitespace stripped).
func (g *Git) listRemotes(ctx context.Context, path string, opts *GitOptions) ([]string, error) {
	result, err := g.runGit(ctx, "remote", []string{"remote"}, path, opts)
	if err != nil {
		return nil, err
	}
	var remotes []string
	for _, line := range strings.Split(result.Stdout, "\n") {
		if s := strings.TrimSpace(line); s != "" {
			remotes = append(remotes, s)
		}
	}
	return remotes, nil
}

// RemoteAdd adds (or, when opts.Overwrite=true, replaces) a remote on the
// repository.
func (g *Git) RemoteAdd(ctx context.Context, path, name, repoURL string, opts *RemoteAddOptions) (*CommandResult, error) {
	if opts == nil {
		opts = &RemoteAddOptions{}
	}
	addArgs, err := buildRemoteAddArgs(name, repoURL)
	if err != nil {
		return nil, err
	}

	// Build the add-or-overwrite phase without fetch to keep the semantics
	// distinct from `add -f`'s fallback.
	var addPhase string
	if opts.Overwrite {
		addCmd := buildGitCommand(addArgs, path)
		setURLCmd := buildGitCommand(buildRemoteSetURLArgs(name, repoURL), path)
		addPhase = addCmd + " || " + setURLCmd
	} else {
		addPhase = buildGitCommand(addArgs, path)
	}

	// Without fetch, the add phase alone is enough.
	if !opts.Fetch {
		return g.runShell(ctx, "remote", addPhase, &opts.GitOptions)
	}

	// With fetch, combine into a single shell command so we only fetch once.
	fetchCmd := buildGitCommand([]string{"fetch", name}, path)
	return g.runShell(ctx, "remote", "("+addPhase+") && "+fetchCmd, &opts.GitOptions)
}

// RemoteGet returns the URL of the given remote. Returns an empty string when
// the remote is not configured.
func (g *Git) RemoteGet(ctx context.Context, path, name string, opts *GitOptions) (string, error) {
	if name == "" {
		return "", &InvalidArgumentError{Msg: "Remote name is required."}
	}
	result, err := g.runGit(ctx, "remote", buildRemoteGetURLArgs(name), path, opts)
	if err != nil {
		// `git remote get-url <name>` exits 2 when the remote does not
		// exist; other exit codes are real errors.
		var ce *gitCommandError
		if errors.As(err, &ce) && ce.Result != nil && ce.Result.ExitCode == 2 {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(result.Stdout), nil
}

// SetConfig sets a git config value. GitConfigScopeLocal requires opts.Path.
func (g *Git) SetConfig(ctx context.Context, key, value string, opts *ConfigOptions) (*CommandResult, error) {
	if key == "" {
		return nil, &InvalidArgumentError{Msg: "Git config key is required."}
	}
	if opts == nil {
		opts = &ConfigOptions{}
	}
	scope, repoPath, err := resolveConfigScope(opts.Scope, opts.Path)
	if err != nil {
		return nil, err
	}
	return g.runGit(ctx, "config", []string{"config", scope, key, value}, repoPath, &opts.GitOptions)
}

// GetConfig reads a git config value. Returns an empty string when the key is
// not configured.
func (g *Git) GetConfig(ctx context.Context, key string, opts *ConfigOptions) (string, error) {
	if key == "" {
		return "", &InvalidArgumentError{Msg: "Git config key is required."}
	}
	if opts == nil {
		opts = &ConfigOptions{}
	}
	scope, repoPath, err := resolveConfigScope(opts.Scope, opts.Path)
	if err != nil {
		return "", err
	}
	result, err := g.runGit(ctx, "config", []string{"config", scope, "--get", key}, repoPath, &opts.GitOptions)
	if err != nil {
		// `git config --get` exits 1 with empty stdout/stderr when no value
		// is set; other exit codes are real errors.
		var ce *gitCommandError
		if errors.As(err, &ce) && ce.Result != nil && ce.Result.ExitCode == 1 &&
			strings.TrimSpace(ce.Result.Stdout) == "" && strings.TrimSpace(ce.Result.Stderr) == "" {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(result.Stdout), nil
}

// ConfigureUser sets git's commit user name and email in a single RPC.
func (g *Git) ConfigureUser(ctx context.Context, name, email string, opts *ConfigOptions) (*CommandResult, error) {
	if name == "" || email == "" {
		return nil, &InvalidArgumentError{Msg: "Both name and email are required."}
	}
	if opts == nil {
		opts = &ConfigOptions{}
	}
	scope, repoPath, err := resolveConfigScope(opts.Scope, opts.Path)
	if err != nil {
		return nil, err
	}
	nameCmd := buildGitCommand([]string{"config", scope, "user.name", name}, repoPath)
	emailCmd := buildGitCommand([]string{"config", scope, "user.email", email}, repoPath)
	return g.runShell(ctx, "config", nameCmd+" && "+emailCmd, &opts.GitOptions)
}

// DangerouslyAuthenticate persists credentials to disk via the git credential
// helper.
//
// The credentials apply to every subsequent git operation. Prefer
// short-lived tokens and use this method only when necessary.
func (g *Git) DangerouslyAuthenticate(ctx context.Context, opts *AuthenticateOptions) (*CommandResult, error) {
	if opts == nil || opts.Username == "" || opts.Password == "" {
		return nil, &InvalidArgumentError{Msg: "Both username and password are required to authenticate git."}
	}

	host := strings.TrimSpace(opts.Host)
	if host == "" {
		host = "github.com"
	}
	protocol := strings.TrimSpace(opts.Protocol)
	if protocol == "" {
		protocol = "https"
	}
	if protocol != "https" {
		return nil, &InvalidArgumentError{Msg: "Only https protocol is supported for git authentication."}
	}

	if _, err := g.runGit(ctx, "config", []string{"config", "--global", "credential.helper", "store"}, "", &opts.GitOptions); err != nil {
		return nil, err
	}

	credentialInput := strings.Join([]string{
		"protocol=" + protocol,
		"host=" + host,
		"username=" + opts.Username,
		"password=" + opts.Password,
		"",
		"",
	}, "\n")
	cmd := fmt.Sprintf("printf '%%s' %s | %s", shellEscape(credentialInput), buildGitCommand([]string{"credential", "approve"}, ""))
	return g.runShell(ctx, "credential", cmd, &opts.GitOptions)
}

// resolveConfigScope validates and returns the git config scope flag and
// repository path.
func resolveConfigScope(scope GitConfigScope, repoPath string) (string, string, error) {
	if scope == "" {
		scope = GitConfigScopeGlobal
	}
	switch scope {
	case GitConfigScopeGlobal:
		return "--global", "", nil
	case GitConfigScopeSystem:
		return "--system", "", nil
	case GitConfigScopeLocal:
		if repoPath == "" {
			return "", "", &InvalidArgumentError{Msg: "Repository path is required for local git config scope."}
		}
		return "--local", repoPath, nil
	}
	return "", "", &InvalidArgumentError{Msg: "Unsupported git config scope: " + string(scope)}
}

// hasUpstream reports whether the current branch has upstream configured.
func (g *Git) hasUpstream(ctx context.Context, path string, opts *GitOptions) (bool, error) {
	_, err := g.runGit(ctx, "rev-parse", buildHasUpstreamArgs(), path, opts)
	if err == nil {
		return true, nil
	}
	// rev-parse exits 128 with a "no upstream" stderr when upstream is not
	// configured; other failures (bad path, git crash) bubble up unchanged.
	var ce *gitCommandError
	if errors.As(err, &ce) && ce.Result != nil && isMissingUpstream(err) {
		return false, nil
	}
	return false, err
}

// resolveRemoteName resolves the remote name used by credential-injection
// flows along with its URL.
//
// Aligned with E2B:
//   - When remote is explicit, use it directly.
//   - Otherwise list all remotes via `git remote`: exactly one auto-selects,
//     multiple require explicit selection (error).
//
// Returning the URL avoids an extra RPC inside withRemoteCredentials.
func (g *Git) resolveRemoteName(ctx context.Context, path, remote string, opts *GitOptions) (string, string, error) {
	name := remote
	if name == "" {
		remotes, err := g.listRemotes(ctx, path, opts)
		if err != nil {
			return "", "", err
		}
		switch len(remotes) {
		case 0:
			return "", "", &InvalidArgumentError{Msg: "Repository has no remote configured."}
		case 1:
			name = remotes[0]
		default:
			return "", "", &InvalidArgumentError{Msg: "Remote is required when using username/password and the repository has multiple remotes."}
		}
	}

	url, err := g.RemoteGet(ctx, path, name, opts)
	if err != nil {
		return "", "", err
	}
	if url == "" {
		return "", "", &InvalidArgumentError{Msg: fmt.Sprintf("Remote %q is not configured.", name)}
	}
	return name, url, nil
}

// withRemoteCredentials temporarily injects username/password into the given
// remote's URL, restores the original URL after fn returns (regardless of
// success). originalURL is supplied by the caller to avoid an extra RPC.
func (g *Git) withRemoteCredentials(ctx context.Context, path, remote, originalURL, username, password string, opts *GitOptions, fn func() error) (err error) {
	authedURL, err := withCredentials(originalURL, username, password)
	if err != nil {
		return err
	}
	if authedURL == originalURL {
		return fn()
	}

	if _, err = g.runGit(ctx, "remote", buildRemoteSetURLArgs(remote, authedURL), path, opts); err != nil {
		return err
	}
	defer func() {
		// Always restore the original URL even when ctx is cancelled, so
		// credentials are not left behind in .git/config. Use an
		// independent timeout-bounded context so the defer cannot block
		// forever on a sandbox glitch. URL restoration is a safety step:
		// the main error must not swallow it — use errors.Join to keep
		// both.
		cleanupCtx, cancel := context.WithTimeout(context.Background(), credentialCleanupTimeout)
		defer cancel()
		_, restoreErr := g.runGit(cleanupCtx, "remote", buildRemoteSetURLArgs(remote, originalURL), path, opts)
		err = errors.Join(err, restoreErr)
	}()
	return fn()
}

// runGit assembles and runs a git command.
func (g *Git) runGit(ctx context.Context, sub string, args []string, repoPath string, opts *GitOptions) (*CommandResult, error) {
	cmd := buildGitCommand(args, repoPath)
	return g.runShell(ctx, sub, cmd, opts)
}

// runShell runs a shell-form command with the default git env injected.
func (g *Git) runShell(ctx context.Context, sub, cmd string, opts *GitOptions) (*CommandResult, error) {
	cmdOpts := buildCommandOptions(opts)
	result, err := g.commands.Run(ctx, cmd, cmdOpts...)
	if err != nil {
		return nil, fmt.Errorf("git %s: %w", sub, err)
	}
	if result.ExitCode != 0 {
		return result, &gitCommandError{Cmd: sub, Result: result}
	}
	return result, nil
}

// buildCommandOptions converts GitOptions into a list of CommandOption.
func buildCommandOptions(opts *GitOptions) []CommandOption {
	capacity := len(defaultGitEnv)
	if opts != nil {
		capacity += len(opts.Envs)
	}
	mergedEnvs := make(map[string]string, capacity)
	cmdOpts := []CommandOption{}
	if opts != nil {
		maps.Copy(mergedEnvs, opts.Envs)
		if opts.User != "" {
			cmdOpts = append(cmdOpts, WithCommandUser(opts.User))
		}
		if opts.Cwd != "" {
			cmdOpts = append(cmdOpts, WithCwd(opts.Cwd))
		}
		if opts.Timeout > 0 {
			cmdOpts = append(cmdOpts, WithTimeout(opts.Timeout))
		}
	}
	// Default envs are written last so callers cannot override them (e.g.
	// disabling GIT_TERMINAL_PROMPT).
	maps.Copy(mergedEnvs, defaultGitEnv)
	cmdOpts = append(cmdOpts, WithEnvs(mergedEnvs))
	return cmdOpts
}

// mapPushPullError maps a command error into a typed git error for push/pull.
func mapPushPullError(err error, action, username, password string) error {
	if err == nil {
		return nil
	}
	if isAuthFailure(err) {
		return &GitAuthError{Msg: buildAuthErrorMessage(action, username != "" && password == ""), Err: err}
	}
	if isMissingUpstream(err) {
		return &GitUpstreamError{Msg: buildUpstreamErrorMessage(action), Err: err}
	}
	return err
}
