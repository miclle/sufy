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

//go:build integration

package sandbox

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gitTestEnv groups the objects commonly used across a git integration test
// so each case does not have to provision a sandbox by itself.
type gitTestEnv struct {
	t   *testing.T
	sb  *Sandbox
	git *Git
	ctx context.Context
}

// newGitTestEnv creates a ready-to-use sandbox and registers cleanup.
func newGitTestEnv(t *testing.T) *gitTestEnv {
	t.Helper()
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 240*time.Second)
	t.Cleanup(cancel)
	sb := createTestSandbox(t, c, ctx)
	return &gitTestEnv{t: t, sb: sb, git: sb.Git(), ctx: ctx}
}

// initRepo initializes a normal repository inside the sandbox and configures
// the author identity.
func (e *gitTestEnv) initRepo(path, branch string) {
	e.t.Helper()
	_, err := e.git.Init(e.ctx, path, &InitOptions{InitialBranch: branch})
	require.NoError(e.t, err, "Init %s", path)
	_, err = e.git.ConfigureUser(e.ctx, "Tester", "tester@example.com", &ConfigOptions{
		Scope: GitConfigScopeLocal,
		Path:  path,
	})
	require.NoError(e.t, err, "ConfigureUser")
}

// writeAndCommit writes a file and produces a single commit.
func (e *gitTestEnv) writeAndCommit(repo, file, content, msg string) {
	e.t.Helper()
	_, err := e.sb.Files().Write(e.ctx, repo+"/"+file, []byte(content))
	require.NoError(e.t, err, "Write %s", file)
	_, err = e.git.Add(e.ctx, repo, nil)
	require.NoError(e.t, err, "Add")
	_, err = e.git.Commit(e.ctx, repo, msg, nil)
	require.NoError(e.t, err, "Commit")
}

// TestIntegrationGitInitConfigureBranches exercises Init / ConfigureUser /
// SetConfig / GetConfig / Branches (including the unborn-repo path) /
// CreateBranch / CheckoutBranch / DeleteBranch.
func TestIntegrationGitInitConfigureBranches(t *testing.T) {
	e := newGitTestEnv(t)

	repo := "/tmp/it-init"
	_, err := e.git.Init(e.ctx, repo, &InitOptions{InitialBranch: "main"})
	require.NoError(t, err)

	// Unborn repo: Branches() must fall back to symbolic-ref and return "main".
	br, err := e.git.Branches(e.ctx, repo, nil)
	require.NoError(t, err)
	assert.Empty(t, br.Branches, "unborn repo should have no enumerable branches")
	assert.Equal(t, "main", br.CurrentBranch, "unborn repo fallback should yield main")

	_, err = e.git.ConfigureUser(e.ctx, "Alice", "a@x.io", &ConfigOptions{
		Scope: GitConfigScopeLocal, Path: repo,
	})
	require.NoError(t, err)
	val, err := e.git.GetConfig(e.ctx, "user.name", &ConfigOptions{Scope: GitConfigScopeLocal, Path: repo})
	require.NoError(t, err)
	assert.Equal(t, "Alice", val)

	// A missing key must return empty string and no error.
	val, err = e.git.GetConfig(e.ctx, "user.notexist", &ConfigOptions{Scope: GitConfigScopeLocal, Path: repo})
	require.NoError(t, err)
	assert.Empty(t, val)

	// SetConfig with an arbitrary key.
	_, err = e.git.SetConfig(e.ctx, "core.autocrlf", "input", &ConfigOptions{Scope: GitConfigScopeLocal, Path: repo})
	require.NoError(t, err)
	val, err = e.git.GetConfig(e.ctx, "core.autocrlf", &ConfigOptions{Scope: GitConfigScopeLocal, Path: repo})
	require.NoError(t, err)
	assert.Equal(t, "input", val)

	// Local scope without Path must error out directly.
	_, err = e.git.SetConfig(e.ctx, "k", "v", &ConfigOptions{Scope: GitConfigScopeLocal})
	var ie *InvalidArgumentError
	assert.True(t, errors.As(err, &ie), "Local scope without Path should return InvalidArgumentError")

	// Branch operations require the first commit to exist.
	e.writeAndCommit(repo, "README.md", "# init\n", "feat: init")

	_, err = e.git.CreateBranch(e.ctx, repo, "feature/x", nil)
	require.NoError(t, err)
	br, err = e.git.Branches(e.ctx, repo, nil)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"main", "feature/x"}, br.Branches)
	assert.Equal(t, "feature/x", br.CurrentBranch)

	_, err = e.git.CheckoutBranch(e.ctx, repo, "main", nil)
	require.NoError(t, err)
	_, err = e.git.DeleteBranch(e.ctx, repo, "feature/x", &DeleteBranchOptions{Force: true})
	require.NoError(t, err)

	br, err = e.git.Branches(e.ctx, repo, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"main"}, br.Branches)
	assert.Equal(t, "main", br.CurrentBranch)

	// An empty branch name must be caught by argument validation.
	_, err = e.git.CreateBranch(e.ctx, repo, "", nil)
	assert.True(t, errors.As(err, &ie))
}

// TestIntegrationGitAddOptions covers all three settings of AddOptions.All:
// nil defaulting to -A, explicit false using ".", and an explicit Files list.
func TestIntegrationGitAddOptions(t *testing.T) {
	e := newGitTestEnv(t)

	repo := "/tmp/it-add"
	e.initRepo(repo, "main")
	e.writeAndCommit(repo, "seed", "seed\n", "seed")

	// Place a new file in a subdirectory and one at the repo root.
	_, err := e.sb.Files().Write(e.ctx, repo+"/top.txt", []byte("top\n"))
	require.NoError(t, err)
	_, err = e.sb.Files().Write(e.ctx, repo+"/sub/inner.txt", []byte("inner\n"))
	require.NoError(t, err)

	// 1) All=false: equivalent to `git add .` — at the repo root it should
	//    recurse into subdirectories (git's native behavior). We only assert
	//    the call succeeds and Status reports at least one staged file.
	allFalse := false
	_, err = e.git.Add(e.ctx, repo, &AddOptions{All: &allFalse})
	require.NoError(t, err)
	st, err := e.git.Status(e.ctx, repo, nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, st.StagedCount(), 1)

	// 2) Files=[]: only the specified files are staged.
	_, err = e.git.Reset(e.ctx, repo, &ResetOptions{Paths: []string{"."}})
	require.NoError(t, err)
	_, err = e.git.Add(e.ctx, repo, &AddOptions{Files: []string{"top.txt"}})
	require.NoError(t, err)
	st, err = e.git.Status(e.ctx, repo, nil)
	require.NoError(t, err)
	staged := stagedNames(st)
	assert.Contains(t, staged, "top.txt")
	assert.NotContains(t, staged, "sub/inner.txt")

	// 3) opts=nil: defaults to -A, staging every untracked file.
	_, err = e.git.Add(e.ctx, repo, nil)
	require.NoError(t, err)
	st, err = e.git.Status(e.ctx, repo, nil)
	require.NoError(t, err)
	staged = stagedNames(st)
	assert.Contains(t, staged, "top.txt")
	assert.Contains(t, staged, "sub/inner.txt")
}

// TestIntegrationGitStatusEdgeCases checks parseGitStatus against real git
// output: detached HEAD, spaces in file names, MM (simultaneous staged +
// unstaged), and unborn repos.
func TestIntegrationGitStatusEdgeCases(t *testing.T) {
	e := newGitTestEnv(t)

	repo := "/tmp/it-status"
	e.initRepo(repo, "main")

	// Unborn: CurrentBranch=main and Detached=false should be parsed.
	st, err := e.git.Status(e.ctx, repo, nil)
	require.NoError(t, err)
	assert.False(t, st.Detached)
	assert.Equal(t, "main", st.CurrentBranch)

	e.writeAndCommit(repo, "README.md", "# v1\n", "feat: init")

	// MM: same file staged + worktree modified again.
	_, err = e.sb.Files().Write(e.ctx, repo+"/README.md", []byte("# v2\n"))
	require.NoError(t, err)
	_, err = e.git.Add(e.ctx, repo, nil)
	require.NoError(t, err)
	_, err = e.sb.Files().Write(e.ctx, repo+"/README.md", []byte("# v2 dirty\n"))
	require.NoError(t, err)

	// A file name with spaces (untracked).
	_, err = e.sb.Files().Write(e.ctx, repo+"/with space.txt", []byte("x\n"))
	require.NoError(t, err)

	st, err = e.git.Status(e.ctx, repo, nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, st.StagedCount(), 1, "README.md should also count as staged")
	assert.GreaterOrEqual(t, st.UnstagedCount(), 1, "README.md worktree change should count as unstaged")
	assert.True(t, st.HasUntracked())

	byName := map[string]GitFileStatus{}
	for _, f := range st.FileStatus {
		byName[f.Name] = f
	}
	assert.Contains(t, byName, "with space.txt", "file names with spaces should be unquoted")
	if rm, ok := byName["README.md"]; ok {
		assert.True(t, rm.Staged, "README.md should be marked staged")
	}

	// Detached HEAD.
	res, err := e.sb.Commands().Run(e.ctx, "git -C "+repo+" rev-parse HEAD",
		WithEnvs(map[string]string{"GIT_TERMINAL_PROMPT": "0"}))
	require.NoError(t, err)
	sha := strings.TrimSpace(res.Stdout)
	require.NotEmpty(t, sha)
	_, err = e.sb.Commands().Run(e.ctx, "git -C "+repo+" checkout "+sha,
		WithEnvs(map[string]string{"GIT_TERMINAL_PROMPT": "0"}))
	require.NoError(t, err)
	st, err = e.git.Status(e.ctx, repo, nil)
	require.NoError(t, err)
	assert.True(t, st.Detached, "checking out <sha> should enter detached HEAD")
	assert.Empty(t, st.CurrentBranch)
}

// TestIntegrationGitResetRestore covers Reset modes, Restore --staged and
// --source HEAD, plus invalid argument combinations.
func TestIntegrationGitResetRestore(t *testing.T) {
	e := newGitTestEnv(t)

	repo := "/tmp/it-reset"
	e.initRepo(repo, "main")
	e.writeAndCommit(repo, "a.txt", "v1\n", "init")

	// --hard: discard worktree changes.
	_, err := e.sb.Files().Write(e.ctx, repo+"/a.txt", []byte("dirty\n"))
	require.NoError(t, err)
	_, err = e.git.Reset(e.ctx, repo, &ResetOptions{Mode: GitResetModeHard, Target: "HEAD"})
	require.NoError(t, err)
	got, err := e.sb.Files().ReadText(e.ctx, repo+"/a.txt")
	require.NoError(t, err)
	assert.Equal(t, "v1\n", got)

	// Paths-only: unstage.
	_, err = e.sb.Files().Write(e.ctx, repo+"/a.txt", []byte("staged\n"))
	require.NoError(t, err)
	_, err = e.git.Add(e.ctx, repo, nil)
	require.NoError(t, err)
	_, err = e.git.Reset(e.ctx, repo, &ResetOptions{Paths: []string{"a.txt"}})
	require.NoError(t, err)
	st, err := e.git.Status(e.ctx, repo, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, st.StagedCount())

	// Mode + Paths → InvalidArgumentError.
	_, err = e.git.Reset(e.ctx, repo, &ResetOptions{Mode: GitResetModeHard, Paths: []string{"a.txt"}})
	var ie *InvalidArgumentError
	assert.True(t, errors.As(err, &ie))

	// Invalid mode → InvalidArgumentError.
	_, err = e.git.Reset(e.ctx, repo, &ResetOptions{Mode: GitResetMode("bogus")})
	assert.True(t, errors.As(err, &ie))

	// Restore --staged: unstage only; worktree is preserved.
	_, err = e.sb.Files().Write(e.ctx, repo+"/a.txt", []byte("again\n"))
	require.NoError(t, err)
	_, err = e.git.Add(e.ctx, repo, nil)
	require.NoError(t, err)
	staged := true
	_, err = e.git.Restore(e.ctx, repo, &RestoreOptions{Paths: []string{"a.txt"}, Staged: &staged})
	require.NoError(t, err)
	got, err = e.sb.Files().ReadText(e.ctx, repo+"/a.txt")
	require.NoError(t, err)
	assert.Equal(t, "again\n", got, "Restore --staged should not touch the worktree")
	st, err = e.git.Status(e.ctx, repo, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, st.StagedCount())

	// Restore --source HEAD: roll the worktree back to HEAD.
	_, err = e.git.Restore(e.ctx, repo, &RestoreOptions{Paths: []string{"a.txt"}, Source: "HEAD"})
	require.NoError(t, err)
	got, err = e.sb.Files().ReadText(e.ctx, repo+"/a.txt")
	require.NoError(t, err)
	assert.Equal(t, "v1\n", got)

	// Empty Paths → error.
	_, err = e.git.Restore(e.ctx, repo, &RestoreOptions{})
	assert.Error(t, err)
}

// TestIntegrationGitCommitOptions covers CommitOptions: Author overrides and
// AllowEmpty.
func TestIntegrationGitCommitOptions(t *testing.T) {
	e := newGitTestEnv(t)
	repo := "/tmp/it-commit"
	e.initRepo(repo, "main")
	e.writeAndCommit(repo, "seed", "seed\n", "seed")

	// Empty message → rejected.
	_, err := e.git.Commit(e.ctx, repo, "", nil)
	var ie *InvalidArgumentError
	assert.True(t, errors.As(err, &ie))

	// Author override.
	_, err = e.sb.Files().Write(e.ctx, repo+"/b.txt", []byte("b\n"))
	require.NoError(t, err)
	_, err = e.git.Add(e.ctx, repo, nil)
	require.NoError(t, err)
	_, err = e.git.Commit(e.ctx, repo, "feat: b", &CommitOptions{
		AuthorName: "Bob", AuthorEmail: "bob@x.io",
	})
	require.NoError(t, err)
	res, err := e.sb.Commands().Run(e.ctx, "git -C "+repo+" log -1 --pretty=%an",
		WithEnvs(map[string]string{"GIT_TERMINAL_PROMPT": "0"}))
	require.NoError(t, err)
	assert.Equal(t, "Bob", strings.TrimSpace(res.Stdout))

	// AllowEmpty=true: commit succeeds even on a clean worktree.
	_, err = e.git.Commit(e.ctx, repo, "chore: empty", &CommitOptions{AllowEmpty: true})
	require.NoError(t, err)

	// AllowEmpty=false (default): clean worktree should fail.
	_, err = e.git.Commit(e.ctx, repo, "chore: should fail", nil)
	assert.Error(t, err)
}

// TestIntegrationGitRemoteAndPushPull covers RemoteAdd (with Overwrite) /
// RemoteGet (missing) / Push (SetUpstream default true and explicit false) /
// Pull / argument validation.
func TestIntegrationGitRemoteAndPushPull(t *testing.T) {
	e := newGitTestEnv(t)

	repo := "/tmp/it-remote"
	bare := "/tmp/it-remote.git"
	consumer := "/tmp/it-consumer"
	e.initRepo(repo, "main")
	e.writeAndCommit(repo, "README.md", "# v1\n", "init")
	_, err := e.git.Init(e.ctx, bare, &InitOptions{Bare: true, InitialBranch: "main"})
	require.NoError(t, err)

	// RemoteGet missing → empty string, no error.
	url, err := e.git.RemoteGet(e.ctx, repo, "nope", nil)
	require.NoError(t, err)
	assert.Empty(t, url)

	// RemoteAdd a placeholder URL, then Overwrite to the bare path.
	_, err = e.git.RemoteAdd(e.ctx, repo, "origin", "https://example.com/x.git", nil)
	require.NoError(t, err)
	_, err = e.git.RemoteAdd(e.ctx, repo, "origin", bare, &RemoteAddOptions{Overwrite: true})
	require.NoError(t, err)
	url, err = e.git.RemoteGet(e.ctx, repo, "origin", nil)
	require.NoError(t, err)
	assert.Equal(t, bare, url)

	// Push with no Remote but a Branch: a single-remote repo should
	// auto-select origin and push successfully.
	_, err = e.git.Push(e.ctx, repo, &PushOptions{Branch: "main"})
	require.NoError(t, err, "Push should auto-select origin on a single-remote repo")

	// Push with explicit SetUpstream=false (do not record upstream).
	noUpstream := false
	_, err = e.git.Push(e.ctx, repo, &PushOptions{
		Remote: "origin", Branch: "main", SetUpstream: &noUpstream,
	})
	require.NoError(t, err)

	// Clone the consumer repository.
	_, err = e.git.Clone(e.ctx, bare, &CloneOptions{Path: consumer})
	require.NoError(t, err)

	// Push from the primary repo again, this time with default SetUpstream=true.
	e.writeAndCommit(repo, "CHANGELOG.md", "v1\n", "docs")
	_, err = e.git.Push(e.ctx, repo, &PushOptions{Remote: "origin", Branch: "main"})
	require.NoError(t, err)

	// consumer Pull — no Branch / no Remote: hasUpstream must take the default path.
	_, err = e.git.Pull(e.ctx, consumer, &PullOptions{})
	require.NoError(t, err)
	exists, err := e.sb.Files().Exists(e.ctx, consumer+"/CHANGELOG.md")
	require.NoError(t, err)
	assert.True(t, exists)

	// Pull with no Remote but a Branch: single-remote repo auto-selects origin.
	_, err = e.git.Pull(e.ctx, consumer, &PullOptions{Branch: "main"})
	require.NoError(t, err, "Pull should auto-select origin on a single-remote repo")
}

// TestIntegrationGitPullMissingUpstream verifies Pull raises GitUpstreamError
// when the local branch has no upstream.
func TestIntegrationGitPullMissingUpstream(t *testing.T) {
	e := newGitTestEnv(t)
	repo := "/tmp/it-no-upstream"
	e.initRepo(repo, "main")
	e.writeAndCommit(repo, "a", "a\n", "init")

	_, err := e.git.Pull(e.ctx, repo, &PullOptions{})
	var ue *GitUpstreamError
	assert.True(t, errors.As(err, &ue), "missing-upstream Pull should return GitUpstreamError; got: %T %v", err, err)
}

// TestIntegrationGitPushPullBranchWithoutRemote verifies that on multi-remote
// or no-remote repos, specifying only Branch returns InvalidArgumentError
// rather than passing the branch name to git as a remote name.
func TestIntegrationGitPushPullBranchWithoutRemote(t *testing.T) {
	e := newGitTestEnv(t)

	// 1) No-remote repository.
	noRemote := "/tmp/it-no-remote"
	e.initRepo(noRemote, "main")
	e.writeAndCommit(noRemote, "a", "a\n", "init")

	_, err := e.git.Push(e.ctx, noRemote, &PushOptions{Branch: "main"})
	var ie1 *InvalidArgumentError
	assert.True(t, errors.As(err, &ie1), "no-remote Push(Branch:main) should return InvalidArgumentError; got: %T %v", err, err)

	_, err = e.git.Pull(e.ctx, noRemote, &PullOptions{Branch: "main"})
	var ie2 *InvalidArgumentError
	assert.True(t, errors.As(err, &ie2), "no-remote Pull(Branch:main) should return InvalidArgumentError; got: %T %v", err, err)

	// 2) Multi-remote repository.
	multi := "/tmp/it-multi-remote"
	e.initRepo(multi, "main")
	e.writeAndCommit(multi, "a", "a\n", "init")
	bare1 := "/tmp/it-multi-bare1.git"
	bare2 := "/tmp/it-multi-bare2.git"
	_, err = e.git.Init(e.ctx, bare1, &InitOptions{Bare: true, InitialBranch: "main"})
	require.NoError(t, err)
	_, err = e.git.Init(e.ctx, bare2, &InitOptions{Bare: true, InitialBranch: "main"})
	require.NoError(t, err)
	_, err = e.git.RemoteAdd(e.ctx, multi, "origin", bare1, nil)
	require.NoError(t, err)
	_, err = e.git.RemoteAdd(e.ctx, multi, "backup", bare2, nil)
	require.NoError(t, err)

	_, err = e.git.Push(e.ctx, multi, &PushOptions{Branch: "main"})
	var ie3 *InvalidArgumentError
	assert.True(t, errors.As(err, &ie3), "multi-remote Push(Branch:main) should return InvalidArgumentError; got: %T %v", err, err)

	_, err = e.git.Pull(e.ctx, multi, &PullOptions{Branch: "main"})
	var ie4 *InvalidArgumentError
	assert.True(t, errors.As(err, &ie4), "multi-remote Pull(Branch:main) should return InvalidArgumentError; got: %T %v", err, err)

	// Explicit Remote keeps things working.
	_, err = e.git.Push(e.ctx, multi, &PushOptions{Remote: "origin", Branch: "main"})
	require.NoError(t, err, "multi-remote repo with explicit Remote should Push successfully")
}

// TestIntegrationGitPushPullSurfacesRepoErrors verifies that when path is not
// a git repository, Push/Pull surface the real repository error instead of
// having autoSelectRemote swallow it into InvalidArgumentError.
func TestIntegrationGitPushPullSurfacesRepoErrors(t *testing.T) {
	e := newGitTestEnv(t)
	notRepo := "/tmp/it-not-a-repo"
	_, err := e.sb.Files().MakeDir(e.ctx, notRepo)
	require.NoError(t, err)

	_, err = e.git.Push(e.ctx, notRepo, &PushOptions{Branch: "main"})
	require.Error(t, err)
	var ie1 *InvalidArgumentError
	assert.False(t, errors.As(err, &ie1), "Push error on non-repo dir must not be masked as InvalidArgumentError; got: %T %v", err, err)

	_, err = e.git.Pull(e.ctx, notRepo, &PullOptions{Branch: "main"})
	require.Error(t, err)
	var ie2 *InvalidArgumentError
	assert.False(t, errors.As(err, &ie2), "Pull error on non-repo dir must not be masked as InvalidArgumentError; got: %T %v", err, err)
}

// TestIntegrationGitDangerouslyAuthenticate verifies credentials are
// persisted by the git credential helper.
func TestIntegrationGitDangerouslyAuthenticate(t *testing.T) {
	e := newGitTestEnv(t)

	// Non-https protocol → InvalidArgumentError.
	_, err := e.git.DangerouslyAuthenticate(e.ctx, &AuthenticateOptions{
		Username: "u", Password: "p", Host: "h", Protocol: "ssh",
	})
	var ie *InvalidArgumentError
	assert.True(t, errors.As(err, &ie))

	// Write credentials (inside the sandbox only).
	_, err = e.git.DangerouslyAuthenticate(e.ctx, &AuthenticateOptions{
		Username: "demo-user", Password: "demo-token", Host: "fake-host.example",
	})
	require.NoError(t, err)

	// Verify with `git credential fill`: feed protocol/host and git fills in
	// username/password from the store.
	input := "protocol=https\nhost=fake-host.example\n\n"
	res, err := e.sb.Commands().Run(e.ctx,
		"printf %s '"+input+"' | git credential fill",
		WithEnvs(map[string]string{"GIT_TERMINAL_PROMPT": "0"}),
	)
	require.NoError(t, err)
	assert.Equal(t, 0, res.ExitCode, "credential fill should succeed; stdout=%q stderr=%q", res.Stdout, res.Stderr)
	assert.Contains(t, res.Stdout, "username=demo-user")
	assert.Contains(t, res.Stdout, "password=demo-token")
}

// TestIntegrationGitCloneStripsCredentials verifies that, by default, clone
// leaves no credentials in origin's URL. Runs only when SUFY_GIT_REPO_URL /
// USERNAME / PASSWORD are provided.
func TestIntegrationGitCloneStripsCredentials(t *testing.T) {
	repoURL, username, password := getGitCredsFromEnv(t)
	e := newGitTestEnv(t)

	clonePath := "/tmp/it-clone-strip"
	_, err := e.git.Clone(e.ctx, repoURL, &CloneOptions{
		Path: clonePath, Depth: 1, Username: username, Password: password,
	})
	require.NoError(t, err)
	got, err := e.git.RemoteGet(e.ctx, clonePath, "origin", nil)
	require.NoError(t, err)
	assert.NotContains(t, got, username+":", "by default the cloned origin URL should not carry credentials")
	assert.NotContains(t, got, password)
}

// TestIntegrationGitCloneBranch simulates a multi-branch remote with an
// in-sandbox bare repo, and verifies CloneOptions.Branch makes HEAD point at
// the requested branch while origin's URL retains no credentials.
func TestIntegrationGitCloneBranch(t *testing.T) {
	e := newGitTestEnv(t)

	source := "/tmp/it-clone-src"
	bare := "/tmp/it-clone-src.git"
	dst := "/tmp/it-clone-dst"

	e.initRepo(source, "main")
	e.writeAndCommit(source, "main.txt", "main\n", "feat: main")

	// Create a release branch and add a commit on source.
	_, err := e.git.CreateBranch(e.ctx, source, "release", nil)
	require.NoError(t, err)
	e.writeAndCommit(source, "release.txt", "release\n", "feat: release")
	_, err = e.git.CheckoutBranch(e.ctx, source, "main", nil)
	require.NoError(t, err)

	// Push source to the bare repo, which acts as the "remote".
	_, err = e.git.Init(e.ctx, bare, &InitOptions{Bare: true, InitialBranch: "main"})
	require.NoError(t, err)
	_, err = e.git.RemoteAdd(e.ctx, source, "origin", bare, nil)
	require.NoError(t, err)
	_, err = e.git.Push(e.ctx, source, &PushOptions{Remote: "origin", Branch: "main"})
	require.NoError(t, err)
	_, err = e.git.Push(e.ctx, source, &PushOptions{Remote: "origin", Branch: "release"})
	require.NoError(t, err)

	// Clone with CloneOptions.Branch fetching release.
	_, err = e.git.Clone(e.ctx, bare, &CloneOptions{Path: dst, Branch: "release"})
	require.NoError(t, err)

	// HEAD should point at release.
	br, err := e.git.Branches(e.ctx, dst, nil)
	require.NoError(t, err)
	assert.Equal(t, "release", br.CurrentBranch)
	exists, err := e.sb.Files().Exists(e.ctx, dst+"/release.txt")
	require.NoError(t, err)
	assert.True(t, exists, "with Branch=release, release.txt should exist")

	// origin URL must not contain any user info (no credential leakage).
	origin, err := e.git.RemoteGet(e.ctx, dst, "origin", nil)
	require.NoError(t, err)
	assert.NotContains(t, origin, "@")
}

// TestIntegrationGitRemoteAddFetch verifies that RemoteAddOptions.Fetch=true
// fetches refs right after adding the remote so origin/<branch> becomes
// resolvable by rev-parse.
func TestIntegrationGitRemoteAddFetch(t *testing.T) {
	e := newGitTestEnv(t)

	source := "/tmp/it-fetch-src"
	bare := "/tmp/it-fetch-src.git"
	consumer := "/tmp/it-fetch-consumer"

	e.initRepo(source, "main")
	e.writeAndCommit(source, "a.txt", "a\n", "init")
	_, err := e.git.Init(e.ctx, bare, &InitOptions{Bare: true, InitialBranch: "main"})
	require.NoError(t, err)
	_, err = e.git.RemoteAdd(e.ctx, source, "origin", bare, nil)
	require.NoError(t, err)
	_, err = e.git.Push(e.ctx, source, &PushOptions{Remote: "origin", Branch: "main"})
	require.NoError(t, err)

	// consumer: init then RemoteAdd with Fetch=false → origin/main must not
	// resolve.
	_, err = e.git.Init(e.ctx, consumer, &InitOptions{InitialBranch: "main"})
	require.NoError(t, err)
	_, err = e.git.RemoteAdd(e.ctx, consumer, "origin", bare, nil)
	require.NoError(t, err)

	res, err := e.sb.Commands().Run(e.ctx, "git -C "+consumer+" rev-parse --verify -q origin/main",
		WithEnvs(map[string]string{"GIT_TERMINAL_PROMPT": "0"}))
	require.NoError(t, err)
	assert.NotEqual(t, 0, res.ExitCode, "with Fetch=false, origin/main should not resolve")

	// Re-add with Overwrite=true + Fetch=true (same origin).
	_, err = e.git.RemoteAdd(e.ctx, consumer, "origin", bare, &RemoteAddOptions{
		Overwrite: true, Fetch: true,
	})
	require.NoError(t, err)

	res, err = e.sb.Commands().Run(e.ctx, "git -C "+consumer+" rev-parse --verify -q origin/main",
		WithEnvs(map[string]string{"GIT_TERMINAL_PROMPT": "0"}))
	require.NoError(t, err)
	assert.Equal(t, 0, res.ExitCode, "with Fetch=true, origin/main should be fetched; stderr=%q", res.Stderr)
	assert.NotEmpty(t, strings.TrimSpace(res.Stdout))
}

// TestIntegrationGitOptionsEnvsCwdTimeout verifies the real effects of
// GitOptions.Envs / Cwd / Timeout:
//   - Envs: a custom GIT_AUTHOR_DATE reaches commit; GIT_TERMINAL_PROMPT=0
//     cannot be overridden by callers.
//   - Cwd: Init with a relative path under cwd creates the repo inside cwd.
//   - Timeout: a tiny timeout causes the command to fail instead of hanging.
func TestIntegrationGitOptionsEnvsCwdTimeout(t *testing.T) {
	e := newGitTestEnv(t)

	// --- Envs ---
	envRepo := "/tmp/it-envs"
	e.initRepo(envRepo, "main")
	_, err := e.sb.Files().Write(e.ctx, envRepo+"/a.txt", []byte("a\n"))
	require.NoError(t, err)
	_, err = e.git.Add(e.ctx, envRepo, nil)
	require.NoError(t, err)

	authorDate := "2020-01-02T03:04:05+00:00"
	// Deliberately set GIT_TERMINAL_PROMPT="1" to verify the SDK default
	// reasserts itself by overwriting to "0".
	_, err = e.git.Commit(e.ctx, envRepo, "feat: dated", &CommitOptions{
		GitOptions: GitOptions{
			Envs: map[string]string{
				"GIT_AUTHOR_DATE":     authorDate,
				"GIT_COMMITTER_DATE":  authorDate,
				"GIT_TERMINAL_PROMPT": "1", // must NOT win; SDK forces 0
			},
		},
	})
	require.NoError(t, err)
	res, err := e.sb.Commands().Run(e.ctx, "git -C "+envRepo+" log -1 --pretty=%aI",
		WithEnvs(map[string]string{"GIT_TERMINAL_PROMPT": "0"}))
	require.NoError(t, err)
	assert.Equal(t, "2020-01-02T03:04:05+00:00", strings.TrimSpace(res.Stdout))

	// --- Cwd ---
	cwdParent := "/tmp/it-cwd"
	_, err = e.sb.Commands().Run(e.ctx, "mkdir -p "+cwdParent,
		WithEnvs(map[string]string{"GIT_TERMINAL_PROMPT": "0"}))
	require.NoError(t, err)
	_, err = e.git.Init(e.ctx, "nested", &InitOptions{
		InitialBranch: "main",
		GitOptions:    GitOptions{Cwd: cwdParent},
	})
	require.NoError(t, err)
	exists, err := e.sb.Files().Exists(e.ctx, cwdParent+"/nested/.git/HEAD")
	require.NoError(t, err)
	assert.True(t, exists, "Init with a relative path under Cwd should create the repo at %s/nested", cwdParent)

	// --- Timeout ---
	// A 1ns timeout should fail the command outright rather than blocking.
	_, err = e.git.Status(e.ctx, envRepo, &GitOptions{Timeout: 1 * time.Nanosecond})
	assert.Error(t, err, "Timeout=1ns should fail the command")
}

// TestIntegrationGitPushPullWithCredentials end-to-end exercises a credentialed
// Clone → Commit → Push → Pull cycle against a real remote:
//  1. Clone the repository into the sandbox (with credentials; stripped by default).
//  2. Create a unique branch (it-push-<timestamp>-<pid>).
//  3. Write a file + commit + Push (SetUpstream=true) to the remote.
//  4. Re-clone a fresh copy + Pull the branch and verify the file is visible.
//  5. t.Cleanup removes the remote branch via push --delete.
//
// Runs only when SUFY_GIT_REPO_URL/USERNAME/PASSWORD point at a writable
// repository. Use a dedicated test repo; do not point at production.
func TestIntegrationGitPushPullWithCredentials(t *testing.T) {
	repoURL, username, password := getGitCredsFromEnv(t)
	e := newGitTestEnv(t)

	branch := fmt.Sprintf("it-push-%d-%d", time.Now().UnixNano(), os.Getpid())
	work := "/tmp/it-rpush-work"
	verify := "/tmp/it-rpush-verify"
	marker := fmt.Sprintf("sandbox integration test %s\n", branch)
	markerFile := branch + ".txt"

	// 1) Clone the working repo (with credentials; stripped by default).
	_, err := e.git.Clone(e.ctx, repoURL, &CloneOptions{
		Path: work, Depth: 1, Username: username, Password: password,
	})
	require.NoError(t, err)

	originURL, err := e.git.RemoteGet(e.ctx, work, "origin", nil)
	require.NoError(t, err)
	assert.NotContains(t, originURL, username+":", "after clone, origin URL should not carry credentials")

	// 2) Configure author so we do not depend on remote defaults.
	_, err = e.git.ConfigureUser(e.ctx, "Sandbox Integration", "sandbox-it@example.com",
		&ConfigOptions{Scope: GitConfigScopeLocal, Path: work})
	require.NoError(t, err)

	// 3) Commit + push on a unique branch.
	_, err = e.git.CreateBranch(e.ctx, work, branch, nil)
	require.NoError(t, err)
	_, err = e.sb.Files().Write(e.ctx, work+"/"+markerFile, []byte(marker))
	require.NoError(t, err)
	_, err = e.git.Add(e.ctx, work, nil)
	require.NoError(t, err)
	_, err = e.git.Commit(e.ctx, work, "test: sandbox integration "+branch, nil)
	require.NoError(t, err)

	// 4) Cleanup: try to delete the remote branch on the way out, success or not.
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		// Use a direct shell push --delete to avoid the SDK credential
		// injection path here.
		authedURL, werr := withCredentials(repoURL, username, password)
		if werr != nil {
			t.Logf("cleanup withCredentials failed: %v", werr)
			return
		}
		cmd := fmt.Sprintf("git -C %s push %s --delete %s",
			shellEscape(work), shellEscape(authedURL), shellEscape(branch))
		res, err := e.sb.Commands().Run(cleanupCtx, cmd,
			WithEnvs(map[string]string{"GIT_TERMINAL_PROMPT": "0"}))
		if err != nil || (res != nil && res.ExitCode != 0) {
			t.Logf("cleanup failed to delete remote branch %s (perhaps already absent): err=%v exit=%v stderr=%q",
				branch, err, res.ExitCode, res.Stderr)
		} else {
			t.Logf("deleted remote test branch %s", branch)
		}
	})

	_, err = e.git.Push(e.ctx, work, &PushOptions{
		Remote: "origin", Branch: branch,
		Username: username, Password: password,
	})
	require.NoError(t, err, "Push failed — verify the token has push access to the target repo")

	// After push, origin URL must still not carry credentials.
	originURL, err = e.git.RemoteGet(e.ctx, work, "origin", nil)
	require.NoError(t, err)
	assert.NotContains(t, originURL, username+":", "after Push, origin URL should not carry credentials")
	assert.NotContains(t, originURL, password)

	// 5) Use a fresh repo to Clone + Pull and verify the pushed commit is
	// actually visible at the remote.
	_, err = e.git.Clone(e.ctx, repoURL, &CloneOptions{
		Path: verify, Branch: branch, Depth: 1,
		Username: username, Password: password,
	})
	require.NoError(t, err, "Clone --branch %s failed — Push may not have landed", branch)

	exists, err := e.sb.Files().Exists(e.ctx, verify+"/"+markerFile)
	require.NoError(t, err)
	assert.True(t, exists, "freshly cloned repo should see pushed file %s", markerFile)
	got, err := e.sb.Files().ReadText(e.ctx, verify+"/"+markerFile)
	require.NoError(t, err)
	assert.Equal(t, marker, got)

	// 6) Append another commit on work and push it; Pull on verify should
	// pick up the new commit.
	follow := "follow-up\n"
	followFile := branch + "-2.txt"
	_, err = e.sb.Files().Write(e.ctx, work+"/"+followFile, []byte(follow))
	require.NoError(t, err)
	_, err = e.git.Add(e.ctx, work, nil)
	require.NoError(t, err)
	_, err = e.git.Commit(e.ctx, work, "test: follow-up "+branch, nil)
	require.NoError(t, err)
	_, err = e.git.Push(e.ctx, work, &PushOptions{
		Remote: "origin", Branch: branch,
		Username: username, Password: password,
	})
	require.NoError(t, err)

	_, err = e.git.Pull(e.ctx, verify, &PullOptions{
		Remote: "origin", Branch: branch,
		Username: username, Password: password,
	})
	require.NoError(t, err)
	exists, err = e.sb.Files().Exists(e.ctx, verify+"/"+followFile)
	require.NoError(t, err)
	assert.True(t, exists, "after Pull, follow-up file %s should be visible", followFile)
}

// stagedNames returns the set of staged file names in a status.
func stagedNames(s *GitStatus) []string {
	var out []string
	for _, f := range s.FileStatus {
		if f.Staged {
			out = append(out, f.Name)
		}
	}
	return out
}

// getGitCredsFromEnv reads git credentials from environment variables; if any
// is missing it skips the test.
func getGitCredsFromEnv(t *testing.T) (string, string, string) {
	t.Helper()
	repoURL := strings.TrimSpace(os.Getenv("SUFY_GIT_REPO_URL"))
	username := strings.TrimSpace(os.Getenv("SUFY_GIT_USERNAME"))
	password := strings.TrimSpace(os.Getenv("SUFY_GIT_PASSWORD"))
	if repoURL == "" || username == "" || password == "" {
		t.Skip("SUFY_GIT_REPO_URL / SUFY_GIT_USERNAME / SUFY_GIT_PASSWORD not set; skipping credentialed Clone test")
	}
	return repoURL, username, password
}
