package commands

import (
	"github.com/spf13/cobra"

	"github.com/sufy-dev/sufy/cmd/internal/sandbox"
)

// sandboxKillCmd corresponds to cmd/sufy/sandbox_kill_cmd.gox.
var sandboxKillCmd = func() *cobra.Command {
	var (
		all      bool
		state    string
		metadata string
	)
	cmd := &cobra.Command{
		Use:     "kill [sandboxIDs...]",
		Short:   "Kill one or more sandboxes (alias: kl)",
		Aliases: []string{"kl"},
		Example: `  # Kill a single sandbox
  sufy sandbox kill sb-xxxxxxxxxxxx
  sufy sbx kl sb-xxxxxxxxxxxx

  # Kill multiple sandboxes
  sufy sandbox kill sb-aaa sb-bbb sb-ccc
  sufy sbx kl sb-aaa sb-bbb sb-ccc

  # Kill all running sandboxes
  sufy sandbox kill --all
  sufy sbx kl -a

  # Kill all paused sandboxes with specific metadata
  sufy sandbox kill --all -s paused -m env=dev
  sufy sbx kl -a -s paused -m env=dev`,
		Run: func(_ *cobra.Command, args []string) {
			sandbox.KillBatch(sandbox.KillInfo{
				SandboxIDs: args,
				All:        all,
				State:      state,
				Metadata:   metadata,
			})
		},
	}
	f := cmd.Flags()
	f.BoolVarP(&all, "all", "a", false, "kill all sandboxes")
	f.StringVarP(&state, "state", "s", "", "filter by state when using --all (comma-separated: running,paused). Defaults to running")
	f.StringVarP(&metadata, "metadata", "m", "", "filter by metadata when using --all (key1=value1,key2=value2)")
	return cmd
}()

func init() {
	sandboxCmd.AddCommand(sandboxKillCmd)
}
