package commands

import (
	"github.com/spf13/cobra"

	"github.com/sufy-dev/sufy/cmd/internal/sandbox"
)

// sandboxInjectionRuleDeleteCmd corresponds to cmd/sufy/sandbox_injectionrule_delete_cmd.gox.
var sandboxInjectionRuleDeleteCmd = func() *cobra.Command {
	var (
		yes bool
		sel bool
	)
	cmd := &cobra.Command{
		Use:     "delete [ruleIDs...]",
		Short:   "Delete one or more injection rules (alias: dl)",
		Aliases: []string{"dl"},
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
				_ = cmd.Help()
				return
			}
			sandbox.InjectionRuleDelete(args, yes, sel)
		},
	}
	f := cmd.Flags()
	f.BoolVarP(&yes, "yes", "y", false, "skip confirmation")
	f.BoolVarP(&sel, "select", "s", false, "interactively select rules to delete")
	return cmd
}()

func init() {
	sandboxInjectionRuleCmd.AddCommand(sandboxInjectionRuleDeleteCmd)
}
