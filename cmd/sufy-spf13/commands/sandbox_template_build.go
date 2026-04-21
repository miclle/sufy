package commands

import (
	"github.com/spf13/cobra"

	"github.com/sufy-dev/sufy/cmd/internal/sandbox"
)

// sandboxTemplateBuildCmd corresponds to cmd/sufy/sandbox_template_build_cmd.gox.
var sandboxTemplateBuildCmd = func() *cobra.Command {
	var info sandbox.BuildInfo
	cmd := &cobra.Command{
		Use:     "build",
		Short:   "Build a template (alias: bd)",
		Aliases: []string{"bd"},
		Long: `Create a new template and build it, or rebuild an existing template.

Supports three build modes:
  1. --from-image: Build from a base Docker image
  2. --from-template: Build from an existing template
  3. --dockerfile: Build from a Dockerfile (v2 build system)`,
		Example: `  # Create and build a new template from a Docker image
  sufy sandbox template build --name my-template --from-image ubuntu:22.04 --wait
  sufy sbx tpl bd --name my-template --from-image ubuntu:22.04 --wait

  # Build from a Dockerfile
  sufy sandbox template build --name my-template --dockerfile ./Dockerfile --wait
  sufy sbx tpl bd --name my-template --dockerfile ./Dockerfile --wait

  # Build from a Dockerfile with a custom context directory
  sufy sandbox template build --name my-template --dockerfile ./Dockerfile --path ./context --wait
  sufy sbx tpl bd --name my-template --dockerfile ./Dockerfile --path ./context --wait

  # Rebuild an existing template
  sufy sandbox template build --template-id tmpl-xxxxxxxxxxxx --from-image ubuntu:22.04
  sufy sbx tpl bd --template-id tmpl-xxxxxxxxxxxx --from-image ubuntu:22.04

  # Force rebuild without cache
  sufy sandbox template build --template-id tmpl-xxxxxxxxxxxx --no-cache --wait
  sufy sbx tpl bd --template-id tmpl-xxxxxxxxxxxx --no-cache --wait`,
		Run: func(_ *cobra.Command, _ []string) {
			sandbox.TemplateBuild(info)
		},
	}
	f := cmd.Flags()
	f.StringVar(&info.Name, "name", "", "template name (for creating a new template)")
	f.StringVar(&info.TemplateID, "template-id", "", "existing template ID (for rebuilding)")
	f.StringVar(&info.FromImage, "from-image", "", "base Docker image")
	f.StringVar(&info.FromTemplate, "from-template", "", "base template")
	f.StringVar(&info.StartCmd, "start-cmd", "", "command to run after build")
	f.StringVar(&info.ReadyCmd, "ready-cmd", "", "readiness check command")
	f.Int32Var(&info.CPUCount, "cpu", 0, "sandbox CPU count")
	f.Int32Var(&info.MemoryMB, "memory", 0, "sandbox memory size in MiB")
	f.BoolVar(&info.Wait, "wait", false, "wait for build to complete")
	f.BoolVar(&info.NoCache, "no-cache", false, "force full rebuild ignoring cache")
	f.StringVar(&info.Dockerfile, "dockerfile", "", "path to Dockerfile (enables v2 build)")
	f.StringVar(&info.Path, "path", "", "build context directory (defaults to Dockerfile's parent)")
	return cmd
}()

func init() {
	sandboxTemplateCmd.AddCommand(sandboxTemplateBuildCmd)
}
