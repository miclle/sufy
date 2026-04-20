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

// Package ops provides subcommand registrars for multi-level CLI commands that
// cannot be expressed as .gox classfiles (e.g. three-level nesting).
package ops

import (
	"context"
	"fmt"
	"os"

	"github.com/goplus/cobra"

	"github.com/sufy-dev/sufy/cmd/sufy/internal/cli"
	"github.com/sufy-dev/sufy/sandbox"
)

// RegisterInjectionRuleChildren attaches the injection-rule subcommands (list,
// get, create, update, delete) to the given parent cobra command.
func RegisterInjectionRuleChildren(parent *cobra.Command) {
	parent.AddCommand(
		newInjectionRuleListCmd(),
		newInjectionRuleGetCmd(),
		newInjectionRuleCreateCmd(),
		newInjectionRuleUpdateCmd(),
		newInjectionRuleDeleteCmd(),
	)
}

func newInjectionRuleListCmd() *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List injection rules (alias: ls)",
		Example: `  # List all injection rules
  sufy sandbox injection-rule list
  sufy sbx ir ls

  # Output as JSON
  sufy sandbox injection-rule list --format json
  sufy sbx ir ls --format json`,
		Run: func(cmd *cobra.Command, args []string) {
			injectionRuleList(format)
		},
	}
	cmd.Flags().StringVar(&format, "format", "pretty", "output format: pretty or json")
	return cmd
}

func injectionRuleList(format string) {
	client := cli.MustNewSandboxClient()
	ctx := context.Background()

	rules, err := client.ListInjectionRules(ctx)
	if err != nil {
		cli.PrintError("list injection rules failed: %v", err)
		return
	}

	if format == cli.FormatJSON {
		cli.PrintJSON(rules)
		return
	}

	if len(rules) == 0 {
		fmt.Println("No injection rules found.")
		return
	}

	tw := cli.NewTable(os.Stdout)
	fmt.Fprintln(tw, "RULE ID\tNAME\tTYPE\tCREATED\tUPDATED")
	for _, r := range rules {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			r.RuleID,
			r.Name,
			injectionTypeLabel(r.Injection),
			cli.FormatTimestamp(r.CreatedAt),
			cli.FormatTimestamp(r.UpdatedAt),
		)
	}
	_ = tw.Flush()
}

func injectionTypeLabel(s sandbox.InjectionSpec) string {
	switch {
	case s.OpenAI != nil:
		return "openai"
	case s.Anthropic != nil:
		return "anthropic"
	case s.Gemini != nil:
		return "gemini"
	case s.HTTP != nil:
		return "http"
	default:
		return "unknown"
	}
}

func newInjectionRuleGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "get <ruleID>",
		Aliases: []string{"gt"},
		Short:   "Get injection rule details (alias: gt)",
		Example: `  # Get injection rule details
  sufy sandbox injection-rule get rule-xxxxxxxxxxxx
  sufy sbx ir gt rule-xxxxxxxxxxxx`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			injectionRuleGet(args[0])
		},
	}
}

func injectionRuleGet(ruleID string) {
	client := cli.MustNewSandboxClient()
	ctx := context.Background()

	r, err := client.GetInjectionRule(ctx, ruleID)
	if err != nil {
		cli.PrintError("get injection rule failed: %v", err)
		return
	}
	cli.PrintJSON(r)
}

func newInjectionRuleCreateCmd() *cobra.Command {
	var (
		name, typ, apiKey, baseURL, headers string
	)
	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"cr"},
		Short:   "Create an injection rule (alias: cr)",
		Example: `  # Create an OpenAI injection rule
  sufy sandbox injection-rule create --name openai-default --type openai --api-key sk-xxx
  sufy sbx ir cr --name openai-default --type openai --api-key sk-xxx

  # Create an Anthropic injection rule with custom base URL
  sufy sandbox injection-rule create --name anthropic-proxy --type anthropic --api-key sk-ant --base-url https://anthropic-proxy.example.com

  # Create a custom HTTP injection rule
  sufy sandbox injection-rule create --name api-auth --type http --base-url https://api.example.com --headers "Authorization=Bearer token123,X-Env=prod"`,
		Run: func(cmd *cobra.Command, args []string) {
			injectionRuleCreate(name, typ, apiKey, baseURL, headers)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "rule name (required, unique per user)")
	cmd.Flags().StringVar(&typ, "type", "", "injection type: openai, anthropic, gemini, http")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key for openai/anthropic/gemini injection types (warning: passing secrets via CLI may leak through shell history or process lists)")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "override base URL or target base URL for http injection")
	cmd.Flags().StringVar(&headers, "headers", "", "HTTP headers for custom http injection (comma-separated key=value pairs)")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("type")
	return cmd
}

func injectionRuleCreate(name, typ, apiKey, baseURL, headersRaw string) {
	spec, err := cli.BuildInjectionSpec(typ, apiKey, baseURL, cli.ParseKeyValueMap(headersRaw))
	if err != nil {
		cli.PrintError("%v", err)
		return
	}

	client := cli.MustNewSandboxClient()
	ctx := context.Background()

	r, err := client.CreateInjectionRule(ctx, sandbox.CreateInjectionRuleParams{
		Name:      name,
		Injection: spec,
	})
	if err != nil {
		cli.PrintError("create injection rule failed: %v", err)
		return
	}
	cli.PrintSuccess("Injection rule %s created (%s)", r.RuleID, r.Name)
}

func newInjectionRuleUpdateCmd() *cobra.Command {
	var (
		name, typ, apiKey, baseURL, headers string
	)
	cmd := &cobra.Command{
		Use:     "update <ruleID>",
		Aliases: []string{"up"},
		Short:   "Update an injection rule (alias: up)",
		Example: `  # Update rule name
  sufy sandbox injection-rule update rule-xxxxxxxxxxxx --name new-name
  sufy sbx ir up rule-xxxxxxxxxxxx --name new-name

  # Update to a Gemini injection with custom base URL
  sufy sandbox injection-rule update rule-xxxxxxxxxxxx --type gemini --api-key sk-gem --base-url https://gemini-proxy.example.com

  # Update custom HTTP headers
  sufy sandbox injection-rule update rule-xxxxxxxxxxxx --type http --base-url https://api.example.com --headers "Authorization=Bearer newtoken"`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			injectionRuleUpdate(args[0], name, typ, apiKey, baseURL, headers)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "new rule name")
	cmd.Flags().StringVar(&typ, "type", "", "new injection type: openai, anthropic, gemini, http")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "new API key for openai/anthropic/gemini injection types (warning: passing secrets via CLI may leak through shell history or process lists)")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "new base URL or target base URL for http injection")
	cmd.Flags().StringVar(&headers, "headers", "", "new HTTP headers for custom http injection (comma-separated key=value pairs)")
	return cmd
}

func injectionRuleUpdate(ruleID, name, typ, apiKey, baseURL, headersRaw string) {
	params := sandbox.UpdateInjectionRuleParams{}
	if name != "" {
		params.Name = &name
	}
	if typ != "" {
		spec, err := cli.BuildInjectionSpec(typ, apiKey, baseURL, cli.ParseKeyValueMap(headersRaw))
		if err != nil {
			cli.PrintError("%v", err)
			return
		}
		params.Injection = &spec
	}
	if params.Name == nil && params.Injection == nil {
		cli.PrintError("nothing to update: pass --name or --type with related flags")
		return
	}

	client := cli.MustNewSandboxClient()
	ctx := context.Background()

	r, err := client.UpdateInjectionRule(ctx, ruleID, params)
	if err != nil {
		cli.PrintError("update injection rule failed: %v", err)
		return
	}
	cli.PrintSuccess("Injection rule %s updated (%s)", r.RuleID, r.Name)
}

func newInjectionRuleDeleteCmd() *cobra.Command {
	var yes, sel bool
	cmd := &cobra.Command{
		Use:     "delete [ruleIDs...]",
		Aliases: []string{"dl"},
		Short:   "Delete one or more injection rules (alias: dl)",
		Example: `  # Delete a single rule (skip confirmation)
  sufy sandbox injection-rule delete rule-xxxxxxxxxxxx -y
  sufy sbx ir dl rule-xxxxxxxxxxxx -y

  # Delete multiple rules
  sufy sandbox injection-rule delete rule-aaa rule-bbb -y
  sufy sbx ir dl rule-aaa rule-bbb -y

  # Interactively select rules to delete
  sufy sandbox injection-rule delete -s
  sufy sbx ir dl -s`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 && !sel {
				_ = cmd.Usage()
				return
			}
			injectionRuleDelete(args, yes, sel)
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation")
	cmd.Flags().BoolVarP(&sel, "select", "s", false, "interactively select rules to delete")
	return cmd
}

func injectionRuleDelete(ruleIDs []string, yes, sel bool) {
	client := cli.MustNewSandboxClient()
	ctx := context.Background()

	if sel {
		rules, err := client.ListInjectionRules(ctx)
		if err != nil {
			cli.PrintError("list injection rules failed: %v", err)
			return
		}
		if len(rules) == 0 {
			fmt.Println("No injection rules found.")
			return
		}
		options := make([]cli.SelectOption, 0, len(rules))
		for _, r := range rules {
			label := fmt.Sprintf("%s (%s)", r.RuleID, r.Name)
			options = append(options, cli.SelectOption{Label: label, Value: r.RuleID})
		}
		selected, err := cli.SelectMultiple("Select injection rules to delete", options)
		if err != nil {
			cli.PrintError("selection cancelled: %v", err)
			return
		}
		if len(selected) == 0 {
			fmt.Println("No rules selected.")
			return
		}
		ruleIDs = selected
	}

	if !cli.ConfirmAction(fmt.Sprintf("Are you sure you want to delete %d injection rule(s)?", len(ruleIDs)), yes) {
		fmt.Println("Aborted.")
		return
	}

	for _, id := range ruleIDs {
		if err := client.DeleteInjectionRule(ctx, id); err != nil {
			cli.PrintError("delete injection rule %s failed: %v", id, err)
			continue
		}
		cli.PrintSuccess("Injection rule %s deleted", id)
	}
}
