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

// BuildFields is the subset of template build parameters that can be
// supplied via the configuration file.
//
// Defined separately so the sandboxcfg package does not depend on the
// caller's BuildInfo type (which would create a cycle). Callers copy
// their own fields into BuildFields before ApplyTo and copy them back
// afterwards.
type BuildFields struct {
	// TemplateID is the template ID.
	TemplateID string
	// Name is the template name.
	Name string
	// Dockerfile is the Dockerfile path.
	Dockerfile string
	// Path is the build context directory.
	Path string
	// FromImage is the base Docker image.
	FromImage string
	// FromTemplate is the base template.
	FromTemplate string
	// StartCmd is the container startup command.
	StartCmd string
	// ReadyCmd is the readiness check command.
	ReadyCmd string
	// CPUCount is the sandbox CPU count.
	CPUCount int32
	// MemoryMB is the sandbox memory size in MiB.
	MemoryMB int32
	// NoCache forces a full rebuild, ignoring cache.
	NoCache bool
	// NoCacheChanged signals whether the CLI explicitly set --no-cache.
	NoCacheChanged bool
}

// ApplyTo merges values from FileConfig into dst: a file value is used
// only when the corresponding dst field is its zero value.
//
// Returns the list of TOML keys that were overridden by the CLI — i.e.
// dst already had a non-zero value that differs from the file value.
func (c *FileConfig) ApplyTo(dst *BuildFields) []string {
	var overrides []string

	applyString := func(fileVal string, dstVal *string, key string) {
		if fileVal == "" {
			return
		}
		if *dstVal == "" {
			*dstVal = fileVal
			return
		}
		if *dstVal != fileVal {
			overrides = append(overrides, key)
		}
	}
	applyInt32 := func(fileVal int32, dstVal *int32, key string) {
		if fileVal == 0 {
			return
		}
		if *dstVal == 0 {
			*dstVal = fileVal
			return
		}
		if *dstVal != fileVal {
			overrides = append(overrides, key)
		}
	}
	applyBool := func(fileVal bool, dstVal *bool, key string) {
		// When the file does not define the key, respect the CLI value.
		if !c.defined[key] {
			return
		}
		if *dstVal == fileVal {
			return
		}
		if dst.NoCacheChanged {
			overrides = append(overrides, key)
			return
		}
		if !*dstVal {
			*dstVal = fileVal
			return
		}
		overrides = append(overrides, key)
	}

	applyString(c.TemplateID, &dst.TemplateID, "template_id")
	applyString(c.Name, &dst.Name, "name")
	applyString(c.Dockerfile, &dst.Dockerfile, "dockerfile")
	applyString(c.Path, &dst.Path, "path")
	applyString(c.FromImage, &dst.FromImage, "from_image")
	applyString(c.FromTemplate, &dst.FromTemplate, "from_template")
	applyString(c.StartCmd, &dst.StartCmd, "start_cmd")
	applyString(c.ReadyCmd, &dst.ReadyCmd, "ready_cmd")
	applyInt32(c.CPUCount, &dst.CPUCount, "cpu_count")
	applyInt32(c.MemoryMB, &dst.MemoryMB, "memory_mb")
	applyBool(c.NoCache, &dst.NoCache, "no_cache")

	return overrides
}
