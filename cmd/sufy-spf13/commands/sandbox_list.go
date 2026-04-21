package commands

import (
	"github.com/spf13/cobra"

	"github.com/sufy-dev/sufy/cmd/internal/sandbox"
)

// sandboxListCmd corresponds to cmd/sufy/sandbox_list_cmd.gox.
var sandboxListCmd = func() *cobra.Command {
	var (
		state    string
		metadata string
		format   string
		limit    int32
	)
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List sandboxes (alias: ls)",
		Aliases: []string{"ls"},
		Example: `  # List running sandboxes
  sufy sandbox list
  sufy sbx ls

  # Filter by state and limit
  sufy sandbox list --state running,paused --limit 10
  sufy sbx ls -s running,paused -l 10

  # Filter by metadata
  sufy sandbox list -m env=prod,team=backend
  sufy sbx ls -m env=prod,team=backend

  # Output as JSON
  sufy sandbox list -f json
  sufy sbx ls -f json`,
		Run: func(_ *cobra.Command, _ []string) {
			sandbox.List(sandbox.ListInfo{
				State:    state,
				Metadata: metadata,
				Limit:    limit,
				Format:   format,
			})
		},
	}
	f := cmd.Flags()
	f.StringVarP(&state, "state", "s", "", "filter by state (comma-separated: running,paused). Defaults to running")
	f.StringVarP(&metadata, "metadata", "m", "", "filter by metadata (key1=value1,key2=value2)")
	f.StringVarP(&format, "format", "f", "pretty", "output format: pretty or json")
	f.Int32VarP(&limit, "limit", "l", 0, "maximum number of sandboxes to return")
	return cmd
}()

func init() {
	sandboxCmd.AddCommand(sandboxListCmd)
}
