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

// Package sandboxcfg loads, merges, and rewrites the sufy.sandbox.toml
// configuration file used by `sufy sandbox template` commands.
package templatecfg

// DefaultFileName is the default template configuration file name.
const DefaultFileName = "sufy.sandbox.toml"

// FileConfig maps every configurable field in sufy.sandbox.toml.
//
// All fields are optional: missing values fall back to CLI flags or
// built-in defaults.
type FileConfig struct {
	// TemplateID is the template ID. After the first successful build,
	// sufy writes it back to the file.
	TemplateID string `toml:"template_id"`

	// Name is the template name. Used only on first build when TemplateID
	// is not set.
	Name string `toml:"name"`

	// Dockerfile is the Dockerfile path; setting it enables v2 build.
	Dockerfile string `toml:"dockerfile"`

	// Path is the build context directory; defaults to the Dockerfile's
	// parent directory.
	Path string `toml:"path"`

	// FromImage is the base Docker image.
	FromImage string `toml:"from_image"`

	// FromTemplate is the base template.
	FromTemplate string `toml:"from_template"`

	// StartCmd is the container startup command.
	StartCmd string `toml:"start_cmd"`

	// ReadyCmd is the readiness check command.
	ReadyCmd string `toml:"ready_cmd"`

	// CPUCount is the sandbox CPU count.
	CPUCount int32 `toml:"cpu_count"`

	// MemoryMB is the sandbox memory size in MiB.
	MemoryMB int32 `toml:"memory_mb"`

	// NoCache forces a full rebuild, ignoring cache.
	NoCache bool `toml:"no_cache"`

	// sourcePath records the absolute path of the loaded file, used for
	// writeback. Unexported.
	sourcePath string

	// defined records the set of TOML keys explicitly present in the file.
	// Unexported.
	defined map[string]bool
}

// SourcePath returns the absolute path of the source file, or "" when
// the FileConfig was not loaded from disk.
func (c *FileConfig) SourcePath() string {
	return c.sourcePath
}
