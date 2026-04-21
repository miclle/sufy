package commands

import (
	"github.com/spf13/cobra"

	"github.com/sufy-dev/sufy/cmd/internal/sandbox"
)

// sandboxInjectionRuleGetCmd corresponds to cmd/sufy/sandbox_injectionrule_get_cmd.gox.
var sandboxInjectionRuleGetCmd = &cobra.Command{
	Use:     "get <ruleID>",
	Short:   "Get injection rule details (alias: gt)",
	Aliases: []string{"gt"},
	Args:    cobra.ExactArgs(1),
	Example: `  # Get injection rule details
  sufy sandbox injection-rule get rule-xxxxxxxxxxxx
  sufy sbx ir gt rule-xxxxxxxxxxxx`,
	Run: func(_ *cobra.Command, args []string) {
		sandbox.InjectionRuleGet(args[0])
	},
}

func init() {
	sandboxInjectionRuleCmd.AddCommand(sandboxInjectionRuleGetCmd)
}
