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

// Package dockerfile provides a lightweight Dockerfile parser and converter.
//
// It parses Dockerfile content into structured instructions, supporting escape
// directives, line continuations, heredoc syntax, and quoted value handling.
// The parser follows buildkit's lexical processing rules without depending on
// buildkit itself.
package dockerfile

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Instruction represents a parsed Dockerfile instruction.
type Instruction struct {
	// Name is the uppercase instruction name (e.g. "RUN", "COPY", "ENV").
	Name string

	// Args is the raw argument string after the instruction name.
	Args string

	// Flags holds --flag=value pairs extracted from the instruction (e.g. --chown=user:group).
	Flags map[string]string

	// Heredoc contains the heredoc body when the instruction uses heredoc syntax.
	Heredoc string

	// Line is the 1-based line number in the original Dockerfile.
	Line int
}

// ParseResult holds the complete output of a Dockerfile parse.
type ParseResult struct {
	// Instructions is the ordered list of parsed instructions.
	Instructions []Instruction

	// Warnings contains non-fatal parsing issues (e.g. unknown instructions, multi-stage builds).
	Warnings []string

	// EscapeToken is the escape character used (default '\\', can be overridden to '`').
	EscapeToken rune
}

// defaultEscapeToken is the default line continuation character.
const defaultEscapeToken = '\\'

// reHeredoc matches heredoc markers like <<EOF, 0<<-EOF.
var reHeredoc = regexp.MustCompile(`^(\d*)<<(-?)\s*['"]?([a-zA-Z_]\w*)['"]?$`)

// Parse parses Dockerfile content into a ParseResult.
func Parse(content string) (*ParseResult, error) {
	content = stripBOM(content)
	rawLines := strings.Split(content, "\n")

	escapeToken, rawLines := detectEscapeDirective(rawLines)
	lines := joinContinuationLines(rawLines, escapeToken)

	result := &ParseResult{EscapeToken: escapeToken}

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		name, rest := splitInstruction(line)
		name = strings.ToUpper(name)

		inst := Instruction{
			Name: name,
			Args: rest,
			Line: i + 1,
		}

		// Extract flags from instructions that support them.
		switch name {
		case "COPY", "ADD", "FROM":
			inst.Flags, inst.Args = extractFlags(rest)
		}

		// Handle heredoc for instructions that support it.
		switch name {
		case "RUN", "COPY", "ADD":
			body, advance, err := maybeParseHeredoc(rest, lines[i+1:])
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", i+1, err)
			}
			if advance > 0 {
				inst.Heredoc = body
				if name == "RUN" {
					inst.Args = body
				}
				i += advance
			}
		}

		// Generate warnings for specific instructions.
		switch name {
		case "ONBUILD":
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("line %d: ONBUILD instruction is not supported and will be ignored", i+1))
		case "EXPOSE", "VOLUME", "LABEL", "STOPSIGNAL", "HEALTHCHECK", "SHELL",
			"FROM", "RUN", "COPY", "ADD", "WORKDIR", "USER", "ENV", "ARG",
			"CMD", "ENTRYPOINT", "MAINTAINER":
			// Known instructions, no warning needed.
		default:
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("line %d: unknown instruction %q will be ignored", i+1, name))
		}

		result.Instructions = append(result.Instructions, inst)
	}

	return result, nil
}

// detectEscapeDirective scans the top of the Dockerfile for an `# escape=X` directive.
// Only backslash and backtick are valid escape characters.
// The directive must appear before any instruction or non-directive comment.
func detectEscapeDirective(lines []string) (rune, []string) {
	escapeToken := defaultEscapeToken

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || !strings.HasPrefix(trimmed, "#") {
			return escapeToken, lines
		}

		comment := strings.TrimSpace(strings.TrimPrefix(trimmed, "#"))
		if key, value, ok := strings.Cut(comment, "="); ok {
			key = strings.TrimSpace(strings.ToLower(key))
			value = strings.TrimSpace(value)
			if key == "escape" {
				if value == "\\" {
					escapeToken = '\\'
				} else if value == "`" {
					escapeToken = '`'
				}
				return escapeToken, append(lines[:i:i], lines[i+1:]...)
			}
			if key == "syntax" || key == "check" {
				continue
			}
		}

		// Non-directive comment, stop scanning.
		return escapeToken, lines
	}

	return escapeToken, lines
}

// joinContinuationLines merges lines ending with the escape character.
// Comments within a continuation block are skipped (matching buildkit behavior).
func joinContinuationLines(lines []string, escapeToken rune) []string {
	var result []string
	var current strings.Builder
	escStr := string(escapeToken)

	for _, line := range lines {
		trimmed := strings.TrimRight(line, " \t\r")

		if current.Len() > 0 {
			if t := strings.TrimSpace(trimmed); t == "" || strings.HasPrefix(t, "#") {
				continue
			}
		}

		if before, found := strings.CutSuffix(trimmed, escStr); found {
			if !isEscapedEscape(before, escapeToken) {
				current.WriteString(before)
				current.WriteString(" ")
				continue
			}
		}

		if current.Len() > 0 {
			current.WriteString(line)
			result = append(result, current.String())
			current.Reset()
		} else {
			result = append(result, line)
		}
	}
	if current.Len() > 0 {
		result = append(result, current.String())
	}
	return result
}

// isEscapedEscape checks whether the trailing escape character is itself escaped.
func isEscapedEscape(s string, escapeToken rune) bool {
	count := 0
	for len(s) > 0 {
		r, size := utf8.DecodeLastRuneInString(s)
		if r == escapeToken {
			count++
			s = s[:len(s)-size]
		} else {
			break
		}
	}
	return count%2 == 1
}

// maybeParseHeredoc checks if rest contains heredoc markers and consumes
// following lines until all terminators are found.
func maybeParseHeredoc(rest string, followingLines []string) (string, int, error) {
	if !strings.Contains(rest, "<<") {
		return rest, 0, nil
	}

	words := strings.Fields(rest)
	var terminators []string
	for _, w := range words {
		if m := reHeredoc.FindStringSubmatch(w); m != nil {
			terminators = append(terminators, m[3])
		}
	}

	if len(terminators) == 0 {
		return rest, 0, nil
	}

	var bodies []string
	consumed := 0
	termIdx := 0

	for termIdx < len(terminators) && consumed < len(followingLines) {
		line := strings.TrimRight(followingLines[consumed], "\r\n")
		consumed++
		if strings.TrimSpace(line) == terminators[termIdx] {
			termIdx++
			continue
		}
		bodies = append(bodies, line)
	}

	if termIdx < len(terminators) {
		return "", 0, fmt.Errorf("unterminated heredoc: expected %q", terminators[termIdx])
	}

	return strings.Join(bodies, "\n"), consumed, nil
}

// splitInstruction splits a line into instruction name and remaining arguments.
func splitInstruction(line string) (string, string) {
	parts := strings.SplitN(line, " ", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], strings.TrimSpace(parts[1])
}

// extractFlags extracts leading --flag=value tokens from args.
// Stops at the first non-flag word. Returns the flags map and remaining args string.
func extractFlags(args string) (map[string]string, string) {
	flags := make(map[string]string)
	fields := strings.Fields(args)

	i := 0
	for i < len(fields) {
		f := fields[i]
		if !strings.HasPrefix(f, "--") {
			break
		}
		if key, value, ok := strings.Cut(f, "="); ok {
			flags[strings.TrimPrefix(key, "--")] = value
		} else {
			flags[strings.TrimPrefix(f, "--")] = ""
		}
		i++
	}

	return flags, strings.Join(fields[i:], " ")
}

// ParseEnvValues parses ENV-style KEY=VALUE pairs, correctly handling quotes.
// Supports double-quoted (with escape), single-quoted (literal), and unquoted values.
// Returns a flat key-value slice: ["K1", "V1", "K2", "V2", ...].
func ParseEnvValues(rest string, escapeToken rune) ([]string, error) {
	if rest == "" {
		return nil, fmt.Errorf("empty ENV instruction")
	}

	// Check if using KEY=VALUE format.
	if strings.Contains(strings.Fields(rest)[0], "=") {
		return parseEnvKeyValue(rest, escapeToken), nil
	}

	// Legacy format: ENV KEY VALUE
	key, value, _ := strings.Cut(rest, " ")
	return []string{key, strings.TrimSpace(value)}, nil
}

// parseEnvKeyValue parses KEY=VALUE pairs, handling quotes and escapes.
func parseEnvKeyValue(rest string, escapeToken rune) []string {
	var result []string
	pos := 0

	for pos < len(rest) {
		for pos < len(rest) && (rest[pos] == ' ' || rest[pos] == '\t') {
			pos++
		}
		if pos >= len(rest) {
			break
		}

		eqIdx := strings.IndexByte(rest[pos:], '=')
		if eqIdx < 0 {
			result = append(result, rest[pos:], "")
			break
		}
		key := rest[pos : pos+eqIdx]
		pos += eqIdx + 1

		value, newPos := parseQuotedValue(rest, pos, escapeToken)
		pos = newPos
		result = append(result, key, value)
	}

	return result
}

// parseQuotedValue extracts a value from pos, handling double-quoted, single-quoted,
// and unquoted values.
func parseQuotedValue(s string, pos int, escapeToken rune) (string, int) {
	if pos >= len(s) {
		return "", pos
	}

	switch ch := s[pos]; ch {
	case '"':
		return parseDoubleQuoted(s, pos+1, escapeToken)
	case '\'':
		return parseSingleQuoted(s, pos+1)
	default:
		return parseUnquoted(s, pos)
	}
}

// parseDoubleQuoted parses a double-quoted string, handling escape sequences.
func parseDoubleQuoted(s string, pos int, escapeToken rune) (string, int) {
	var value strings.Builder
	for pos < len(s) {
		r, size := utf8.DecodeRuneInString(s[pos:])
		if r == '"' {
			return value.String(), pos + size
		}
		if r == escapeToken && pos+size < len(s) {
			next, nextSize := utf8.DecodeRuneInString(s[pos+size:])
			value.WriteRune(next)
			pos += size + nextSize
			continue
		}
		value.WriteRune(r)
		pos += size
	}
	return value.String(), pos
}

// parseSingleQuoted parses a single-quoted string (all characters are literal).
func parseSingleQuoted(s string, pos int) (string, int) {
	end := strings.IndexByte(s[pos:], '\'')
	if end < 0 {
		return s[pos:], len(s)
	}
	return s[pos : pos+end], pos + end + 1
}

// parseUnquoted parses an unquoted value, ending at whitespace.
func parseUnquoted(s string, pos int) (string, int) {
	start := pos
	for pos < len(s) && s[pos] != ' ' && s[pos] != '\t' {
		pos++
	}
	return s[start:pos], pos
}

// ParseCommand parses CMD or ENTRYPOINT arguments.
// Supports exec format ["cmd", "arg1", ...] and shell format.
// Arguments containing shell special characters are single-quoted for safety.
func ParseCommand(rest string) string {
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return ""
	}

	if inner, ok := strings.CutPrefix(rest, "["); ok {
		inner = strings.TrimSuffix(strings.TrimSpace(inner), "]")
		var parts []string
		for item := range strings.SplitSeq(inner, ",") {
			item = strings.TrimSpace(item)
			item = strings.Trim(item, "\"'")
			if item != "" {
				parts = append(parts, shellQuote(item))
			}
		}
		return strings.Join(parts, " ")
	}

	return rest
}

// shellQuote wraps a string in single quotes if it contains shell special characters.
func shellQuote(s string) string {
	for _, c := range s {
		if !isShellSafe(c) {
			return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
		}
	}
	return s
}

// isShellSafe checks whether a character is safe for shell use (no quoting needed).
func isShellSafe(c rune) bool {
	if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' {
		return true
	}
	switch c {
	case '-', '_', '.', '/', ':', ',', '+', '=', '@', '%':
		return true
	}
	return false
}

// StripHeredocMarkers removes <<WORD tokens from a string.
func StripHeredocMarkers(s string) string {
	words := strings.Fields(s)
	var filtered []string
	for _, w := range words {
		if reHeredoc.MatchString(w) {
			continue
		}
		filtered = append(filtered, w)
	}
	return strings.Join(filtered, " ")
}

// stripBOM removes a leading UTF-8 BOM from the content.
func stripBOM(s string) string {
	return strings.TrimPrefix(s, "\xef\xbb\xbf")
}
