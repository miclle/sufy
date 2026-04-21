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
	"fmt"

	"github.com/charmbracelet/huh"
)

// ConfirmAction asks the user for confirmation using a simple stdin prompt.
// Returns true if the user confirms. When yes is true, confirmation is skipped.
func ConfirmAction(prompt string, yes bool) bool {
	if yes {
		return true
	}
	fmt.Printf("%s [y/N] ", prompt)
	var confirm string
	fmt.Scanln(&confirm)
	return confirm == "y" || confirm == "Y"
}

// SelectOption represents a single item in a multi-select picker.
type SelectOption struct {
	Label string
	Value string
}

// SelectMultiple presents an interactive multi-select TUI and returns the
// selected values. Returns nil and an error if the user cancels.
func SelectMultiple(title string, options []SelectOption) ([]string, error) {
	huhOpts := make([]huh.Option[string], 0, len(options))
	for _, o := range options {
		huhOpts = append(huhOpts, huh.NewOption(o.Label, o.Value))
	}

	var selected []string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title(title).
				Options(huhOpts...).
				Value(&selected),
		),
	)
	if err := form.Run(); err != nil {
		return nil, err
	}
	return selected, nil
}
