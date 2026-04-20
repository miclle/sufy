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

	"github.com/sufy-dev/sufy/sandbox"
)

// InjectionRuleList lists all injection rules.
func InjectionRuleList(format string) {
	client := MustNewSandboxClient()
	ctx := context.Background()

	rules, err := client.ListInjectionRules(ctx)
	if err != nil {
		PrintError("list injection rules failed: %v", err)
		return
	}

	if format == FormatJSON {
		PrintJSON(rules)
		return
	}

	if len(rules) == 0 {
		fmt.Println("No injection rules found.")
		return
	}

	tw := NewTable(os.Stdout)
	fmt.Fprintln(tw, "RULE ID\tNAME\tTYPE\tCREATED\tUPDATED")
	for _, r := range rules {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			r.RuleID,
			r.Name,
			injectionTypeLabel(r.Injection),
			FormatTimestamp(r.CreatedAt),
			FormatTimestamp(r.UpdatedAt),
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

// InjectionRuleGet prints injection rule details as JSON.
func InjectionRuleGet(ruleID string) {
	client := MustNewSandboxClient()
	ctx := context.Background()

	r, err := client.GetInjectionRule(ctx, ruleID)
	if err != nil {
		PrintError("get injection rule failed: %v", err)
		return
	}
	PrintJSON(r)
}

// InjectionRuleCreate creates a new injection rule.
func InjectionRuleCreate(name, typ, apiKey, baseURL, headersRaw string) {
	spec, err := BuildInjectionSpec(typ, apiKey, baseURL, ParseKeyValueMap(headersRaw))
	if err != nil {
		PrintError("%v", err)
		return
	}

	client := MustNewSandboxClient()
	ctx := context.Background()

	r, err := client.CreateInjectionRule(ctx, sandbox.CreateInjectionRuleParams{
		Name:      name,
		Injection: spec,
	})
	if err != nil {
		PrintError("create injection rule failed: %v", err)
		return
	}
	PrintSuccess("Injection rule %s created (%s)", r.RuleID, r.Name)
}

// InjectionRuleUpdate updates an existing injection rule.
func InjectionRuleUpdate(ruleID, name, typ, apiKey, baseURL, headersRaw string) {
	params := sandbox.UpdateInjectionRuleParams{}
	if name != "" {
		params.Name = &name
	}
	if typ != "" {
		spec, err := BuildInjectionSpec(typ, apiKey, baseURL, ParseKeyValueMap(headersRaw))
		if err != nil {
			PrintError("%v", err)
			return
		}
		params.Injection = &spec
	}
	if params.Name == nil && params.Injection == nil {
		PrintError("nothing to update: pass --name or --type with related flags")
		return
	}

	client := MustNewSandboxClient()
	ctx := context.Background()

	r, err := client.UpdateInjectionRule(ctx, ruleID, params)
	if err != nil {
		PrintError("update injection rule failed: %v", err)
		return
	}
	PrintSuccess("Injection rule %s updated (%s)", r.RuleID, r.Name)
}

// InjectionRuleDelete deletes one or more injection rules with optional interactive selection.
func InjectionRuleDelete(ruleIDs []string, yes, sel bool) {
	client := MustNewSandboxClient()
	ctx := context.Background()

	if sel {
		rules, err := client.ListInjectionRules(ctx)
		if err != nil {
			PrintError("list injection rules failed: %v", err)
			return
		}
		if len(rules) == 0 {
			fmt.Println("No injection rules found.")
			return
		}
		options := make([]SelectOption, 0, len(rules))
		for _, r := range rules {
			label := fmt.Sprintf("%s (%s)", r.RuleID, r.Name)
			options = append(options, SelectOption{Label: label, Value: r.RuleID})
		}
		selected, err := SelectMultiple("Select injection rules to delete", options)
		if err != nil {
			PrintError("selection cancelled: %v", err)
			return
		}
		if len(selected) == 0 {
			fmt.Println("No rules selected.")
			return
		}
		ruleIDs = selected
	}

	if !ConfirmAction(fmt.Sprintf("Are you sure you want to delete %d injection rule(s)?", len(ruleIDs)), yes) {
		fmt.Println("Aborted.")
		return
	}

	for _, id := range ruleIDs {
		if err := client.DeleteInjectionRule(ctx, id); err != nil {
			PrintError("delete injection rule %s failed: %v", id, err)
			continue
		}
		PrintSuccess("Injection rule %s deleted", id)
	}
}
