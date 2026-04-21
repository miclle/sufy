package commands

import (
	"github.com/spf13/cobra"

	"github.com/sufy-dev/sufy/cmd/internal/sandbox"
)

// sandboxTemplatePublishCmd corresponds to cmd/sufy/sandbox_template_publish_cmd.gox.
var sandboxTemplatePublishCmd = func() *cobra.Command {
	var (
		yes bool
		sel bool
	)
	cmd := &cobra.Command{
		Use:     "publish [templateIDs...]",
		Short:   "Publish templates (make public) (alias: pb)",
		Aliases: []string{"pb"},
		Example: `  # Publish a single template (skip confirmation)
  sufy sandbox template publish tmpl-xxxxxxxxxxxx -y
  sufy sbx tpl pb tmpl-xxxxxxxxxxxx -y

  # Interactively select templates to publish
  sufy sandbox template publish -s
  sufy sbx tpl pb -s`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 && !sel {
				_ = cmd.Help()
				return
			}
			sandbox.TemplateSetPublic(args, true, yes, sel, "publish")
		},
	}
	f := cmd.Flags()
	f.BoolVarP(&yes, "yes", "y", false, "skip confirmation")
	f.BoolVarP(&sel, "select", "s", false, "interactively select templates")
	return cmd
}()

func init() {
	sandboxTemplateCmd.AddCommand(sandboxTemplatePublishCmd)
}
