package commands

import (
	"github.com/spf13/cobra"

	"github.com/sufy-dev/sufy/cmd/internal/sandbox"
)

// sandboxTemplateListCmd corresponds to cmd/sufy/sandbox_template_list_cmd.gox.
var sandboxTemplateListCmd = func() *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List sandbox templates (alias: ls)",
		Aliases: []string{"ls"},
		Example: `  # List all templates
  sufy sandbox template list
  sufy sbx tpl ls

  # Output as JSON
  sufy sandbox template list --format json
  sufy sbx tpl ls --format json`,
		Run: func(_ *cobra.Command, _ []string) {
			sandbox.TemplateList(format)
		},
	}
	cmd.Flags().StringVar(&format, "format", "pretty", "output format: pretty or json")
	return cmd
}()

func init() {
	sandboxTemplateCmd.AddCommand(sandboxTemplateListCmd)
}
