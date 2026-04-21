package commands

import (
	"github.com/spf13/cobra"

	"github.com/sufy-dev/sufy/cmd/internal/sandbox"
)

// sandboxTemplateGetCmd corresponds to cmd/sufy/sandbox_template_get_cmd.gox.
var sandboxTemplateGetCmd = &cobra.Command{
	Use:     "get <templateID>",
	Short:   "Get template details (alias: gt)",
	Aliases: []string{"gt"},
	Args:    cobra.ExactArgs(1),
	Example: `  # Get template details
  sufy sandbox template get tmpl-xxxxxxxxxxxx
  sufy sbx tpl gt tmpl-xxxxxxxxxxxx`,
	Run: func(_ *cobra.Command, args []string) {
		sandbox.TemplateGet(args[0])
	},
}

func init() {
	sandboxTemplateCmd.AddCommand(sandboxTemplateGetCmd)
}
