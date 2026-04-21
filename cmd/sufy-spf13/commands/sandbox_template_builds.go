package commands

import (
	"github.com/spf13/cobra"

	"github.com/sufy-dev/sufy/cmd/internal/sandbox"
)

// sandboxTemplateBuildsCmd corresponds to cmd/sufy/sandbox_template_builds_cmd.gox.
var sandboxTemplateBuildsCmd = &cobra.Command{
	Use:     "builds <templateID> <buildID>",
	Short:   "View template build status (alias: bds)",
	Aliases: []string{"bds"},
	Args:    cobra.ExactArgs(2),
	Example: `  # View build status
  sufy sandbox template builds tmpl-xxxxxxxxxxxx build-xxxxxxxxxxxx
  sufy sbx tpl bds tmpl-xxxxxxxxxxxx build-xxxxxxxxxxxx`,
	Run: func(_ *cobra.Command, args []string) {
		sandbox.TemplateBuilds(args[0], args[1])
	},
}

func init() {
	sandboxTemplateCmd.AddCommand(sandboxTemplateBuildsCmd)
}
