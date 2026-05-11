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

// GitFileStatus describes a single file entry from `git status`.
type GitFileStatus struct {
	// Name is the path relative to the repository root.
	Name string
	// Status is the normalized status string, e.g. "modified", "added",
	// "deleted", "untracked", "conflict".
	Status string
	// IndexStatus is the index-position character from porcelain output.
	IndexStatus string
	// WorkingTreeStatus is the working-tree-position character from porcelain output.
	WorkingTreeStatus string
	// Staged reports whether the file has been staged.
	Staged bool
	// RenamedFrom records the original path when the file has been renamed.
	RenamedFrom string
}

// GitStatus describes the overall repository status.
type GitStatus struct {
	// CurrentBranch is the current branch name. Empty when HEAD is detached.
	CurrentBranch string
	// Upstream is the upstream branch name. Empty when no upstream is configured.
	Upstream string
	// Ahead is the number of commits the current branch is ahead of upstream.
	Ahead int
	// Behind is the number of commits the current branch is behind upstream.
	Behind int
	// Detached reports whether HEAD is in detached state.
	Detached bool
	// FileStatus carries all changed-file entries.
	FileStatus []GitFileStatus
}

// IsClean returns true when the repository has no file changes.
func (s *GitStatus) IsClean() bool {
	return len(s.FileStatus) == 0
}

// HasChanges returns true when the repository has any file changes.
func (s *GitStatus) HasChanges() bool {
	return len(s.FileStatus) > 0
}

// HasStaged returns true when at least one file has staged changes.
func (s *GitStatus) HasStaged() bool {
	for i := range s.FileStatus {
		if s.FileStatus[i].Staged {
			return true
		}
	}
	return false
}

// HasUntracked returns true when at least one untracked file exists.
func (s *GitStatus) HasUntracked() bool {
	for i := range s.FileStatus {
		if s.FileStatus[i].Status == "untracked" {
			return true
		}
	}
	return false
}

// HasConflicts returns true when at least one conflicted file exists.
func (s *GitStatus) HasConflicts() bool {
	for i := range s.FileStatus {
		if s.FileStatus[i].Status == "conflict" {
			return true
		}
	}
	return false
}

// TotalCount returns the total number of changed files.
func (s *GitStatus) TotalCount() int {
	return len(s.FileStatus)
}

// StagedCount returns the number of staged files.
func (s *GitStatus) StagedCount() int {
	n := 0
	for i := range s.FileStatus {
		if s.FileStatus[i].Staged {
			n++
		}
	}
	return n
}

// UnstagedCount returns the number of files with unstaged changes. A file with
// both staged and unstaged changes (e.g. "MM file") is also counted here.
func (s *GitStatus) UnstagedCount() int {
	n := 0
	for i := range s.FileStatus {
		f := &s.FileStatus[i]
		if f.Status == "untracked" || f.Status == "ignored" {
			continue
		}
		if f.WorkingTreeStatus != " " {
			n++
		}
	}
	return n
}

// UntrackedCount returns the number of untracked files.
func (s *GitStatus) UntrackedCount() int {
	n := 0
	for i := range s.FileStatus {
		if s.FileStatus[i].Status == "untracked" {
			n++
		}
	}
	return n
}

// ConflictCount returns the number of conflicted files.
func (s *GitStatus) ConflictCount() int {
	n := 0
	for i := range s.FileStatus {
		if s.FileStatus[i].Status == "conflict" {
			n++
		}
	}
	return n
}

// GitBranches describes the branch listing of a repository.
type GitBranches struct {
	// Branches lists all local branch names.
	Branches []string
	// CurrentBranch is the current branch name. Empty when HEAD is detached.
	CurrentBranch string
}

// GitConfigScope identifies the scope used by `git config` commands.
type GitConfigScope string

const (
	// GitConfigScopeGlobal targets user-level config (~/.gitconfig).
	GitConfigScopeGlobal GitConfigScope = "global"
	// GitConfigScopeLocal targets repository-level config
	// (<repo>/.git/config). Requires a repository path.
	GitConfigScopeLocal GitConfigScope = "local"
	// GitConfigScopeSystem targets system-level config (/etc/gitconfig).
	GitConfigScopeSystem GitConfigScope = "system"
)
