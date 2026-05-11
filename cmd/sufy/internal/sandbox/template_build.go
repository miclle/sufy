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
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/sufy-dev/sufy/cmd/sufy/internal/dockerfile"
	"github.com/sufy-dev/sufy/cmd/sufy/internal/sandbox/templatecfg"
	sdk "github.com/sufy-dev/sufy/sandbox"
)

// BuildInfo holds all parameters for the template build command.
type BuildInfo struct {
	Name         string // template name (new template)
	TemplateID   string // existing template ID (rebuild)
	FromImage    string // base Docker image
	FromTemplate string // base template
	StartCmd     string // startup command
	ReadyCmd     string // readiness check command
	CPUCount     int32  // CPU cores
	MemoryMB     int32  // memory in MiB
	Wait         bool   // block until done
	NoCache bool   // force full rebuild
	// NoCacheChanged reports whether --no-cache was explicitly set on the CLI.
	NoCacheChanged bool
	Dockerfile     string // path to Dockerfile (v2 build)
	Path           string // build context directory
	// ConfigPath is an explicit path to sufy.sandbox.toml.
	// When empty, the file is auto-discovered from the current working directory.
	ConfigPath string
}

// TemplateBuild creates or rebuilds a template.
// If TemplateID is provided, it starts a new build for the existing template.
// Otherwise, it creates a new template with the given Name and starts a build.
//
// Config file support: when sufy.sandbox.toml is present, CLI flag > file > built-in default.
// When --template-id is not provided, the command first looks up the template by --name
// on the server: a hit triggers a rebuild, a miss falls back to create. After a successful
// first create, template_id is written back to the config file.
func TemplateBuild(info BuildInfo) {
	// Capture "CLI did not provide TemplateID" before merging so we can decide
	// later whether to write template_id back to the config file.
	noIDBeforeMerge := info.TemplateID == ""
	cliFromImage := info.FromImage != ""
	cliFromTemplate := info.FromTemplate != ""

	// Load config file and merge (CLI > file > default).
	var cfg *templatecfg.FileConfig
	if info.ConfigPath != "" {
		loaded, cErr := templatecfg.Load(info.ConfigPath)
		if cErr != nil {
			PrintError("load config: %v", cErr)
			return
		}
		if loaded == nil {
			PrintError("config file not found: %s", info.ConfigPath)
			return
		}
		cfg = loaded
	} else {
		loaded, cErr := templatecfg.LoadFromCwd()
		if cErr != nil {
			PrintError("load config: %v", cErr)
			return
		}
		cfg = loaded
	}

	if cfg != nil {
		fields := templatecfg.BuildFields{
			TemplateID:     info.TemplateID,
			Name:           info.Name,
			Dockerfile:     info.Dockerfile,
			Path:           info.Path,
			FromImage:      info.FromImage,
			FromTemplate:   info.FromTemplate,
			StartCmd:       info.StartCmd,
			ReadyCmd:       info.ReadyCmd,
			CPUCount:       info.CPUCount,
			MemoryMB:       info.MemoryMB,
			NoCache:        info.NoCache,
			NoCacheChanged: info.NoCacheChanged,
		}
		overrides := cfg.ApplyTo(&fields)
		info.TemplateID = fields.TemplateID
		info.Name = fields.Name
		info.Dockerfile = fields.Dockerfile
		info.Path = fields.Path
		info.FromImage = fields.FromImage
		info.FromTemplate = fields.FromTemplate
		info.StartCmd = fields.StartCmd
		info.ReadyCmd = fields.ReadyCmd
		info.CPUCount = fields.CPUCount
		info.MemoryMB = fields.MemoryMB
		info.NoCache = fields.NoCache

		for _, key := range overrides {
			fmt.Fprintf(os.Stderr, "[config] CLI overrides %s from %s\n", key, cfg.SourcePath())
		}
	}

	if err := validateRebuildSourceSelection(info, cliFromImage, cliFromTemplate); err != nil {
		PrintError("%v", err)
		return
	}

	client := MustNewSandboxClient()
	ctx := context.Background()
	templateID := info.TemplateID
	buildID := ""
	// foundByName means the current template was located by name lookup;
	// in that case the name alone uniquely identifies the template, so we
	// don't write template_id back to the toml — that keeps the config
	// environment-agnostic.
	foundByName := false

	// Read the Dockerfile once: rebuild requests need its content in the
	// request body, and the v2 build path needs it for step parsing.
	var dockerfileContent string
	if info.Dockerfile != "" {
		raw, rfErr := os.ReadFile(info.Dockerfile)
		if rfErr != nil {
			PrintError("read Dockerfile: %v", rfErr)
			return
		}
		dockerfileContent = string(raw)
	}

	// When --template-id is not set, look up by name on the server first:
	// a hit triggers a rebuild, a miss falls back to create. This way
	// switching environments doesn't require syncing template_id by hand.
	if templateID == "" && info.Name != "" {
		existingID, lookupErr := lookupTemplateIDByName(ctx, client, info.Name)
		if lookupErr != nil {
			PrintError("lookup template by name %q: %v", info.Name, lookupErr)
			return
		}
		if existingID != "" {
			templateID = existingID
			info.TemplateID = existingID
			foundByName = true
			fmt.Fprintf(os.Stderr, "[lookup] template %q resolved to %s (rebuild)\n", info.Name, templateID)
		}
	}

	if templateID == "" {
		if info.Name == "" {
			PrintError("template name (--name) or template ID (--template-id) is required")
			return
		}
		if err := validateBuildSourceSelection(info); err != nil {
			PrintError("%v", err)
			return
		}

		createParams := sdk.CreateTemplateParams{
			Name: &info.Name,
		}
		if info.CPUCount > 0 {
			createParams.CPUCount = &info.CPUCount
		}
		if info.MemoryMB > 0 {
			createParams.MemoryMB = &info.MemoryMB
		}

		fmt.Printf("Creating template %s...\n", info.Name)
		resp, err := client.CreateTemplate(ctx, createParams)
		if err != nil {
			PrintError("create template failed: %v", err)
			return
		}
		templateID = resp.TemplateID
		buildID = resp.BuildID
		PrintSuccess("Template %s created (build ID: %s)", templateID, buildID)
	} else {
		// The rebuild API requires Dockerfile content in the request body,
		// so --template-id without --dockerfile is rejected.
		if dockerfileContent == "" {
			PrintError("--dockerfile is required when rebuilding an existing template (--template-id); --from-image and --from-template only apply to new templates")
			return
		}
		rebuildParams := sdk.RebuildTemplateParams{
			Dockerfile: dockerfileContent,
		}
		if info.CPUCount > 0 {
			rebuildParams.CPUCount = &info.CPUCount
		}
		if info.MemoryMB > 0 {
			rebuildParams.MemoryMB = &info.MemoryMB
		}
		if info.StartCmd != "" {
			rebuildParams.StartCmd = &info.StartCmd
		}
		if info.ReadyCmd != "" {
			rebuildParams.ReadyCmd = &info.ReadyCmd
		}

		fmt.Printf("Requesting new build for template %s...\n", templateID)
		resp, rErr := client.RebuildTemplate(ctx, templateID, rebuildParams)
		if rErr != nil {
			PrintError("rebuild template failed: %v", rErr)
			return
		}
		buildID = resp.BuildID
		PrintSuccess("New build requested (build ID: %s)", buildID)
	}

	if info.Dockerfile != "" {
		if err := buildFromDockerfile(ctx, client, templateID, buildID, dockerfileContent, info); err != nil {
			PrintError("%v", err)
			return
		}
	} else {
		buildParams := sdk.StartTemplateBuildParams{}
		if info.FromImage != "" {
			buildParams.FromImage = &info.FromImage
		}
		if info.FromTemplate != "" {
			buildParams.FromTemplate = &info.FromTemplate
		}
		if info.StartCmd != "" {
			buildParams.StartCmd = &info.StartCmd
		}
		if info.ReadyCmd != "" {
			buildParams.ReadyCmd = &info.ReadyCmd
		}
		if info.NoCache {
			force := true
			buildParams.Force = &force
		}

		fmt.Printf("Starting build for template %s (build ID: %s)...\n", templateID, buildID)
		if err := client.StartTemplateBuild(ctx, templateID, buildID, buildParams); err != nil {
			PrintError("start build failed: %v", err)
			return
		}
	}

	if !info.Wait {
		writeTemplateIDToConfigIfNeeded(cfg, noIDBeforeMerge && !foundByName, templateID)
		fmt.Printf("Build started. Use 'sufy sandbox template builds %s %s' to check status.\n", templateID, buildID)
		return
	}

	// Stream build logs with Ctrl+C support.
	fmt.Println("Waiting for build to complete...")

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	var cursor *int64
	for {
		logs, err := client.GetTemplateBuildLogs(ctx, templateID, buildID, &sdk.GetBuildLogsParams{
			Cursor: cursor,
		})
		if err == nil && logs != nil {
			for _, entry := range logs.Logs {
				fmt.Printf("[%s] %s %s\n",
					FormatTimestamp(entry.Timestamp),
					LogLevelBadge(string(entry.Level)),
					entry.Message,
				)
				ts := entry.Timestamp.UnixMilli() + 1
				cursor = &ts
			}
		}

		buildInfo, err := client.GetTemplateBuildStatus(ctx, templateID, buildID, nil)
		if err != nil {
			PrintError("get build status failed: %v", err)
			return
		}

		if buildInfo.Status == "ready" || buildInfo.Status == "error" {
			if buildInfo.Status == "error" {
				PrintError("build failed")
			} else {
				PrintSuccess("Build completed!")
				writeTemplateIDToConfigIfNeeded(cfg, noIDBeforeMerge && !foundByName, templateID)
			}
			fmt.Printf("Template ID:  %s\n", buildInfo.TemplateID)
			fmt.Printf("Build ID:     %s\n", buildInfo.BuildID)
			fmt.Printf("Status:       %s\n", buildInfo.Status)

			if buildInfo.Status == "ready" {
				printSDKExamples(buildInfo.TemplateID)
			}
			return
		}

		select {
		case <-ctx.Done():
			PrintError("build watch cancelled")
			return
		case <-time.After(3 * time.Second):
		}
	}
}

// buildFromDockerfile handles the v2 Dockerfile build pipeline:
// parse Dockerfile -> upload COPY files -> start build with steps.
func buildFromDockerfile(ctx context.Context, client *sdk.Client, templateID, buildID, dockerfileContent string, info BuildInfo) error {
	contextPath := info.Path
	if contextPath == "" {
		contextPath = filepath.Dir(info.Dockerfile)
	}
	contextPath, err := filepath.Abs(contextPath)
	if err != nil {
		return fmt.Errorf("resolve context path: %w", err)
	}

	result, err := dockerfile.Convert(dockerfileContent)
	if err != nil {
		return fmt.Errorf("parse Dockerfile: %w", err)
	}
	fmt.Printf("Parsed Dockerfile: base image=%s, %d steps\n", result.BaseImage, len(result.Steps))

	ignorePatterns := dockerfile.ReadDockerignore(contextPath)

	for i := range result.Steps {
		step := &result.Steps[i]
		if step.Type != "COPY" || step.Args == nil || len(*step.Args) < 2 {
			continue
		}
		args := *step.Args
		src, dest := args[0], args[1]

		hash, err := dockerfile.ComputeFilesHash(src, dest, contextPath, ignorePatterns)
		if err != nil {
			return fmt.Errorf("compute file hash for COPY %s %s: %w", src, dest, err)
		}
		step.FilesHash = &hash

		fileInfo, err := client.GetTemplateFiles(ctx, templateID, hash)
		if err != nil {
			return fmt.Errorf("get template files for hash %s: %w", hash, err)
		}

		if !fileInfo.Present && fileInfo.URL != nil {
			fmt.Printf("Uploading files for COPY %s %s...\n", src, dest)
			if err := dockerfile.CollectAndUpload(ctx, *fileInfo.URL, src, contextPath, ignorePatterns); err != nil {
				return fmt.Errorf("upload files for COPY %s %s: %w", src, dest, err)
			}
		} else if fileInfo.Present {
			fmt.Printf("Files for COPY %s %s already uploaded (cached)\n", src, dest)
		}
	}

	buildParams := buildParamsFromDockerfileResult(result, info)

	startCmd := result.StartCmd
	if info.StartCmd != "" {
		startCmd = info.StartCmd
	}
	if startCmd != "" {
		buildParams.StartCmd = &startCmd
	}

	readyCmd := result.ReadyCmd
	if info.ReadyCmd != "" {
		readyCmd = info.ReadyCmd
	}
	if readyCmd != "" {
		buildParams.ReadyCmd = &readyCmd
	}

	if info.NoCache {
		force := true
		buildParams.Force = &force
	}

	fmt.Printf("Starting build for template %s (build ID: %s)...\n", templateID, buildID)
	if err := client.StartTemplateBuild(ctx, templateID, buildID, buildParams); err != nil {
		return fmt.Errorf("start build: %w", err)
	}

	return nil
}

// printSDKExamples prints SDK usage examples for the given template ID.
func printSDKExamples(templateID string) {
	fmt.Println()
	fmt.Println("Template is ready! Use it with the SDK:")

	fmt.Printf("\n%s\n", ColorInfo.Sprint("Go:"))
	fmt.Printf("  sb, _ := client.CreateAndWait(ctx, sdk.CreateParams{\n")
	fmt.Printf("      TemplateID: %q,\n", templateID)
	fmt.Printf("  })\n")

	fmt.Printf("\n%s\n", ColorInfo.Sprint("Python:"))
	fmt.Printf("  sandbox = client.sandboxes.create(%q)\n", templateID)

	fmt.Printf("\n%s\n", ColorInfo.Sprint("TypeScript:"))
	fmt.Printf("  const sandbox = await client.sandboxes.create(%q)\n", templateID)
	fmt.Println()
}

// validateBuildSourceSelection checks input combination for creating a new template.
func validateBuildSourceSelection(info BuildInfo) error {
	if info.FromImage != "" && info.FromTemplate != "" {
		return fmt.Errorf("cannot specify both --from-image and --from-template")
	}
	if info.Dockerfile == "" && info.FromImage == "" && info.FromTemplate == "" {
		return fmt.Errorf("--from-image, --from-template, or --dockerfile is required")
	}
	return nil
}

// validateRebuildSourceSelection validates the from-* sources on the rebuild path.
//
// It only errors when the CLI explicitly passes --from-image / --from-template
// while going through a rebuild — those flags apply to creating a new template
// only. A from_template coming from sufy.sandbox.toml is still forwarded on
// rebuild to avoid an accidental "FROM scratch" in the Dockerfile.
func validateRebuildSourceSelection(info BuildInfo, cliFromImage, cliFromTemplate bool) error {
	if info.TemplateID == "" {
		return nil
	}
	if cliFromImage || cliFromTemplate {
		return fmt.Errorf("cannot specify --from-image or --from-template when rebuilding an existing template")
	}
	return nil
}

// buildParamsFromDockerfileResult picks the base image / template from the
// Dockerfile parse result and BuildInfo.
// Precedence: FromTemplate > FromImage > Dockerfile FROM.
func buildParamsFromDockerfileResult(result *dockerfile.ConvertResult, info BuildInfo) sdk.StartTemplateBuildParams {
	buildParams := sdk.StartTemplateBuildParams{
		Steps: &result.Steps,
	}

	switch {
	case info.FromTemplate != "":
		buildParams.FromTemplate = &info.FromTemplate
	case info.FromImage != "":
		buildParams.FromImage = &info.FromImage
	default:
		buildParams.FromImage = &result.BaseImage
	}

	return buildParams
}

// writeTemplateIDToConfigIfNeeded writes template_id back to the config
// file after the first successful create, so subsequent runs can locate
// the template via the same config.
//
// No write happens when cfg is nil (no config file discovered), when the
// template_id was already provided by the config, or when the template
// was located via a name lookup — the first has no file, the latter two
// can already locate the template reliably.
func writeTemplateIDToConfigIfNeeded(cfg *templatecfg.FileConfig, noIDBeforeMerge bool, templateID string) {
	if cfg == nil || !noIDBeforeMerge || cfg.SourcePath() == "" {
		return
	}
	if wErr := templatecfg.WriteTemplateID(cfg.SourcePath(), templateID); wErr != nil {
		fmt.Fprintf(os.Stderr, "[templatecfg] warning: failed to write template_id to %s: %v\n",
			cfg.SourcePath(), wErr)
		return
	}
	PrintSuccess("Written template_id to %s (please commit this file)", cfg.SourcePath())
}
