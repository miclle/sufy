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

func strPtr(s string) *string { return &s }

func main() {
	c := exampleutil.MustNewClient()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 1. Create an injection rule (OpenAI).
	fmt.Println("=== create injection rule ===")
	rule, err := c.CreateInjectionRule(ctx, sandbox.CreateInjectionRuleParams{
		Name: "sdk-example-openai",
		Injection: sandbox.InjectionSpec{
			OpenAI: &sandbox.OpenAIInjection{
				APIKey: strPtr("sk-example-key"),
			},
		},
	})
	if err != nil {
		log.Fatalf("CreateInjectionRule failed: %v", err)
	}
	fmt.Printf("rule created: id=%s, name=%s\n", rule.RuleID, rule.Name)

	// Clean up the rule at the end.
	defer func() {
		fmt.Println("\n=== delete injection rule ===")
		if err := c.DeleteInjectionRule(context.Background(), rule.RuleID); err != nil {
			fmt.Printf("DeleteInjectionRule failed: %v\n", err)
		} else {
			fmt.Printf("rule %s deleted\n", rule.RuleID)
		}
	}()

	// 2. List all injection rules.
	fmt.Println("\n=== list injection rules ===")
	rules, err := c.ListInjectionRules(ctx)
	if err != nil {
		log.Fatalf("ListInjectionRules failed: %v", err)
	}
	fmt.Printf("found %d rules:\n", len(rules))
	for _, r := range rules {
		injType := describeInjectionType(r.Injection)
		fmt.Printf("  - %s (name: %s, type: %s, created: %s)\n",
			r.RuleID, r.Name, injType, r.CreatedAt.Format(time.RFC3339))
	}

	// 3. Get the rule by ID.
	fmt.Println("\n=== get injection rule ===")
	got, err := c.GetInjectionRule(ctx, rule.RuleID)
	if err != nil {
		log.Fatalf("GetInjectionRule failed: %v", err)
	}
	fmt.Printf("rule: id=%s, name=%s, type=%s\n", got.RuleID, got.Name, describeInjectionType(got.Injection))

	// 4. Update the rule (change the API key).
	fmt.Println("\n=== update injection rule ===")
	newName := "sdk-example-openai-updated"
	updated, err := c.UpdateInjectionRule(ctx, rule.RuleID, sandbox.UpdateInjectionRuleParams{
		Name: &newName,
		Injection: &sandbox.InjectionSpec{
			OpenAI: &sandbox.OpenAIInjection{
				APIKey: strPtr("sk-updated-key"),
			},
		},
	})
	if err != nil {
		log.Fatalf("UpdateInjectionRule failed: %v", err)
	}
	fmt.Printf("rule updated: id=%s, name=%s\n", updated.RuleID, updated.Name)

	// 5. Show how to reference a saved rule when creating a sandbox.
	fmt.Println("\n=== sandbox creation with injections (reference) ===")
	fmt.Println("To create a sandbox that references this rule:")
	fmt.Printf("  c.Create(ctx, sandbox.CreateParams{\n")
	fmt.Printf("      TemplateID: \"base\",\n")
	fmt.Printf("      Injections: &[]sandbox.SandboxInjectionSpec{\n")
	fmt.Printf("          {ByID: strPtr(\"%s\")},\n", rule.RuleID)
	fmt.Printf("      },\n")
	fmt.Printf("  })\n")
	fmt.Println("\nOr inline an injection directly:")
	fmt.Printf("  c.Create(ctx, sandbox.CreateParams{\n")
	fmt.Printf("      TemplateID: \"base\",\n")
	fmt.Printf("      Injections: &[]sandbox.SandboxInjectionSpec{\n")
	fmt.Printf("          {OpenAI: &sandbox.OpenAIInjection{APIKey: strPtr(\"sk-xxx\")}},\n")
	fmt.Printf("          {Anthropic: &sandbox.AnthropicInjection{APIKey: strPtr(\"ak-xxx\")}},\n")
	fmt.Printf("      },\n")
	fmt.Printf("  })\n")

	// 6. Deletion runs via defer above.
}

// describeInjectionType returns a human-readable label for the injection type.
func describeInjectionType(spec sandbox.InjectionSpec) string {
	switch {
	case spec.OpenAI != nil:
		return "openai"
	case spec.Anthropic != nil:
		return "anthropic"
	case spec.Gemini != nil:
		return "gemini"
	case spec.HTTP != nil:
		return "http"
	default:
		return "unknown"
	}
}
