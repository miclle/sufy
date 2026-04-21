package commands

import (
	"github.com/spf13/cobra"

	"github.com/sufy-dev/sufy/cmd/internal/sandbox"
)

// sandboxTemplateUnpublishCmd corresponds to cmd/sufy/sandbox_template_unpublish_cmd.gox.
var sandboxTemplateUnpublishCmd = func() *cobra.Command {
	var (
		yes bool
		sel bool
	)
	cmd := &cobra.Command{
		Use:     "unpublish [templateIDs...]",
		Short:   "Unpublish templates (make private) (alias: upb)",
		Aliases: []string{"upb"},
		Example: `  # Unpublish a single template (skip confirmation)
  sufy sandbox template unpublish tmpl-xxxxxxxxxxxx -y
  sufy sbx tpl upb tmpl-xxxxxxxxxxxx -y

  # Interactively select templates to unpublish
  sufy sandbox template unpublish -s
  sufy sbx tpl upb -s`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 && !sel {
				_ = cmd.Help()
				return
			}
			sandbox.TemplateSetPublic(args, false, yes, sel, "unpublish")
		},
	}
	f := cmd.Flags()
	f.BoolVarP(&yes, "yes", "y", false, "skip confirmation")
	f.BoolVarP(&sel, "select", "s", false, "interactively select templates")
	return cmd
}()

func init() {
	sandboxTemplateCmd.AddCommand(sandboxTemplateUnpublishCmd)
}
