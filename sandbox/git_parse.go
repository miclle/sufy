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
	"net/url"
	"path"
	"strconv"
	"strings"
)

// deriveRepoDirFromURL derives the default repository directory name from a
// git URL (with the .git suffix stripped).
func deriveRepoDirFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	var name string
	if err == nil && u.Path != "" {
		name = path.Base(u.Path)
	} else {
		// Handle SCP-style addresses such as git@host:owner/repo.git.
		if idx := strings.LastIndex(rawURL, ":"); idx >= 0 {
			name = path.Base(rawURL[idx+1:])
		} else {
			name = path.Base(rawURL)
		}
	}
	name = strings.TrimSuffix(name, ".git")
	if name == "." || name == "/" || name == "" {
		return ""
	}
	return name
}

// parseGitStatus parses the output of `git status --porcelain=1 -b`.
func parseGitStatus(out string) *GitStatus {
	status := &GitStatus{}
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "## ") {
			parseBranchLine(line[3:], status)
			continue
		}
		entry, ok := parseFileStatusLine(line)
		if ok {
			status.FileStatus = append(status.FileStatus, entry)
		}
	}
	return status
}

// parseBranchLine parses the branch info line from porcelain output.
//
// Handles the variations git emits in different states:
//   - "main...origin/main [ahead 1, behind 2]"  normal tracking branch
//   - "HEAD (no branch)"                        legacy detached HEAD (e.g. during rebase)
//   - "HEAD (detached at <ref>)"                explicit detached HEAD
//   - "No commits yet on main"                  unborn branch (no initial commit)
//   - "Initial commit on main"                  unborn branch on older git versions
//
// We cannot simply split on the first space; otherwise "No commits yet on main"
// would treat "No" as the branch name.
func parseBranchLine(line string, status *GitStatus) {
	if line == "" {
		return
	}

	// First, strip the trailing "[ahead N, behind M]".
	branchPart := line
	rest := ""
	if i := strings.Index(line, " ["); i >= 0 && strings.HasSuffix(line, "]") {
		branchPart = line[:i]
		rest = line[i+2 : len(line)-1]
	}

	// Handle both flavors of detached HEAD.
	if strings.HasPrefix(branchPart, "HEAD (no branch)") ||
		strings.HasPrefix(branchPart, "HEAD (detached") ||
		strings.HasPrefix(branchPart, "(no branch)") {
		status.Detached = true
		parseAheadBehind(rest, status)
		return
	}

	// Handle unborn branches: before the initial commit, git prints
	// "No commits yet on <branch>" or, on older versions,
	// "Initial commit on <branch>". The branch is recorded but no upstream is
	// available.
	switch {
	case strings.HasPrefix(branchPart, "No commits yet on "):
		status.CurrentBranch = strings.TrimPrefix(branchPart, "No commits yet on ")
	case strings.HasPrefix(branchPart, "Initial commit on "):
		status.CurrentBranch = strings.TrimPrefix(branchPart, "Initial commit on ")
	default:
		if before, after, ok := strings.Cut(branchPart, "..."); ok {
			status.CurrentBranch = before
			status.Upstream = after
		} else {
			status.CurrentBranch = branchPart
		}
	}

	parseAheadBehind(rest, status)
}

// parseAheadBehind parses the ahead/behind segment (without the surrounding
// brackets).
func parseAheadBehind(rest string, status *GitStatus) {
	if rest == "" {
		return
	}
	for _, seg := range strings.Split(rest, ",") {
		seg = strings.TrimSpace(seg)
		switch {
		case strings.HasPrefix(seg, "ahead "):
			if n, err := strconv.Atoi(strings.TrimPrefix(seg, "ahead ")); err == nil {
				status.Ahead = n
			}
		case strings.HasPrefix(seg, "behind "):
			if n, err := strconv.Atoi(strings.TrimPrefix(seg, "behind ")); err == nil {
				status.Behind = n
			}
		}
	}
}

// parseFileStatusLine parses a single file status line from porcelain output.
//
// When the file name contains spaces, double quotes, or non-printable bytes,
// git emits a C-style quoted path by default (e.g. `?? "with space.txt"`,
// `R  "old name" -> "new name"`). The quoted body must be unescaped so the
// caller receives the actual relative path.
func parseFileStatusLine(line string) (GitFileStatus, bool) {
	if len(line) < 3 {
		return GitFileStatus{}, false
	}
	x := string(line[0])
	y := string(line[1])
	rest := line[3:]

	entry := GitFileStatus{
		IndexStatus:       x,
		WorkingTreeStatus: y,
	}

	// Rename/copy: the path is "old -> new" with either side potentially
	// quoted.
	if x == "R" || x == "C" || y == "R" || y == "C" {
		if before, after, ok := splitRenamePath(rest); ok {
			entry.RenamedFrom = unquoteCPath(before)
			entry.Name = unquoteCPath(after)
		} else {
			entry.Name = unquoteCPath(rest)
		}
	} else {
		entry.Name = unquoteCPath(rest)
	}

	entry.Status = normalizeFileStatus(x, y)
	entry.Staged = isStaged(x, entry.Status)
	return entry, true
}

// splitRenamePath splits a "old -> new" rename path, correctly handling cases
// where either side is wrapped in quotes. The simple form splits on " -> ";
// if old starts with `"` we locate the matching closing quote first before
// looking for " -> ".
func splitRenamePath(s string) (string, string, bool) {
	if strings.HasPrefix(s, `"`) {
		// Find the first unescaped closing quote.
		i := 1
		for i < len(s) {
			if s[i] == '\\' && i+1 < len(s) {
				i += 2
				continue
			}
			if s[i] == '"' {
				break
			}
			i++
		}
		if i >= len(s) {
			return "", "", false
		}
		head := s[:i+1]
		rest := s[i+1:]
		const sep = " -> "
		if strings.HasPrefix(rest, sep) {
			return head, rest[len(sep):], true
		}
		return "", "", false
	}
	before, after, ok := strings.Cut(s, " -> ")
	return before, after, ok
}

// unquoteCPath unescapes a git porcelain C-style quoted path.
//
// Inputs that do not start with a double quote are returned as is. Otherwise
// it follows git's quote_c_style rules: supports \\, \", \a \b \f \n \r \t \v
// and \<3-octal> arbitrary bytes. On parse failure the original string is
// returned to avoid breaking the caller.
func unquoteCPath(s string) string {
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return s
	}
	body := s[1 : len(s)-1]
	out := make([]byte, 0, len(body))
	for i := 0; i < len(body); i++ {
		c := body[i]
		if c != '\\' {
			out = append(out, c)
			continue
		}
		if i+1 >= len(body) {
			return s
		}
		next := body[i+1]
		switch next {
		case '\\', '"', '\'':
			out = append(out, next)
			i++
		case 'a':
			out = append(out, '\a')
			i++
		case 'b':
			out = append(out, '\b')
			i++
		case 'f':
			out = append(out, '\f')
			i++
		case 'n':
			out = append(out, '\n')
			i++
		case 'r':
			out = append(out, '\r')
			i++
		case 't':
			out = append(out, '\t')
			i++
		case 'v':
			out = append(out, '\v')
			i++
		default:
			// Three octal digits: \NNN. Git uses this for non-ASCII bytes
			// (e.g. multi-byte UTF-8 sequences). Three octal digits can
			// represent values up to 0o777=511, which overflows a byte;
			// strconv.ParseUint then errors. In that case treat it as an
			// invalid escape: keep the literal `\` and advance to the next
			// character so the whole string is not lost.
			if i+3 < len(body) && isOctal(body[i+1]) && isOctal(body[i+2]) && isOctal(body[i+3]) {
				if v, perr := strconv.ParseUint(string(body[i+1:i+4]), 8, 8); perr == nil {
					out = append(out, byte(v))
					i += 3
					continue
				}
			}
			out = append(out, '\\')
		}
	}
	return string(out)
}

// isOctal reports whether c is an octal digit byte.
func isOctal(c byte) bool { return c >= '0' && c <= '7' }

// normalizeFileStatus normalizes the X/Y characters into a human-readable
// status string.
func normalizeFileStatus(x, y string) string {
	if x == "?" && y == "?" {
		return "untracked"
	}
	if x == "!" && y == "!" {
		return "ignored"
	}
	if isConflict(x, y) {
		return "conflict"
	}
	// Prefer the index position, then the working-tree position.
	if s := mapStatusChar(x); s != "" && s != "unmodified" {
		return s
	}
	if s := mapStatusChar(y); s != "" && s != "unmodified" {
		return s
	}
	return "unmodified"
}

// mapStatusChar maps a single status character to a readable string.
func mapStatusChar(c string) string {
	switch c {
	case " ":
		return "unmodified"
	case "M":
		return "modified"
	case "A":
		return "added"
	case "D":
		return "deleted"
	case "R":
		return "renamed"
	case "C":
		return "copied"
	case "T":
		return "type-changed"
	case "U":
		return "conflict"
	}
	return ""
}

// isConflict reports whether the X/Y pair represents a git merge conflict.
// Reference: the "Unmerged" table in `git status` docs
// (DD/AU/UD/UA/DU/AA/UU).
func isConflict(x, y string) bool {
	switch x + y {
	case "DD", "AU", "UD", "UA", "DU", "AA", "UU":
		return true
	}
	return x == "U" || y == "U"
}

// isStaged reports whether the entry is in the staged state.
func isStaged(x, status string) bool {
	if status == "untracked" || status == "ignored" || status == "conflict" {
		return false
	}
	if x == " " || x == "?" || x == "!" {
		return false
	}
	return true
}

// parseGitBranches parses the output of
// `git branch --format=%(refname:short)\t%(HEAD)`.
//
// When HEAD is detached, git emits a "(HEAD detached at <sha>)" /
// "(HEAD detached from <ref>)" pseudo-branch marked as the current HEAD. We
// skip it so CurrentBranch stays empty on detached HEAD.
func parseGitBranches(out string) *GitBranches {
	result := &GitBranches{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		name := strings.TrimSpace(parts[0])
		if name == "" {
			continue
		}
		isCurrent := len(parts) == 2 && strings.TrimSpace(parts[1]) == "*"
		if strings.HasPrefix(name, "(HEAD detached") || name == "(no branch)" {
			continue
		}
		result.Branches = append(result.Branches, name)
		if isCurrent {
			result.CurrentBranch = name
		}
	}
	return result
}
