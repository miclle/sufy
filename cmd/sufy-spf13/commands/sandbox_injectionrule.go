package commands

import "github.com/spf13/cobra"

// sandboxInjectionRuleCmd corresponds to cmd/sufy/sandbox_injectionrule_cmd.gox.
var sandboxInjectionRuleCmd = &cobra.Command{
	Use:     "injection-rule",
	Short:   "Manage sandbox injection rules (alias: ir)",
	Aliases: []string{"ir"},
	Example: `  # View injection-rule subcommands
  sufy sandbox injection-rule -h
  sufy sbx ir -h

  # List all injection rules
  sufy sandbox injection-rule list
  sufy sbx ir ls

  # Create an OpenAI injection rule
  sufy sandbox injection-rule create --name openai-default --type openai --api-key sk-xxx
  sufy sbx ir cr --name openai-default --type openai --api-key sk-xxx

  # Get injection rule details
  sufy sandbox injection-rule get rule-xxxxxxxxxxxx
  sufy sbx ir gt rule-xxxxxxxxxxxx`,
	Run: func(cmd *cobra.Command, _ []string) { _ = cmd.Help() },
}

func init() {
	sandboxCmd.AddCommand(sandboxInjectionRuleCmd)
}
