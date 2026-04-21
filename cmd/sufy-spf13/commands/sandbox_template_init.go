package commands

import (
	"github.com/spf13/cobra"

	"github.com/sufy-dev/sufy/cmd/internal/sandbox"
)

// sandboxTemplateInitCmd corresponds to cmd/sufy/sandbox_template_init_cmd.gox.
var sandboxTemplateInitCmd = func() *cobra.Command {
	var info sandbox.InitInfo
	cmd := &cobra.Command{
		Use:     "init",
		Short:   "Initialize a new template project (alias: it)",
		Long:    "Scaffold a new template project with boilerplate files for the selected language.",
		Aliases: []string{"it"},
		Example: `  # Interactive mode
  sufy sandbox template init
  sufy sbx tpl it

  # Non-interactive mode
  sufy sandbox template init --name my-template --language go
  sufy sbx tpl it --name my-template --language go

  # Non-interactive mode with custom path
  sufy sandbox template init --name my-api --language typescript --path ./my-api
  sufy sbx tpl it --name my-api --language typescript --path ./my-api`,
		Run: func(_ *cobra.Command, _ []string) {
			sandbox.TemplateInit(info)
		},
	}
	f := cmd.Flags()
	f.StringVar(&info.Name, "name", "", "template project name")
	f.StringVar(&info.Language, "language", "", "programming language (go, typescript, python)")
	f.StringVar(&info.Path, "path", "", "output directory (defaults to ./<name>)")
	return cmd
}()

func init() {
	sandboxTemplateCmd.AddCommand(sandboxTemplateInitCmd)
}
