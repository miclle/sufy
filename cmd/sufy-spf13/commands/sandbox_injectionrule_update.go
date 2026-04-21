package commands

import (
	"github.com/spf13/cobra"

	"github.com/sufy-dev/sufy/cmd/internal/sandbox"
)

// sandboxInjectionRuleUpdateCmd corresponds to cmd/sufy/sandbox_injectionrule_update_cmd.gox.
var sandboxInjectionRuleUpdateCmd = func() *cobra.Command {
	var (
		name, typ, apiKey, baseURL, headers string
	)
	cmd := &cobra.Command{
		Use:     "update <ruleID>",
		Short:   "Update an injection rule (alias: up)",
		Aliases: []string{"up"},
		Args:    cobra.ExactArgs(1),
		Example: `  # Update rule name
  sufy sandbox injection-rule update rule-xxxxxxxxxxxx --name new-name
  sufy sbx ir up rule-xxxxxxxxxxxx --name new-name

  # Update to a Gemini injection with custom base URL
  sufy sandbox injection-rule update rule-xxxxxxxxxxxx --type gemini --api-key sk-gem --base-url https://gemini-proxy.example.com

  # Update custom HTTP headers
  sufy sandbox injection-rule update rule-xxxxxxxxxxxx --type http --base-url https://api.example.com --headers "Authorization=Bearer newtoken"`,
		Run: func(_ *cobra.Command, args []string) {
			sandbox.InjectionRuleUpdate(args[0], name, typ, apiKey, baseURL, headers)
		},
	}
	f := cmd.Flags()
	f.StringVar(&name, "name", "", "new rule name")
	f.StringVar(&typ, "type", "", "new injection type: openai, anthropic, gemini, http")
	f.StringVar(&apiKey, "api-key", "", "new API key for openai/anthropic/gemini injection types (warning: passing secrets via CLI may leak through shell history or process lists)")
	f.StringVar(&baseURL, "base-url", "", "new base URL or target base URL for http injection")
	f.StringVar(&headers, "headers", "", "new HTTP headers for custom http injection (comma-separated key=value pairs)")
	return cmd
}()

func init() {
	sandboxInjectionRuleCmd.AddCommand(sandboxInjectionRuleUpdateCmd)
}
