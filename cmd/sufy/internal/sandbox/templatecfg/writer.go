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

package templatecfg

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// templateIDLineRegex matches an uncommented template_id assignment line
// (leading whitespace allowed).
var templateIDLineRegex = regexp.MustCompile(`^\s*template_id\s*=`)

// WriteTemplateID writes templateID into the TOML file at path.
//
// If an uncommented template_id line already exists, its value is
// replaced; otherwise a new line is inserted at the top of the file.
// File permissions, comments, field order, indentation, line endings,
// and surrounding whitespace are preserved.
func WriteTemplateID(path, templateID string) error {
	stat, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat %s: %w", path, err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	// Detect the original line-ending style so we preserve CRLF / LF on
	// writeback.
	lineEnd := "\n"
	if bytes.Contains(data, []byte("\r\n")) {
		lineEnd = "\r\n"
	}

	var out bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(data))
	replaced := false
	inRootTable := true
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if !isCommentLine(line) && strings.HasPrefix(trimmed, "[") {
			inRootTable = false
		}
		if inRootTable && !replaced && templateIDLineRegex.MatchString(line) && !isCommentLine(line) {
			// Preserve the original line's leading indent.
			idx := strings.Index(line, "template_id")
			indent := ""
			if idx > 0 {
				indent = line[:idx]
			}
			fmt.Fprintf(&out, "%stemplate_id = %q%s", indent, templateID, lineEnd)
			replaced = true
			continue
		}
		out.WriteString(line)
		out.WriteString(lineEnd)
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan %s: %w", path, err)
	}

	if !replaced {
		// No assignment line found: insert one at the top of the file.
		var prefix bytes.Buffer
		fmt.Fprintf(&prefix, "template_id = %q%s", templateID, lineEnd)
		prefix.Write(out.Bytes())
		out = prefix
	}

	// Keep the trailing newline consistent with the original file.
	final := out.Bytes()
	if len(data) > 0 && !bytes.HasSuffix(data, []byte(lineEnd)) && bytes.HasSuffix(final, []byte(lineEnd)) {
		final = final[:len(final)-len(lineEnd)]
	}

	if err := os.WriteFile(path, final, stat.Mode().Perm()); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// isCommentLine reports whether a line begins with `#` (after optional
// leading whitespace).
func isCommentLine(line string) bool {
	return strings.HasPrefix(strings.TrimLeft(line, " \t"), "#")
}
