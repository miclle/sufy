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
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// stdinScanner is a package-level scanner so multiple prompts in the same
// session share a single buffered reader (avoids losing buffered bytes).
var stdinScanner = bufio.NewScanner(os.Stdin)

// IsInteractive reports whether stdin is attached to a terminal. Automation
// scenarios (pipes, CI, ssh without -t) return false so callers can fail fast
// instead of blocking on prompts.
func IsInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// readLine reads a single line from stdin. Returns io.EOF when the user
// cancels input (Ctrl+D) so callers can treat cancellation distinctly.
func readLine() (string, error) {
	if !stdinScanner.Scan() {
		if err := stdinScanner.Err(); err != nil {
			return "", err
		}
		return "", io.EOF
	}
	return strings.TrimSpace(stdinScanner.Text()), nil
}

// ConfirmAction asks the user for confirmation. Returns true if the user
// confirms. When yes is true, confirmation is skipped. In non-interactive
// environments without --yes, returns false so destructive actions abort
// instead of hanging on a prompt that nobody will answer.
func ConfirmAction(prompt string, yes bool) bool {
	if yes {
		return true
	}
	if !IsInteractive() {
		PrintError("%s: refusing to prompt in non-interactive mode (use --yes to confirm)", prompt)
		return false
	}
	fmt.Printf("%s [y/N] ", prompt)
	line, err := readLine()
	if err != nil {
		fmt.Println()
		return false
	}
	line = strings.ToLower(line)
	return line == "y" || line == "yes"
}

// SelectOption represents a single item in a select picker.
type SelectOption struct {
	Label string
	Value string
}

// errSelectCancelled is returned when the user cancels selection (EOF).
var errSelectCancelled = errors.New("selection cancelled")

// SelectMultiple presents a numbered list and asks the user to pick one or
// more entries by index. Accepts comma-separated indices (e.g. "1,3,5"),
// ranges (e.g. "1-3"), or the literal "all". Empty input returns an empty
// slice. Returns an error if stdin is not a TTY or the user cancels.
func SelectMultiple(title string, options []SelectOption) ([]string, error) {
	if !IsInteractive() {
		return nil, fmt.Errorf("cannot prompt for selection in non-interactive mode")
	}
	if len(options) == 0 {
		return nil, nil
	}

	fmt.Println(title)
	for i, o := range options {
		fmt.Printf("  %d) %s\n", i+1, o.Label)
	}
	fmt.Printf("Enter numbers separated by commas (e.g. 1,3 or 1-3), 'all', or empty to skip: ")

	for {
		line, err := readLine()
		if err != nil {
			fmt.Println()
			return nil, errSelectCancelled
		}
		if line == "" {
			return nil, nil
		}
		if strings.EqualFold(line, "all") {
			values := make([]string, len(options))
			for i, o := range options {
				values[i] = o.Value
			}
			return values, nil
		}

		indices, err := parseIndexList(line, len(options))
		if err != nil {
			fmt.Printf("Invalid input: %v. Try again: ", err)
			continue
		}

		seen := make(map[int]struct{}, len(indices))
		values := make([]string, 0, len(indices))
		for _, idx := range indices {
			if _, ok := seen[idx]; ok {
				continue
			}
			seen[idx] = struct{}{}
			values = append(values, options[idx].Value)
		}
		return values, nil
	}
}

// parseIndexList parses inputs like "1,3,5" or "1-3,7" into zero-based
// indices, validating each entry is within [1, max].
func parseIndexList(input string, max int) ([]int, error) {
	parts := strings.Split(input, ",")
	out := make([]int, 0, len(parts))
	for _, raw := range parts {
		part := strings.TrimSpace(raw)
		if part == "" {
			continue
		}
		if strings.Contains(part, "-") {
			bounds := strings.SplitN(part, "-", 2)
			start, err1 := strconv.Atoi(strings.TrimSpace(bounds[0]))
			end, err2 := strconv.Atoi(strings.TrimSpace(bounds[1]))
			if err1 != nil || err2 != nil {
				return nil, fmt.Errorf("invalid range %q", part)
			}
			if start < 1 || end > max || start > end {
				return nil, fmt.Errorf("range %q out of bounds (1-%d)", part, max)
			}
			for i := start; i <= end; i++ {
				out = append(out, i-1)
			}
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid number %q", part)
		}
		if n < 1 || n > max {
			return nil, fmt.Errorf("number %d out of bounds (1-%d)", n, max)
		}
		out = append(out, n-1)
	}
	return out, nil
}

// SelectOne presents a numbered list and asks the user to pick exactly one
// entry by index. Returns an error if stdin is not a TTY or the user cancels.
func SelectOne(title string, options []SelectOption) (string, error) {
	if !IsInteractive() {
		return "", fmt.Errorf("cannot prompt for selection in non-interactive mode")
	}
	if len(options) == 0 {
		return "", fmt.Errorf("no options to choose from")
	}

	fmt.Println(title)
	for i, o := range options {
		fmt.Printf("  %d) %s\n", i+1, o.Label)
	}
	fmt.Printf("Enter a number (1-%d): ", len(options))

	for {
		line, err := readLine()
		if err != nil {
			fmt.Println()
			return "", errSelectCancelled
		}
		n, err := strconv.Atoi(line)
		if err != nil || n < 1 || n > len(options) {
			fmt.Printf("Invalid choice, enter a number between 1 and %d: ", len(options))
			continue
		}
		return options[n-1].Value, nil
	}
}

// PromptInput asks the user for a text input. The optional validate callback
// is called on each entry; a non-nil error reprompts the user with the
// validation message. Returns an error if stdin is not a TTY or the user
// cancels.
func PromptInput(title, description string, validate func(string) error) (string, error) {
	if !IsInteractive() {
		return "", fmt.Errorf("cannot prompt for input in non-interactive mode")
	}
	if description != "" {
		fmt.Printf("%s (%s): ", title, description)
	} else {
		fmt.Printf("%s: ", title)
	}
	for {
		line, err := readLine()
		if err != nil {
			fmt.Println()
			return "", errSelectCancelled
		}
		if validate != nil {
			if err := validate(line); err != nil {
				fmt.Printf("%v. Try again: ", err)
				continue
			}
		}
		return line, nil
	}
}
