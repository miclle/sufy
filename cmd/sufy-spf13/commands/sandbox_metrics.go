package commands

import (
	"github.com/spf13/cobra"

	"github.com/sufy-dev/sufy/cmd/internal/sandbox"
)

// sandboxMetricsCmd corresponds to cmd/sufy/sandbox_metrics_cmd.gox.
var sandboxMetricsCmd = func() *cobra.Command {
	var (
		format string
		follow bool
	)
	cmd := &cobra.Command{
		Use:     "metrics <sandboxID>",
		Short:   "View sandbox resource metrics (alias: mt)",
		Aliases: []string{"mt"},
		Example: `  # View current metrics
  sufy sandbox metrics sb-xxxxxxxxxxxx
  sufy sbx mt sb-xxxxxxxxxxxx

  # Stream metrics in follow mode
  sufy sandbox metrics sb-xxxxxxxxxxxx -f
  sufy sbx mt sb-xxxxxxxxxxxx -f

  # Output as JSON
  sufy sandbox metrics sb-xxxxxxxxxxxx --format json
  sufy sbx mt sb-xxxxxxxxxxxx --format json`,
		Run: func(_ *cobra.Command, args []string) {
			if len(args) == 0 {
				sandbox.Metrics(sandbox.MetricsInfo{})
				return
			}
			sandbox.Metrics(sandbox.MetricsInfo{
				SandboxID: args[0],
				Format:    format,
				Follow:    follow,
			})
		},
	}
	f := cmd.Flags()
	f.StringVar(&format, "format", "pretty", "output format: pretty or json")
	f.BoolVarP(&follow, "follow", "f", false, "keep streaming metrics until the sandbox is closed")
	return cmd
}()

func init() {
	sandboxCmd.AddCommand(sandboxMetricsCmd)
}
