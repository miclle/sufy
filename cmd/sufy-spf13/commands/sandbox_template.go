package commands

import "github.com/spf13/cobra"

// sandboxTemplateCmd corresponds to cmd/sufy/sandbox_template_cmd.gox.
var sandboxTemplateCmd = &cobra.Command{
	Use:     "template",
	Short:   "Manage sandbox templates (alias: tpl)",
	Aliases: []string{"tpl"},
	Example: `  # View template subcommands
  sufy sandbox template -h
  sufy sbx tpl -h

  # List all templates
  sufy sandbox template list
  sufy sbx tpl ls

  # Build a new template
  sufy sandbox template build --name my-template --from-image ubuntu:22.04 --wait
  sufy sbx tpl bd --name my-template --from-image ubuntu:22.04 --wait

  # Get template details
  sufy sandbox template get tmpl-xxxxxxxxxxxx
  sufy sbx tpl gt tmpl-xxxxxxxxxxxx`,
	Run: func(cmd *cobra.Command, _ []string) { _ = cmd.Help() },
}

func init() {
	sandboxCmd.AddCommand(sandboxTemplateCmd)
}
