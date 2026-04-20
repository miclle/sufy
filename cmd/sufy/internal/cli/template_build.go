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

package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/sufy-dev/sufy/cmd/sufy/internal/dockerfile"
	"github.com/sufy-dev/sufy/sandbox"
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
	NoCache      bool   // force full rebuild
	Dockerfile   string // path to Dockerfile (v2 build)
	Path         string // build context directory
}

// TemplateBuild creates or rebuilds a template.
// If TemplateID is provided, it starts a new build for the existing template.
// Otherwise, it creates a new template with the given Name and starts a build.
func TemplateBuild(info BuildInfo) {
	client := MustNewSandboxClient()
	ctx := context.Background()
	templateID := info.TemplateID
	buildID := ""

	if templateID == "" {
		if info.Name == "" {
			PrintError("template name (--name) or template ID (--template-id) is required")
			return
		}

		createParams := sandbox.CreateTemplateParams{
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
		tmpl, err := client.GetTemplate(ctx, templateID, nil)
		if err != nil {
			PrintError("get template failed: %v", err)
			return
		}
		if len(tmpl.Builds) > 0 {
			buildID = tmpl.Builds[len(tmpl.Builds)-1].BuildID
		} else {
			PrintError("no builds found for template, cannot rebuild")
			return
		}
	}

	if info.Dockerfile != "" {
		if err := buildFromDockerfile(ctx, client, templateID, buildID, info); err != nil {
			PrintError("%v", err)
			return
		}
	} else {
		if info.FromImage == "" && info.FromTemplate == "" {
			PrintError("--from-image, --from-template, or --dockerfile is required")
			return
		}

		buildParams := sandbox.StartTemplateBuildParams{}
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
		fmt.Printf("Build started. Use 'sufy sandbox template builds %s %s' to check status.\n", templateID, buildID)
		return
	}

	// Stream build logs with Ctrl+C support.
	fmt.Println("Waiting for build to complete...")

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	var cursor *int64
	for {
		logs, err := client.GetTemplateBuildLogs(ctx, templateID, buildID, &sandbox.GetBuildLogsParams{
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
func buildFromDockerfile(ctx context.Context, client *sandbox.Client, templateID, buildID string, info BuildInfo) error {
	content, err := os.ReadFile(info.Dockerfile)
	if err != nil {
		return fmt.Errorf("read Dockerfile: %w", err)
	}

	contextPath := info.Path
	if contextPath == "" {
		contextPath = filepath.Dir(info.Dockerfile)
	}
	contextPath, err = filepath.Abs(contextPath)
	if err != nil {
		return fmt.Errorf("resolve context path: %w", err)
	}

	result, err := dockerfile.Convert(string(content))
	if err != nil {
		return fmt.Errorf("parse Dockerfile: %w", err)
	}
	fmt.Printf("Parsed Dockerfile: base image=%s, %d steps\n", result.BaseImage, len(result.Steps))

	ignorePatterns := dockerfile.ReadDockerignore(contextPath)

	// Process COPY steps: compute file hashes and upload files.
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

	buildParams := sandbox.StartTemplateBuildParams{
		FromImage: &result.BaseImage,
		Steps:     &result.Steps,
	}

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
	fmt.Printf("  sb, _ := client.CreateAndWait(ctx, sandbox.CreateParams{\n")
	fmt.Printf("      TemplateID: %q,\n", templateID)
	fmt.Printf("  })\n")

	fmt.Printf("\n%s\n", ColorInfo.Sprint("Python:"))
	fmt.Printf("  sandbox = client.sandboxes.create(%q)\n", templateID)

	fmt.Printf("\n%s\n", ColorInfo.Sprint("TypeScript:"))
	fmt.Printf("  const sandbox = await client.sandboxes.create(%q)\n", templateID)
	fmt.Println()
}
