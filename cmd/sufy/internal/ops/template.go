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

package ops

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/goplus/cobra"

	"github.com/sufy-dev/sufy/cmd/sufy/internal/cli"
	"github.com/sufy-dev/sufy/sandbox"
)

// RegisterTemplateChildren attaches the template subcommands (list, get, delete,
// publish, unpublish, builds) to the given parent cobra command.
func RegisterTemplateChildren(parent *cobra.Command) {
	parent.AddCommand(
		newTemplateListCmd(),
		newTemplateGetCmd(),
		newTemplateDeleteCmd(),
		newTemplatePublishCmd(true),
		newTemplatePublishCmd(false),
		newTemplateBuildCmd(),
		newTemplateInitCmd(),
		newTemplateBuildsCmd(),
	)
}

func newTemplateListCmd() *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List sandbox templates (alias: ls)",
		Example: `  # List all templates
  sufy sandbox template list
  sufy sbx tpl ls

  # Output as JSON
  sufy sandbox template list --format json
  sufy sbx tpl ls --format json`,
		Run: func(cmd *cobra.Command, args []string) {
			templateList(format)
		},
	}
	cmd.Flags().StringVar(&format, "format", "pretty", "output format: pretty or json")
	return cmd
}

func templateList(format string) {
	client := cli.MustNewSandboxClient()
	ctx := context.Background()

	templates, err := client.ListTemplates(ctx, nil)
	if err != nil {
		cli.PrintError("list templates failed: %v", err)
		return
	}

	if format == cli.FormatJSON {
		cli.PrintJSON(templates)
		return
	}

	if len(templates) == 0 {
		fmt.Println("No templates found.")
		return
	}

	tw := cli.NewTable(os.Stdout)
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
			cli.FormatTimestamp(t.UpdatedAt),
		)
	}
	_ = tw.Flush()
}

func newTemplateGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "get <templateID>",
		Aliases: []string{"gt"},
		Short:   "Get template details (alias: gt)",
		Example: `  # Get template details
  sufy sandbox template get tmpl-xxxxxxxxxxxx
  sufy sbx tpl gt tmpl-xxxxxxxxxxxx`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			templateGet(args[0])
		},
	}
}

func templateGet(templateID string) {
	client := cli.MustNewSandboxClient()
	ctx := context.Background()

	t, err := client.GetTemplate(ctx, templateID, nil)
	if err != nil {
		cli.PrintError("get template failed: %v", err)
		return
	}
	cli.PrintJSON(t)
}

func newTemplateDeleteCmd() *cobra.Command {
	var yes, sel bool
	cmd := &cobra.Command{
		Use:     "delete [templateIDs...]",
		Aliases: []string{"dl"},
		Short:   "Delete one or more templates (alias: dl)",
		Example: `  # Delete a single template (skip confirmation)
  sufy sandbox template delete tmpl-xxxxxxxxxxxx -y
  sufy sbx tpl dl tmpl-xxxxxxxxxxxx -y

  # Delete multiple templates
  sufy sandbox template delete tmpl-aaa tmpl-bbb -y
  sufy sbx tpl dl tmpl-aaa tmpl-bbb -y

  # Interactively select templates to delete
  sufy sandbox template delete -s
  sufy sbx tpl dl -s`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 && !sel {
				_ = cmd.Usage()
				return
			}
			templateDelete(args, yes, sel)
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation")
	cmd.Flags().BoolVarP(&sel, "select", "s", false, "interactively select templates to delete")
	return cmd
}

func templateDelete(templateIDs []string, yes, sel bool) {
	client := cli.MustNewSandboxClient()
	ctx := context.Background()

	if sel {
		templates, err := client.ListTemplates(ctx, nil)
		if err != nil {
			cli.PrintError("list templates failed: %v", err)
			return
		}
		if len(templates) == 0 {
			fmt.Println("No templates found.")
			return
		}
		options := make([]cli.SelectOption, 0, len(templates))
		for _, t := range templates {
			label := t.TemplateID
			if len(t.Aliases) > 0 {
				label = fmt.Sprintf("%s (%s)", t.TemplateID, strings.Join(t.Aliases, ", "))
			}
			options = append(options, cli.SelectOption{Label: label, Value: t.TemplateID})
		}
		selected, err := cli.SelectMultiple("Select templates to delete", options)
		if err != nil {
			cli.PrintError("selection cancelled: %v", err)
			return
		}
		if len(selected) == 0 {
			fmt.Println("No templates selected.")
			return
		}
		templateIDs = selected
	}

	if !cli.ConfirmAction(fmt.Sprintf("Are you sure you want to delete %d template(s)?", len(templateIDs)), yes) {
		fmt.Println("Aborted.")
		return
	}

	for _, id := range templateIDs {
		if err := client.DeleteTemplate(ctx, id); err != nil {
			cli.PrintError("delete template %s failed: %v", id, err)
			continue
		}
		cli.PrintSuccess("Template %s deleted", id)
	}
}

// newTemplatePublishCmd returns either "publish" or "unpublish" depending on
// the public flag.
func newTemplatePublishCmd(public bool) *cobra.Command {
	use, alias, short := "publish [templateIDs...]", "pb", "Publish templates (make public) (alias: pb)"
	action := "publish"
	example := `  # Publish a single template (skip confirmation)
  sufy sandbox template publish tmpl-xxxxxxxxxxxx -y
  sufy sbx tpl pb tmpl-xxxxxxxxxxxx -y

  # Interactively select templates to publish
  sufy sandbox template publish -s
  sufy sbx tpl pb -s`
	if !public {
		use, alias, short = "unpublish [templateIDs...]", "upb", "Unpublish templates (make private) (alias: upb)"
		action = "unpublish"
		example = `  # Unpublish a single template (skip confirmation)
  sufy sandbox template unpublish tmpl-xxxxxxxxxxxx -y
  sufy sbx tpl upb tmpl-xxxxxxxxxxxx -y

  # Interactively select templates to unpublish
  sufy sandbox template unpublish -s
  sufy sbx tpl upb -s`
	}
	var yes, sel bool
	cmd := &cobra.Command{
		Use:     use,
		Aliases: []string{alias},
		Short:   short,
		Example: example,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 && !sel {
				_ = cmd.Usage()
				return
			}
			templateSetPublic(args, public, yes, sel, action)
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation")
	cmd.Flags().BoolVarP(&sel, "select", "s", false, "interactively select templates")
	return cmd
}

func templateSetPublic(templateIDs []string, public, yes, sel bool, action string) {
	client := cli.MustNewSandboxClient()
	ctx := context.Background()

	if sel {
		templates, err := client.ListTemplates(ctx, nil)
		if err != nil {
			cli.PrintError("list templates failed: %v", err)
			return
		}
		if len(templates) == 0 {
			fmt.Println("No templates found.")
			return
		}
		options := make([]cli.SelectOption, 0, len(templates))
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
			options = append(options, cli.SelectOption{Label: label, Value: t.TemplateID})
		}
		selected, err := cli.SelectMultiple(fmt.Sprintf("Select templates to %s", action), options)
		if err != nil {
			cli.PrintError("selection cancelled: %v", err)
			return
		}
		if len(selected) == 0 {
			fmt.Println("No templates selected.")
			return
		}
		templateIDs = selected
	}

	if !cli.ConfirmAction(fmt.Sprintf("Are you sure you want to %s %d template(s)?", action, len(templateIDs)), yes) {
		fmt.Println("Aborted.")
		return
	}

	for _, id := range templateIDs {
		err := client.UpdateTemplate(ctx, id, sandbox.UpdateTemplateParams{Public: &public})
		if err != nil {
			cli.PrintError("update template %s failed: %v", id, err)
			continue
		}
		if public {
			cli.PrintSuccess("Template %s published", id)
		} else {
			cli.PrintSuccess("Template %s unpublished", id)
		}
	}
}

func newTemplateInitCmd() *cobra.Command {
	info := cli.InitInfo{}
	cmd := &cobra.Command{
		Use:     "init",
		Aliases: []string{"it"},
		Short:   "Initialize a new template project (alias: it)",
		Long:    "Scaffold a new template project with boilerplate files for the selected language.",
		Example: `  # Interactive mode
  sufy sandbox template init
  sufy sbx tpl it

  # Non-interactive mode
  sufy sandbox template init --name my-template --language go
  sufy sbx tpl it --name my-template --language go

  # Non-interactive mode with custom path
  sufy sandbox template init --name my-api --language typescript --path ./my-api
  sufy sbx tpl it --name my-api --language typescript --path ./my-api`,
		Run: func(cmd *cobra.Command, args []string) {
			cli.TemplateInit(info)
		},
	}
	cmd.Flags().StringVar(&info.Name, "name", "", "template project name")
	cmd.Flags().StringVar(&info.Language, "language", "", "programming language (go, typescript, python)")
	cmd.Flags().StringVar(&info.Path, "path", "", "output directory (defaults to ./<name>)")
	return cmd
}

func newTemplateBuildCmd() *cobra.Command {
	info := cli.BuildInfo{}
	cmd := &cobra.Command{
		Use:     "build",
		Aliases: []string{"bd"},
		Short:   "Build a template (alias: bd)",
		Long: `Create a new template and build it, or rebuild an existing template.

Supports three build modes:
  1. --from-image: Build from a base Docker image
  2. --from-template: Build from an existing template
  3. --dockerfile: Build from a Dockerfile (v2 build system)`,
		Example: `  # Create and build a new template from a Docker image
  sufy sandbox template build --name my-template --from-image ubuntu:22.04 --wait
  sufy sbx tpl bd --name my-template --from-image ubuntu:22.04 --wait

  # Build from a Dockerfile
  sufy sandbox template build --name my-template --dockerfile ./Dockerfile --wait
  sufy sbx tpl bd --name my-template --dockerfile ./Dockerfile --wait

  # Build from a Dockerfile with a custom context directory
  sufy sandbox template build --name my-template --dockerfile ./Dockerfile --path ./context --wait
  sufy sbx tpl bd --name my-template --dockerfile ./Dockerfile --path ./context --wait

  # Rebuild an existing template
  sufy sandbox template build --template-id tmpl-xxxxxxxxxxxx --from-image ubuntu:22.04
  sufy sbx tpl bd --template-id tmpl-xxxxxxxxxxxx --from-image ubuntu:22.04

  # Force rebuild without cache
  sufy sandbox template build --template-id tmpl-xxxxxxxxxxxx --no-cache --wait
  sufy sbx tpl bd --template-id tmpl-xxxxxxxxxxxx --no-cache --wait`,
		Run: func(cmd *cobra.Command, args []string) {
			cli.TemplateBuild(info)
		},
	}
	cmd.Flags().StringVar(&info.Name, "name", "", "template name (for creating a new template)")
	cmd.Flags().StringVar(&info.TemplateID, "template-id", "", "existing template ID (for rebuilding)")
	cmd.Flags().StringVar(&info.FromImage, "from-image", "", "base Docker image")
	cmd.Flags().StringVar(&info.FromTemplate, "from-template", "", "base template")
	cmd.Flags().StringVar(&info.StartCmd, "start-cmd", "", "command to run after build")
	cmd.Flags().StringVar(&info.ReadyCmd, "ready-cmd", "", "readiness check command")
	cmd.Flags().Int32Var(&info.CPUCount, "cpu", 0, "sandbox CPU count")
	cmd.Flags().Int32Var(&info.MemoryMB, "memory", 0, "sandbox memory size in MiB")
	cmd.Flags().BoolVar(&info.Wait, "wait", false, "wait for build to complete")
	cmd.Flags().BoolVar(&info.NoCache, "no-cache", false, "force full rebuild ignoring cache")
	cmd.Flags().StringVar(&info.Dockerfile, "dockerfile", "", "path to Dockerfile (enables v2 build)")
	cmd.Flags().StringVar(&info.Path, "path", "", "build context directory (defaults to Dockerfile's parent)")
	return cmd
}

func newTemplateBuildsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "builds <templateID> <buildID>",
		Aliases: []string{"bds"},
		Short:   "View template build status (alias: bds)",
		Example: `  # View build status
  sufy sandbox template builds tmpl-xxxxxxxxxxxx build-xxxxxxxxxxxx
  sufy sbx tpl bds tmpl-xxxxxxxxxxxx build-xxxxxxxxxxxx`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			templateBuilds(args[0], args[1])
		},
	}
}

func templateBuilds(templateID, buildID string) {
	client := cli.MustNewSandboxClient()
	ctx := context.Background()

	info, err := client.GetTemplateBuildStatus(ctx, templateID, buildID, nil)
	if err != nil {
		cli.PrintError("get build status failed: %v", err)
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
