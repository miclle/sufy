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
	"strconv"
	"strings"
)

// shellEscape single-quotes a string so it can be embedded safely in a shell
// command.
func shellEscape(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

// buildGitCommand assembles a shell-safe git command string. When repoPath is
// non-empty, `-C <repoPath>` is appended.
func buildGitCommand(args []string, repoPath string) string {
	parts := make([]string, 0, len(args)+3)
	parts = append(parts, "git")
	if repoPath != "" {
		parts = append(parts, "-C", repoPath)
	}
	parts = append(parts, args...)
	for i, p := range parts {
		parts[i] = shellEscape(p)
	}
	return strings.Join(parts, " ")
}

// buildPushArgs builds the argument list for `git push`. target is the final
// remote name (merged from remoteName and the remote option).
func buildPushArgs(target, branch string, setUpstream bool) []string {
	args := []string{"push"}
	if setUpstream && target != "" {
		args = append(args, "--set-upstream")
	}
	if target != "" {
		args = append(args, target)
	}
	if branch != "" {
		args = append(args, branch)
	}
	return args
}

// buildPullArgs builds the argument list for `git pull`.
func buildPullArgs(target, branch string) []string {
	args := []string{"pull"}
	if target != "" {
		args = append(args, target)
	}
	if branch != "" {
		args = append(args, branch)
	}
	return args
}

// buildRemoteAddArgs builds the argument list for `git remote add`.
func buildRemoteAddArgs(name, url string) ([]string, error) {
	if name == "" || url == "" {
		return nil, &InvalidArgumentError{Msg: "Both remote name and URL are required to add a git remote."}
	}
	return []string{"remote", "add", name, url}, nil
}

// buildRemoteSetURLArgs builds the argument list for `git remote set-url`.
func buildRemoteSetURLArgs(name, url string) []string {
	return []string{"remote", "set-url", name, url}
}

// buildRemoteGetURLArgs builds the argument list for `git remote get-url`.
func buildRemoteGetURLArgs(name string) []string {
	return []string{"remote", "get-url", name}
}

// buildHasUpstreamArgs builds the argument list used to detect upstream
// configuration.
func buildHasUpstreamArgs() []string {
	return []string{"rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}"}
}

// buildStatusArgs builds the argument list for `git status`.
func buildStatusArgs() []string {
	return []string{"status", "--porcelain=1", "-b"}
}

// buildBranchesArgs builds the argument list for listing local branches.
func buildBranchesArgs() []string {
	return []string{"branch", "--format=%(refname:short)\t%(HEAD)"}
}

// buildCreateBranchArgs builds the argument list for `git checkout -b`.
func buildCreateBranchArgs(branch string) []string {
	return []string{"checkout", "-b", branch}
}

// buildCheckoutBranchArgs builds the argument list for `git checkout`.
func buildCheckoutBranchArgs(branch string) []string {
	return []string{"checkout", branch}
}

// buildDeleteBranchArgs builds the argument list for `git branch -d/-D`.
func buildDeleteBranchArgs(branch string, force bool) []string {
	flag := "-d"
	if force {
		flag = "-D"
	}
	return []string{"branch", flag, branch}
}

// buildAddArgs builds the argument list for `git add`.
func buildAddArgs(files []string, all bool) []string {
	args := []string{"add"}
	if len(files) == 0 {
		if all {
			args = append(args, "-A")
		} else {
			args = append(args, ".")
		}
		return args
	}
	args = append(args, "--")
	args = append(args, files...)
	return args
}

// buildCommitArgs builds the argument list for `git commit`.
func buildCommitArgs(message, authorName, authorEmail string, allowEmpty bool) []string {
	args := []string{"commit", "-m", message}
	if allowEmpty {
		args = append(args, "--allow-empty")
	}
	var prefix []string
	if authorName != "" {
		prefix = append(prefix, "-c", "user.name="+authorName)
	}
	if authorEmail != "" {
		prefix = append(prefix, "-c", "user.email="+authorEmail)
	}
	if len(prefix) > 0 {
		return append(prefix, args...)
	}
	return args
}

// allowedResetModes is the set of valid `git reset` modes.
var allowedResetModes = map[GitResetMode]struct{}{
	GitResetModeSoft:  {},
	GitResetModeMixed: {},
	GitResetModeHard:  {},
	GitResetModeMerge: {},
	GitResetModeKeep:  {},
}

// buildResetArgs builds the argument list for `git reset`. The two usages are
// mutually exclusive: with mode the command resets HEAD/index/worktree; with
// paths only those paths are unstaged.
func buildResetArgs(mode GitResetMode, target string, paths []string) ([]string, error) {
	if mode != "" {
		if _, ok := allowedResetModes[mode]; !ok {
			return nil, &InvalidArgumentError{Msg: "Reset mode must be one of soft, mixed, hard, merge, keep."}
		}
	}
	if mode != "" && len(paths) > 0 {
		return nil, &InvalidArgumentError{Msg: "Reset mode and paths cannot be used together."}
	}
	args := []string{"reset"}
	if mode != "" {
		args = append(args, "--"+string(mode))
	}
	if target != "" {
		args = append(args, target)
	}
	if len(paths) > 0 {
		args = append(args, "--")
		args = append(args, paths...)
	}
	return args, nil
}

// buildRestoreArgs builds the argument list for `git restore`. Aligned with
// native git: when no flag is explicit, only the worktree is restored. When
// one flag is explicit false and the other is unset, also fall back to the
// worktree default rather than reporting a "must be true" error. Only when
// both are explicit false does the call fail.
func buildRestoreArgs(paths []string, staged, worktree *bool, source string) ([]string, error) {
	if len(paths) == 0 {
		return nil, &InvalidArgumentError{Msg: "At least one path is required."}
	}

	stagedOn := staged != nil && *staged
	worktreeOn := worktree != nil && *worktree
	if !stagedOn && !worktreeOn && (staged == nil || worktree == nil) {
		worktreeOn = true
	}

	if !stagedOn && !worktreeOn {
		return nil, &InvalidArgumentError{Msg: "At least one of staged or worktree must be true."}
	}

	args := []string{"restore"}
	if worktreeOn {
		args = append(args, "--worktree")
	}
	if stagedOn {
		args = append(args, "--staged")
	}
	if source != "" {
		args = append(args, "--source", source)
	}
	args = append(args, "--")
	args = append(args, paths...)
	return args, nil
}

// clonePlan describes a complete clone execution plan.
type clonePlan struct {
	// args is the argument list for `git clone`.
	args []string
	// repoPath is the repository path used by post-clone adjustments.
	repoPath string
	// sanitizedURL is the credential-stripped URL used to reset origin after
	// the clone completes.
	sanitizedURL string
	// shouldStrip indicates whether the origin URL must be reset after clone.
	shouldStrip bool
}

// buildClonePlan builds the clone arguments along with credential-strip
// metadata.
func buildClonePlan(rawURL, path, branch string, depth int, username, password string, dangerouslyStore bool) (*clonePlan, error) {
	// Reject credentials embedded in plain HTTP URLs as well, matching the
	// safety boundary enforced by withCredentials. Without this guard a
	// `http://user:pass@host/...` URL would bypass the HTTPS requirement.
	if err := requireHTTPSIfHasCredentials(rawURL); err != nil {
		return nil, err
	}
	cloneURL := rawURL
	if username != "" && password != "" {
		var err error
		cloneURL, err = withCredentials(rawURL, username, password)
		if err != nil {
			return nil, err
		}
	}
	sanitized := stripCredentials(cloneURL)
	shouldStrip := !dangerouslyStore && sanitized != cloneURL

	repoPath := path
	if shouldStrip {
		if repoPath == "" {
			repoPath = deriveRepoDirFromURL(rawURL)
		}
		if repoPath == "" {
			return nil, &InvalidArgumentError{Msg: "A destination path is required when using credentials without storing them."}
		}
	}

	args := []string{"clone", cloneURL}
	if branch != "" {
		args = append(args, "--branch", branch, "--single-branch")
	}
	if depth > 0 {
		args = append(args, "--depth", strconv.Itoa(depth))
	}
	if path != "" {
		args = append(args, path)
	}

	plan := &clonePlan{args: args, repoPath: repoPath}
	if shouldStrip {
		plan.sanitizedURL = sanitized
		plan.shouldStrip = true
	}
	return plan, nil
}
