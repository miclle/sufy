package commands

import (
	"github.com/spf13/cobra"

	"github.com/sufy-dev/sufy/cmd/internal/sandbox"
)

// sandboxPauseCmd corresponds to cmd/sufy/sandbox_pause_cmd.gox.
var sandboxPauseCmd = func() *cobra.Command {
	var (
		all      bool
		state    string
		metadata string
	)
	cmd := &cobra.Command{
		Use:     "pause [sandboxIDs...]",
		Short:   "Pause one or more sandboxes (alias: ps)",
		Aliases: []string{"ps"},
		Example: `  # Pause a single sandbox
  sufy sandbox pause sb-xxxxxxxxxxxx
  sufy sbx ps sb-xxxxxxxxxxxx

  # Pause multiple sandboxes
  sufy sandbox pause sb-aaa sb-bbb sb-ccc
  sufy sbx ps sb-aaa sb-bbb sb-ccc

  # Pause all running sandboxes
  sufy sandbox pause --all
  sufy sbx ps -a

  # Pause all with specific metadata
  sufy sandbox pause --all -m env=dev
  sufy sbx ps -a -m env=dev`,
		Run: func(_ *cobra.Command, args []string) {
			sandbox.PauseBatch(sandbox.PauseInfo{
				SandboxIDs: args,
				All:        all,
				State:      state,
				Metadata:   metadata,
			})
		},
	}
	f := cmd.Flags()
	f.BoolVarP(&all, "all", "a", false, "pause all sandboxes")
	f.StringVarP(&state, "state", "s", "", "filter by state when using --all (comma-separated: running,paused). Defaults to running")
	f.StringVarP(&metadata, "metadata", "m", "", "filter by metadata when using --all (key1=value1,key2=value2)")
	return cmd
}()

func init() {
	sandboxCmd.AddCommand(sandboxPauseCmd)
}
