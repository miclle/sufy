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
	"strings"

	"github.com/sufy-dev/sufy/sandbox"
)

// TemplateList lists all sandbox templates.
func TemplateList(format string) {
	client := MustNewSandboxClient()
	ctx := context.Background()

	templates, err := client.ListTemplates(ctx, nil)
	if err != nil {
		PrintError("list templates failed: %v", err)
		return
	}

	if format == FormatJSON {
		PrintJSON(templates)
		return
	}

	if len(templates) == 0 {
		fmt.Println("No templates found.")
		return
	}

	tw := NewTable(os.Stdout)
	fmt.Fprintln(tw, "TEMPLATE ID\tALIASES\tBUILD STATUS\tCPU\tMEMORY\tPUBLIC\tUPDATED")
	for _, t := range templates {
		aliases := "-"
		if len(t.Aliases) > 0 {
			aliases = strings.Join(t.Aliases, ", ")
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%d MiB\t%t\t%s\n",
			t.TemplateID,
			aliases,
			string(t.BuildStatus),
			t.CPUCount,
			t.MemoryMB,
			t.Public,
			FormatTimestamp(t.UpdatedAt),
		)
	}
	_ = tw.Flush()
}

// TemplateGet prints template details as JSON.
func TemplateGet(templateID string) {
	client := MustNewSandboxClient()
	ctx := context.Background()

	t, err := client.GetTemplate(ctx, templateID, nil)
	if err != nil {
		PrintError("get template failed: %v", err)
		return
	}
	PrintJSON(t)
}

// TemplateDelete deletes one or more templates with optional interactive selection.
func TemplateDelete(templateIDs []string, yes, sel bool) {
	client := MustNewSandboxClient()
	ctx := context.Background()

	if sel {
		templates, err := client.ListTemplates(ctx, nil)
		if err != nil {
			PrintError("list templates failed: %v", err)
			return
		}
		if len(templates) == 0 {
			fmt.Println("No templates found.")
			return
		}
		options := make([]SelectOption, 0, len(templates))
		for _, t := range templates {
			label := t.TemplateID
			if len(t.Aliases) > 0 {
				label = fmt.Sprintf("%s (%s)", t.TemplateID, strings.Join(t.Aliases, ", "))
			}
			options = append(options, SelectOption{Label: label, Value: t.TemplateID})
		}
		selected, err := SelectMultiple("Select templates to delete", options)
		if err != nil {
			PrintError("selection cancelled: %v", err)
			return
		}
		if len(selected) == 0 {
			fmt.Println("No templates selected.")
			return
		}
		templateIDs = selected
	}

	if !ConfirmAction(fmt.Sprintf("Are you sure you want to delete %d template(s)?", len(templateIDs)), yes) {
		fmt.Println("Aborted.")
		return
	}

	for _, id := range templateIDs {
		if err := client.DeleteTemplate(ctx, id); err != nil {
			PrintError("delete template %s failed: %v", id, err)
			continue
		}
		PrintSuccess("Template %s deleted", id)
	}
}

// TemplateSetPublic publishes or unpublishes one or more templates.
func TemplateSetPublic(templateIDs []string, public, yes, sel bool, action string) {
	client := MustNewSandboxClient()
	ctx := context.Background()

	if sel {
		templates, err := client.ListTemplates(ctx, nil)
		if err != nil {
			PrintError("list templates failed: %v", err)
			return
		}
		if len(templates) == 0 {
			fmt.Println("No templates found.")
			return
		}
		options := make([]SelectOption, 0, len(templates))
		for _, t := range templates {
			label := t.TemplateID
			if len(t.Aliases) > 0 {
				label = fmt.Sprintf("%s (%s)", t.TemplateID, strings.Join(t.Aliases, ", "))
			}
			publicStr := "private"
			if t.Public {
				publicStr = "public"
			}
			label = fmt.Sprintf("%s [%s]", label, publicStr)
			options = append(options, SelectOption{Label: label, Value: t.TemplateID})
		}
		selected, err := SelectMultiple(fmt.Sprintf("Select templates to %s", action), options)
		if err != nil {
			PrintError("selection cancelled: %v", err)
			return
		}
		if len(selected) == 0 {
			fmt.Println("No templates selected.")
			return
		}
		templateIDs = selected
	}

	if !ConfirmAction(fmt.Sprintf("Are you sure you want to %s %d template(s)?", action, len(templateIDs)), yes) {
		fmt.Println("Aborted.")
		return
	}

	for _, id := range templateIDs {
		err := client.UpdateTemplate(ctx, id, sandbox.UpdateTemplateParams{Public: &public})
		if err != nil {
			PrintError("update template %s failed: %v", id, err)
			continue
		}
		if public {
			PrintSuccess("Template %s published", id)
		} else {
			PrintSuccess("Template %s unpublished", id)
		}
	}
}

// TemplateBuilds prints the build status for a template build.
func TemplateBuilds(templateID, buildID string) {
	client := MustNewSandboxClient()
	ctx := context.Background()

	info, err := client.GetTemplateBuildStatus(ctx, templateID, buildID, nil)
	if err != nil {
		PrintError("get build status failed: %v", err)
		return
	}
	fmt.Printf("Template ID: %s\n", info.TemplateID)
	fmt.Printf("Build ID:    %s\n", info.BuildID)
	fmt.Printf("Status:      %s\n", info.Status)
	if len(info.Logs) > 0 {
		fmt.Println("Logs:")
		for _, l := range info.Logs {
			fmt.Println("  ", l)
		}
	}
}
