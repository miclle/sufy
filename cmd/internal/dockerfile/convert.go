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

package dockerfile

import (
	"fmt"
	"strings"

	"github.com/sufy-dev/sufy/sandbox"
)

// ConvertResult holds the output of converting a Dockerfile to sandbox format.
type ConvertResult struct {
	// BaseImage is the image name from the FROM instruction.
	BaseImage string

	// Steps is the list of build steps converted from Dockerfile instructions.
	Steps []sandbox.TemplateStep

	// StartCmd is the startup command extracted from CMD or ENTRYPOINT.
	StartCmd string

	// ReadyCmd defaults to "sleep 20" when CMD/ENTRYPOINT is present (matching e2b behavior).
	ReadyCmd string

	// Warnings contains non-fatal parsing issues.
	Warnings []string
}

// Convert parses Dockerfile content and converts instructions to sandbox TemplateStep.
// It matches e2b v2 build system behavior:
//   - Prepends USER root and WORKDIR / steps
//   - Appends USER user if no USER instruction is found
//   - Appends WORKDIR /home/user if no WORKDIR instruction is found
func Convert(content string) (*ConvertResult, error) {
	parsed, err := Parse(content)
	if err != nil {
		return nil, err
	}

	result := &ConvertResult{
		Warnings: parsed.Warnings,
	}
	var steps []sandbox.TemplateStep
	hasUser := false
	hasWorkdir := false
	fromCount := 0

	// Prepend default steps (matching e2b behavior).
	steps = append(steps, makeStep("USER", "root"))
	steps = append(steps, makeStep("WORKDIR", "/"))

	for _, inst := range parsed.Instructions {
		switch inst.Name {
		case "FROM":
			fromCount++
			if fromCount > 1 {
				result.Warnings = append(result.Warnings,
					"multi-stage build detected; using the last FROM stage as the runtime base image")
			}
			result.BaseImage = extractImage(inst.Args)

		case "RUN":
			if inst.Args == "" {
				return nil, fmt.Errorf("line %d: empty RUN instruction", inst.Line)
			}
			args := []string{inst.Args}
			steps = append(steps, sandbox.TemplateStep{
				Type: "RUN",
				Args: &args,
			})

		case "COPY", "ADD":
			user, src, dest, err := parseCopyArgs(inst.Args, inst.Flags)
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid %s instruction: %w", inst.Line, inst.Name, err)
			}
			args := []string{src, dest, user, ""}
			steps = append(steps, sandbox.TemplateStep{
				Type: "COPY",
				Args: &args,
			})

		case "WORKDIR":
			if inst.Args == "" {
				return nil, fmt.Errorf("line %d: empty WORKDIR instruction", inst.Line)
			}
			hasWorkdir = true
			args := []string{inst.Args}
			steps = append(steps, sandbox.TemplateStep{
				Type: "WORKDIR",
				Args: &args,
			})

		case "USER":
			if inst.Args == "" {
				return nil, fmt.Errorf("line %d: empty USER instruction", inst.Line)
			}
			hasUser = true
			args := []string{inst.Args}
			steps = append(steps, sandbox.TemplateStep{
				Type: "USER",
				Args: &args,
			})

		case "ENV":
			envArgs, err := ParseEnvValues(inst.Args, parsed.EscapeToken)
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid ENV instruction: %w", inst.Line, err)
			}
			steps = append(steps, sandbox.TemplateStep{
				Type: "ENV",
				Args: &envArgs,
			})

		case "ARG":
			argArgs, hasDefault := parseArgValues(inst.Args)
			if hasDefault {
				steps = append(steps, sandbox.TemplateStep{
					Type: "ENV",
					Args: &argArgs,
				})
			}

		case "CMD":
			result.StartCmd = ParseCommand(inst.Args)
			result.ReadyCmd = "sleep 20"

		case "ENTRYPOINT":
			result.StartCmd = ParseCommand(inst.Args)
			result.ReadyCmd = "sleep 20"
		}
	}

	// Append defaults if not explicitly set (matching e2b behavior).
	if !hasUser {
		steps = append(steps, makeStep("USER", "user"))
	}
	if !hasWorkdir {
		steps = append(steps, makeStep("WORKDIR", "/home/user"))
	}

	if result.BaseImage == "" {
		return nil, fmt.Errorf("no FROM instruction found in Dockerfile")
	}

	result.Steps = steps
	return result, nil
}

// extractImage extracts the image name from FROM args, ignoring AS alias.
func extractImage(args string) string {
	for f := range strings.FieldsSeq(args) {
		if strings.ToUpper(f) == "AS" {
			break
		}
		return f
	}
	return args
}

// parseCopyArgs extracts user, src, and dest from a COPY/ADD instruction.
func parseCopyArgs(args string, flags map[string]string) (user, src, dest string, err error) {
	if chown, ok := flags["chown"]; ok {
		if u, _, found := strings.Cut(chown, ":"); found {
			user = u
		} else {
			user = chown
		}
	}

	args = StripHeredocMarkers(args)

	fields := strings.Fields(args)
	if len(fields) < 2 {
		return "", "", "", fmt.Errorf("COPY/ADD requires at least source and destination")
	}

	dest = fields[len(fields)-1]
	src = strings.Join(fields[:len(fields)-1], " ")
	return user, src, dest, nil
}

// parseArgValues parses ARG name[=default_value] into ["name", "value"].
// Returns the result and whether a default value was present.
func parseArgValues(rest string) ([]string, bool) {
	key, value, hasDefault := strings.Cut(rest, "=")
	return []string{strings.TrimSpace(key), strings.TrimSpace(value)}, hasDefault
}

// makeStep creates a simple TemplateStep.
func makeStep(typ string, args ...string) sandbox.TemplateStep {
	a := make([]string, len(args))
	copy(a, args)
	return sandbox.TemplateStep{
		Type: typ,
		Args: &a,
	}
}
