package commands

import (
	"github.com/spf13/cobra"

	"github.com/sufy-dev/sufy/cmd/internal/sandbox"
)

// sandboxTemplateDeleteCmd corresponds to cmd/sufy/sandbox_template_delete_cmd.gox.
var sandboxTemplateDeleteCmd = func() *cobra.Command {
	var (
		yes bool
		sel bool
	)
	cmd := &cobra.Command{
		Use:     "delete [templateIDs...]",
		Short:   "Delete one or more templates (alias: dl)",
		Aliases: []string{"dl"},
		Example: `  # Delete a single template (skip confirmation)
  sufy sandbox template delete tmpl-xxxxxxxxxxxx -y
  sufy sbx tpl dl tmpl-xxxxxxxxxxxx -y

  # Delete multiple templates
  sufy sandbox template delete tmpl-aaa tmpl-bbb -y
  sufy sbx tpl dl tmpl-aaa tmpl-bbb -y

  # Interactively select templates to delete
  sufy sandbox template delete -s
  sufy sbx tpl dl -s`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 && !sel {
				_ = cmd.Help()
				return
			}
			sandbox.TemplateDelete(args, yes, sel)
		},
	}
	f := cmd.Flags()
	f.BoolVarP(&yes, "yes", "y", false, "skip confirmation")
	f.BoolVarP(&sel, "select", "s", false, "interactively select templates to delete")
	return cmd
}()

func init() {
	sandboxTemplateCmd.AddCommand(sandboxTemplateDeleteCmd)
}
