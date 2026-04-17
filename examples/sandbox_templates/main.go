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

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sufy-dev/sufy/examples/internal/exampleutil"
	"github.com/sufy-dev/sufy/sandbox"
)

func main() {
	c := exampleutil.MustNewClient()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// 1. List all templates.
	fmt.Println("=== list templates ===")
	templates, err := c.ListTemplates(ctx, nil)
	if err != nil {
		log.Fatalf("ListTemplates failed: %v", err)
	}
	fmt.Printf("found %d templates:\n", len(templates))
	for _, tmpl := range templates {
		aliases := "-"
		if len(tmpl.Aliases) > 0 {
			aliases = fmt.Sprintf("%v", tmpl.Aliases)
		}
		fmt.Printf("  - %s\n", tmpl.TemplateID)
		fmt.Printf("    aliases: %s, public: %v\n", aliases, tmpl.Public)
		fmt.Printf("    CPU: %d cores, memory: %d MB, disk: %d MB, envd: %s\n",
			tmpl.CPUCount, tmpl.MemoryMB, tmpl.DiskSizeMB, tmpl.EnvdVersion)
		fmt.Printf("    build status: %s, build count: %d, spawn count: %d\n",
			tmpl.BuildStatus, tmpl.BuildCount, tmpl.SpawnCount)
		fmt.Printf("    created: %s, updated: %s\n",
			tmpl.CreatedAt.Format(time.RFC3339), tmpl.UpdatedAt.Format(time.RFC3339))
	}

	// 2. Fetch a single template's details.
	if len(templates) > 0 {
		fmt.Println("\n=== get template details ===")
		detail, err := c.GetTemplate(ctx, templates[0].TemplateID, nil)
		if err != nil {
			log.Fatalf("GetTemplate failed: %v", err)
		}
		fmt.Printf("template: %s, builds: %d\n", detail.TemplateID, len(detail.Builds))
		for i, build := range detail.Builds {
			if i >= 3 {
				fmt.Printf("  ... %d more omitted\n", len(detail.Builds)-3)
				break
			}
			fmt.Printf("  - %s (status: %s, CPU: %d, memory: %d MB)\n",
				build.BuildID, build.Status, build.CPUCount, build.MemoryMB)
		}
	}

	// 3. Create a template.
	fmt.Println("\n=== create template ===")
	cpuCount := int32(2)
	memoryMB := int32(512)
	templateName := "sdk-example-template"
	resp, err := c.CreateTemplate(ctx, sandbox.CreateTemplateParams{
		Name:     &templateName,
		CPUCount: &cpuCount,
		MemoryMB: &memoryMB,
	})
	if err != nil {
		log.Fatalf("CreateTemplate failed: %v", err)
	}
	templateID := resp.TemplateID
	buildID := resp.BuildID
	fmt.Printf("template created: %s (build: %s)\n", templateID, buildID)

	// Clean up the template at the end.
	defer func() {
		fmt.Println("\n=== delete template ===")
		if err := c.DeleteTemplate(context.Background(), templateID); err != nil {
			fmt.Printf("DeleteTemplate failed: %v\n", err)
		} else {
			fmt.Printf("template %s deleted\n", templateID)
		}
	}()

	// 4. Update the template.
	fmt.Println("\n=== update template ===")
	public := true
	if err := c.UpdateTemplate(ctx, templateID, sandbox.UpdateTemplateParams{
		Public: &public,
	}); err != nil {
		log.Fatalf("UpdateTemplate failed: %v", err)
	}
	fmt.Println("template updated to public")

	// 5. Fetch build status.
	fmt.Println("\n=== get build status ===")
	buildInfo, err := c.GetTemplateBuildStatus(ctx, templateID, buildID, nil)
	if err != nil {
		fmt.Printf("GetTemplateBuildStatus failed: %v\n", err)
	} else {
		fmt.Printf("build %s: status=%s, template=%s\n", buildInfo.BuildID, buildInfo.Status, buildInfo.TemplateID)
	}

	// 6. Fetch build logs.
	fmt.Println("\n=== get build logs ===")
	buildLogs, err := c.GetTemplateBuildLogs(ctx, templateID, buildID, nil)
	if err != nil {
		fmt.Printf("GetTemplateBuildLogs failed: %v\n", err)
	} else {
		fmt.Printf("build logs (%d entries):\n", len(buildLogs.Logs))
		for i, entry := range buildLogs.Logs {
			if i >= 5 {
				fmt.Printf("  ... %d more omitted\n", len(buildLogs.Logs)-5)
				break
			}
			step := "-"
			if entry.Step != nil {
				step = *entry.Step
			}
			fmt.Printf("  [%s] [%s] %s: %s\n",
				entry.Timestamp.Format(time.RFC3339), entry.Level, step, entry.Message)
		}
	}

	// 7. Assign / delete template tags.
	//
	// AssignTemplateTags reassigns tags between *already tagged* template
	// builds via a "name:tag" target reference. A freshly created template
	// has no initial tag, so these APIs are documented here without a live
	// call to keep the example deterministic. Real callers should invoke
	// them against a template whose build is complete and already tagged.
	fmt.Println("\n=== template tags (API reference) ===")
	fmt.Println("AssignTemplateTags: c.AssignTemplateTags(ctx, sandbox.ManageTagsParams{")
	fmt.Println("    Target: \"<template-name>:<existing-tag>\",")
	fmt.Println("    Tags:   []string{\"latest\"},")
	fmt.Println("})")
	fmt.Println("DeleteTemplateTags: c.DeleteTemplateTags(ctx, sandbox.DeleteTagsParams{")
	fmt.Println("    Name: \"<template-name>\",")
	fmt.Println("    Tags: []string{\"stale-tag\"},")
	fmt.Println("})")

	// 9. Look up a template by alias.
	fmt.Println("\n=== get template by alias ===")
	if len(templates) > 0 && len(templates[0].Aliases) > 0 {
		alias := templates[0].Aliases[0]
		aliasResp, err := c.GetTemplateByAlias(ctx, alias)
		if err != nil {
			fmt.Printf("GetTemplateByAlias failed: %v\n", err)
		} else {
			fmt.Printf("alias '%s' -> template: %s (public: %v)\n", alias, aliasResp.TemplateID, aliasResp.Public)
		}
	} else {
		fmt.Println("no template alias available, skipping")
	}

	// 10. Get a template file upload URL.
	fmt.Println("\n=== get template file upload URL ===")
	fileUpload, err := c.GetTemplateFiles(ctx, templateID, "example-hash-value")
	if err != nil {
		fmt.Printf("GetTemplateFiles failed: %v\n", err)
	} else {
		url := "-"
		if fileUpload.URL != nil {
			url = *fileUpload.URL
		}
		fmt.Printf("file already present: %v, upload URL: %s\n", fileUpload.Present, url)
	}

	// 11. Start a template build.
	fmt.Println("\n=== start template build ===")
	fromImage := "ubuntu:latest"
	if err := c.StartTemplateBuild(ctx, templateID, buildID, sandbox.StartTemplateBuildParams{
		FromImage: &fromImage,
	}); err != nil {
		fmt.Printf("StartTemplateBuild failed (may already be building): %v\n", err)
	} else {
		fmt.Println("build started")
	}

	// 12. Wait for the build to finish.
	fmt.Println("\n=== wait for build ===")
	finalBuild, err := c.WaitForBuild(ctx, templateID, buildID, sandbox.WithPollInterval(3*time.Second))
	if err != nil {
		fmt.Printf("WaitForBuild failed: %v\n", err)
	} else {
		fmt.Printf("build finished: %s (status: %s)\n", finalBuild.BuildID, finalBuild.Status)
	}

	// 13. Template deletion runs via defer above.
}
