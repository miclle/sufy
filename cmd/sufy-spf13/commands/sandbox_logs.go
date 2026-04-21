package commands

import (
	"github.com/spf13/cobra"

	"github.com/sufy-dev/sufy/cmd/internal/sandbox"
)

// sandboxLogsCmd corresponds to cmd/sufy/sandbox_logs_cmd.gox.
var sandboxLogsCmd = func() *cobra.Command {
	var (
		level   string
		format  string
		follow  bool
		loggers string
		limit   int32
	)
	cmd := &cobra.Command{
		Use:     "logs <sandboxID>",
		Short:   "View sandbox logs (alias: lg)",
		Aliases: []string{"lg"},
		Example: `  # View logs
  sufy sandbox logs sb-xxxxxxxxxxxx
  sufy sbx lg sb-xxxxxxxxxxxx

  # Filter by level (WARN and above)
  sufy sandbox logs sb-xxxxxxxxxxxx --level WARN
  sufy sbx lg sb-xxxxxxxxxxxx --level WARN

  # Stream logs in follow mode
  sufy sandbox logs sb-xxxxxxxxxxxx -f
  sufy sbx lg sb-xxxxxxxxxxxx -f

  # Filter by logger prefix
  sufy sandbox logs sb-xxxxxxxxxxxx --loggers envd,process
  sufy sbx lg sb-xxxxxxxxxxxx --loggers envd,process

  # Output as JSON
  sufy sandbox logs sb-xxxxxxxxxxxx --format json
  sufy sbx lg sb-xxxxxxxxxxxx --format json`,
		Run: func(_ *cobra.Command, args []string) {
			if len(args) == 0 {
				sandbox.Logs(sandbox.LogsInfo{})
				return
			}
			sandbox.Logs(sandbox.LogsInfo{
				SandboxID: args[0],
				Level:     level,
				Limit:     limit,
				Format:    format,
				Follow:    follow,
				Loggers:   loggers,
			})
		},
	}
	f := cmd.Flags()
	f.StringVar(&level, "level", "INFO", "filter by log level (DEBUG, INFO, WARN, ERROR). Higher levels are also shown")
	f.StringVar(&format, "format", "pretty", "output format: pretty or json")
	f.BoolVarP(&follow, "follow", "f", false, "keep streaming logs until the sandbox is closed")
	f.StringVar(&loggers, "loggers", "", "filter logs by loggers (comma-separated prefixes)")
	f.Int32Var(&limit, "limit", 0, "maximum number of log entries to return")
	return cmd
}()

func init() {
	sandboxCmd.AddCommand(sandboxLogsCmd)
}
