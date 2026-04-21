package commands

import (
	"github.com/spf13/cobra"

	"github.com/sufy-dev/sufy/cmd/internal/sandbox"
)

// sandboxInjectionRuleListCmd corresponds to cmd/sufy/sandbox_injectionrule_list_cmd.gox.
var sandboxInjectionRuleListCmd = func() *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List injection rules (alias: ls)",
		Aliases: []string{"ls"},
		Example: `  # List all injection rules
  sufy sandbox injection-rule list
  sufy sbx ir ls

  # Output as JSON
  sufy sandbox injection-rule list --format json
  sufy sbx ir ls --format json`,
		Run: func(_ *cobra.Command, _ []string) {
			sandbox.InjectionRuleList(format)
		},
	}
	cmd.Flags().StringVar(&format, "format", "pretty", "output format: pretty or json")
	return cmd
}()

func init() {
	sandboxInjectionRuleCmd.AddCommand(sandboxInjectionRuleListCmd)
}
