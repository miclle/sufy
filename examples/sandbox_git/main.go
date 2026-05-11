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

// sandbox_git demonstrates the Sandbox.Git() high-level interface end-to-end.
//
// The walkthrough runs entirely inside the sandbox without relying on any
// external repo: a bare repository is initialized inside the sandbox to act
// as the "remote", chaining Init / Add / Commit / Push / Pull. If
// SUFY_GIT_REPO_URL (HTTPS), SUFY_GIT_USERNAME, SUFY_GIT_PASSWORD are set,
// a credentialed Clone is also demonstrated.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/sufy-dev/sufy/examples/internal/exampleutil"
	"github.com/sufy-dev/sufy/sandbox"
)

func main() {
	c := exampleutil.MustNewClient()

	ctx, cancel := context.WithTimeout(context.Background(), 240*time.Second)
	defer cancel()

	// 1. Pick a usable template and create the sandbox.
	templates, err := c.ListTemplates(ctx, nil)
	if err != nil {
		log.Fatalf("ListTemplates failed: %v", err)
	}
	var templateID string
	for _, tmpl := range templates {
		if tmpl.BuildStatus == sandbox.BuildStatusReady || tmpl.BuildStatus == sandbox.BuildStatusUploaded {
			templateID = tmpl.TemplateID
			break
		}
	}
	if templateID == "" {
		log.Fatal("no ready template available")
	}
	fmt.Printf("using template: %s\n", templateID)

	timeout := int32(240)
	sb, _, err := c.CreateAndWait(ctx, sandbox.CreateParams{
		TemplateID: templateID,
		Timeout:    &timeout,
	}, sandbox.WithPollInterval(2*time.Second))
	if err != nil {
		log.Fatalf("CreateAndWait failed: %v", err)
	}
	fmt.Printf("sandbox ready: %s\n", sb.ID())

	defer func() {
		killCtx, killCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer killCancel()
		if err := sb.Kill(killCtx); err != nil {
			log.Printf("Kill failed: %v", err)
		} else {
			fmt.Printf("sandbox %s killed\n", sb.ID())
		}
	}()

	git := sb.Git()
	repoPath := "/tmp/demo-repo"
	bareRepoPath := "/tmp/demo-remote.git"
	consumerPath := "/tmp/demo-consumer"

	// 2. Initialize a working repo + a bare repo to act as the remote.
	fmt.Println("\n--- Init ---")
	if _, err := git.Init(ctx, repoPath, &sandbox.InitOptions{InitialBranch: "main"}); err != nil {
		log.Fatalf("Init failed: %v", err)
	}
	fmt.Printf("initialized repo: %s (initial-branch=main)\n", repoPath)

	if _, err := git.Init(ctx, bareRepoPath, &sandbox.InitOptions{Bare: true, InitialBranch: "main"}); err != nil {
		log.Fatalf("Init(bare) failed: %v", err)
	}
	fmt.Printf("initialized bare repo: %s\n", bareRepoPath)

	// 3. Configure commit user (local scope).
	fmt.Println("\n--- ConfigureUser / SetConfig / GetConfig ---")
	if _, err := git.ConfigureUser(ctx, "Sandbox Demo", "demo@example.com", &sandbox.ConfigOptions{
		Scope: sandbox.GitConfigScopeLocal,
		Path:  repoPath,
	}); err != nil {
		log.Fatalf("ConfigureUser failed: %v", err)
	}
	if _, err := git.SetConfig(ctx, "core.autocrlf", "input", &sandbox.ConfigOptions{
		Scope: sandbox.GitConfigScopeLocal,
		Path:  repoPath,
	}); err != nil {
		log.Fatalf("SetConfig failed: %v", err)
	}
	for _, key := range []string{"user.name", "user.email", "core.autocrlf"} {
		val, err := git.GetConfig(ctx, key, &sandbox.ConfigOptions{
			Scope: sandbox.GitConfigScopeLocal,
			Path:  repoPath,
		})
		if err != nil {
			log.Fatalf("GetConfig(%s) failed: %v", key, err)
		}
		fmt.Printf("  %s = %q\n", key, val)
	}
	// Missing keys return an empty string.
	missing, err := git.GetConfig(ctx, "user.notexist", &sandbox.ConfigOptions{
		Scope: sandbox.GitConfigScopeLocal,
		Path:  repoPath,
	})
	if err != nil {
		log.Fatalf("GetConfig(missing) failed: %v", err)
	}
	fmt.Printf("  user.notexist = %q (not configured)\n", missing)

	// 4. Write a file, stage it, commit it.
	fmt.Println("\n--- Add / Commit ---")
	if _, err := sb.Files().Write(ctx, repoPath+"/README.md", []byte("# demo\n")); err != nil {
		log.Fatalf("write README failed: %v", err)
	}
	if _, err := git.Add(ctx, repoPath, nil); err != nil { // nil → defaults to -A
		log.Fatalf("Add failed: %v", err)
	}
	if _, err := git.Commit(ctx, repoPath, "feat: initial commit", &sandbox.CommitOptions{
		AuthorName:  "Sandbox Demo",
		AuthorEmail: "demo@example.com",
	}); err != nil {
		log.Fatalf("Commit failed: %v", err)
	}
	fmt.Println("initial commit created")

	// 5. Inspect status.
	fmt.Println("\n--- Status ---")
	st, err := git.Status(ctx, repoPath, nil)
	if err != nil {
		log.Fatalf("Status failed: %v", err)
	}
	fmt.Printf("CurrentBranch=%s Detached=%v Clean=%v Total=%d Staged=%d Unstaged=%d\n",
		st.CurrentBranch, st.Detached, st.IsClean(), st.TotalCount(), st.StagedCount(), st.UnstagedCount())

	// 6. Branch management: CreateBranch -> edit -> Commit -> Branches -> Checkout -> Delete.
	fmt.Println("\n--- Branches ---")
	if _, err := git.CreateBranch(ctx, repoPath, "feature/x", nil); err != nil {
		log.Fatalf("CreateBranch failed: %v", err)
	}
	if _, err := sb.Files().Write(ctx, repoPath+"/feature.txt", []byte("hello feature\n")); err != nil {
		log.Fatalf("write feature.txt failed: %v", err)
	}
	if _, err := git.Add(ctx, repoPath, &sandbox.AddOptions{Files: []string{"feature.txt"}}); err != nil {
		log.Fatalf("Add(files) failed: %v", err)
	}
	if _, err := git.Commit(ctx, repoPath, "feat: add feature.txt", nil); err != nil {
		log.Fatalf("Commit failed: %v", err)
	}

	branches, err := git.Branches(ctx, repoPath, nil)
	if err != nil {
		log.Fatalf("Branches failed: %v", err)
	}
	fmt.Printf("branches: %v, current: %s\n", branches.Branches, branches.CurrentBranch)

	if _, err := git.CheckoutBranch(ctx, repoPath, "main", nil); err != nil {
		log.Fatalf("CheckoutBranch failed: %v", err)
	}
	if _, err := git.DeleteBranch(ctx, repoPath, "feature/x", &sandbox.DeleteBranchOptions{Force: true}); err != nil {
		log.Fatalf("DeleteBranch failed: %v", err)
	}
	fmt.Println("checked out main and force-deleted feature/x")

	// 7. Reset / Restore.
	fmt.Println("\n--- Reset / Restore ---")

	// 7a. Reset paths-only: unstage paths without changing HEAD.
	if _, err := sb.Files().Write(ctx, repoPath+"/dirty.txt", []byte("dirty\n")); err != nil {
		log.Fatalf("write dirty.txt failed: %v", err)
	}
	if _, err := git.Add(ctx, repoPath, nil); err != nil {
		log.Fatalf("Add failed: %v", err)
	}
	st, err = git.Status(ctx, repoPath, nil)
	if err != nil {
		log.Fatalf("Status failed: %v", err)
	}
	fmt.Printf("after Add: staged=%d unstaged=%d\n", st.StagedCount(), st.UnstagedCount())
	if _, err := git.Reset(ctx, repoPath, &sandbox.ResetOptions{Paths: []string{"dirty.txt"}}); err != nil {
		log.Fatalf("Reset(paths) failed: %v", err)
	}
	st, err = git.Status(ctx, repoPath, nil)
	if err != nil {
		log.Fatalf("Status failed: %v", err)
	}
	fmt.Printf("after Reset(paths): staged=%d unstaged=%d\n", st.StagedCount(), st.UnstagedCount())

	// 7b. Reset --hard: discard worktree changes and move HEAD to the target.
	if _, err := sb.Files().Write(ctx, repoPath+"/README.md", []byte("# demo (modified)\n")); err != nil {
		log.Fatalf("modify README failed: %v", err)
	}
	if _, err := git.Reset(ctx, repoPath, &sandbox.ResetOptions{
		Mode:   sandbox.GitResetModeHard,
		Target: "HEAD",
	}); err != nil {
		log.Fatalf("Reset(--hard HEAD) failed: %v", err)
	}
	readme, err := sb.Files().ReadText(ctx, repoPath+"/README.md")
	if err != nil {
		log.Fatalf("read README failed: %v", err)
	}
	fmt.Printf("after Reset --hard, README.md = %q\n", readme)

	// 7c. Restore --staged: unstage paths while keeping worktree changes.
	if _, err := sb.Files().Write(ctx, repoPath+"/README.md", []byte("# demo (staged change)\n")); err != nil {
		log.Fatalf("modify README failed: %v", err)
	}
	if _, err := git.Add(ctx, repoPath, nil); err != nil {
		log.Fatalf("Add failed: %v", err)
	}
	stagedPtr := true
	if _, err := git.Restore(ctx, repoPath, &sandbox.RestoreOptions{
		Paths:  []string{"README.md"},
		Staged: &stagedPtr,
	}); err != nil {
		log.Fatalf("Restore(--staged) failed: %v", err)
	}
	st, err = git.Status(ctx, repoPath, nil)
	if err != nil {
		log.Fatalf("Status failed: %v", err)
	}
	fmt.Printf("after Restore --staged: staged=%d unstaged=%d\n", st.StagedCount(), st.UnstagedCount())

	// 7d. Restore --worktree --source=HEAD: roll the worktree back to HEAD.
	if _, err := git.Restore(ctx, repoPath, &sandbox.RestoreOptions{
		Paths:  []string{"README.md"},
		Source: "HEAD",
	}); err != nil {
		log.Fatalf("Restore(--source HEAD) failed: %v", err)
	}
	readme, err = sb.Files().ReadText(ctx, repoPath+"/README.md")
	if err != nil {
		log.Fatalf("read README failed: %v", err)
	}
	fmt.Printf("after Restore --source HEAD, README.md = %q\n", readme)

	// 8. Remote management, including Overwrite.
	fmt.Println("\n--- Remote ---")
	if _, err := git.RemoteAdd(ctx, repoPath, "origin", "https://example.com/placeholder.git", nil); err != nil {
		log.Fatalf("RemoteAdd failed: %v", err)
	}
	if _, err := git.RemoteAdd(ctx, repoPath, "origin", bareRepoPath, &sandbox.RemoteAddOptions{
		Overwrite: true,
	}); err != nil {
		log.Fatalf("RemoteAdd(overwrite) failed: %v", err)
	}
	url, err := git.RemoteGet(ctx, repoPath, "origin", nil)
	if err != nil {
		log.Fatalf("RemoteGet failed: %v", err)
	}
	fmt.Printf("origin URL = %s\n", url)

	missingRemote, err := git.RemoteGet(ctx, repoPath, "nonexistent", nil)
	if err != nil {
		log.Fatalf("RemoteGet(nonexistent) failed: %v", err)
	}
	fmt.Printf("nonexistent URL = %q (not configured)\n", missingRemote)

	// 9. Push (no credentials; SetUpstream defaults to true).
	fmt.Println("\n--- Push ---")
	if _, err := git.Push(ctx, repoPath, &sandbox.PushOptions{
		Remote: "origin",
		Branch: "main",
	}); err != nil {
		log.Fatalf("Push failed: %v", err)
	}
	fmt.Printf("pushed to %s (main)\n", bareRepoPath)

	// 10. Pull: clone a consumer repo from the same bare remote, then pull.
	fmt.Println("\n--- Pull (via local clone) ---")
	if _, err := git.Clone(ctx, bareRepoPath, &sandbox.CloneOptions{Path: consumerPath}); err != nil {
		log.Fatalf("local Clone failed: %v", err)
	}
	fmt.Printf("cloned to %s\n", consumerPath)

	// Make another commit in the source repo and push it.
	if _, err := sb.Files().Write(ctx, repoPath+"/CHANGELOG.md", []byte("# v1\n")); err != nil {
		log.Fatalf("write CHANGELOG failed: %v", err)
	}
	if _, err := git.Add(ctx, repoPath, nil); err != nil {
		log.Fatalf("Add failed: %v", err)
	}
	if _, err := git.Commit(ctx, repoPath, "docs: add CHANGELOG", nil); err != nil {
		log.Fatalf("Commit failed: %v", err)
	}
	if _, err := git.Push(ctx, repoPath, &sandbox.PushOptions{
		Remote: "origin",
		Branch: "main",
	}); err != nil {
		log.Fatalf("Push(2) failed: %v", err)
	}

	// Pull the latest commit in the consumer.
	if _, err := git.Pull(ctx, consumerPath, &sandbox.PullOptions{
		Remote: "origin",
		Branch: "main",
	}); err != nil {
		log.Fatalf("Pull failed: %v", err)
	}
	exists, err := sb.Files().Exists(ctx, consumerPath+"/CHANGELOG.md")
	if err != nil {
		log.Fatalf("Exists failed: %v", err)
	}
	fmt.Printf("after Pull, consumer/CHANGELOG.md exists = %v\n", exists)

	// 11. DangerouslyAuthenticate persists credentials in the sandbox's
	// global credential store; it does not affect the host environment.
	fmt.Println("\n--- DangerouslyAuthenticate (sandbox-local credential store) ---")
	if _, err := git.DangerouslyAuthenticate(ctx, &sandbox.AuthenticateOptions{
		Username: "demo-user",
		Password: "demo-token",
		Host:     "example.com",
		Protocol: "https",
	}); err != nil {
		log.Fatalf("DangerouslyAuthenticate failed: %v", err)
	}
	fmt.Println("approved example.com credentials via git credential approve")

	// 12. Optional: credentialed Clone (only when SUFY_GIT_REPO_URL et al. are set).
	repoURL := os.Getenv("SUFY_GIT_REPO_URL")
	username := os.Getenv("SUFY_GIT_USERNAME")
	password := os.Getenv("SUFY_GIT_PASSWORD")
	if repoURL != "" && username != "" && password != "" {
		fmt.Println("\n--- Clone (HTTPS + token; credentials stripped from origin URL after clone) ---")
		clonePath := "/tmp/cloned-repo"
		if _, err := git.Clone(ctx, repoURL, &sandbox.CloneOptions{
			Path:     clonePath,
			Depth:    1,
			Username: username,
			Password: password,
		}); err != nil {
			log.Fatalf("Clone failed: %v", err)
		}
		clonedURL, err := git.RemoteGet(ctx, clonePath, "origin", nil)
		if err != nil {
			log.Fatalf("RemoteGet(cloned) failed: %v", err)
		}
		fmt.Printf("clone done, origin URL = %s (no credentials)\n", clonedURL)
	} else {
		fmt.Println("\n--- remote Clone skipped (SUFY_GIT_REPO_URL / SUFY_GIT_USERNAME / SUFY_GIT_PASSWORD not set) ---")
	}
}
