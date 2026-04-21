package commands

import (
	"github.com/spf13/cobra"

	"github.com/sufy-dev/sufy/cmd/internal/sandbox"
)

// sandboxResumeCmd corresponds to cmd/sufy/sandbox_resume_cmd.gox.
var sandboxResumeCmd = func() *cobra.Command {
	var (
		all      bool
		metadata string
	)
	cmd := &cobra.Command{
		Use:     "resume [sandboxIDs...]",
		Short:   "Resume one or more paused sandboxes (alias: rs)",
		Aliases: []string{"rs"},
		Example: `  # Resume a paused sandbox
  sufy sandbox resume sb-xxxxxxxxxxxx
  sufy sbx rs sb-xxxxxxxxxxxx

  # Resume multiple sandboxes
  sufy sandbox resume sb-aaa sb-bbb sb-ccc
  sufy sbx rs sb-aaa sb-bbb sb-ccc

  # Resume all paused sandboxes
  sufy sandbox resume --all
  sufy sbx rs -a

  # Resume all with specific metadata
  sufy sandbox resume --all -m env=staging
  sufy sbx rs -a -m env=staging`,
		Run: func(_ *cobra.Command, args []string) {
			sandbox.ResumeBatch(sandbox.ResumeInfo{
				SandboxIDs: args,
				All:        all,
				Metadata:   metadata,
			})
		},
	}
	f := cmd.Flags()
	f.BoolVarP(&all, "all", "a", false, "resume all paused sandboxes")
	f.StringVarP(&metadata, "metadata", "m", "", "filter by metadata when using --all (key1=value1,key2=value2)")
	return cmd
}()

func init() {
	sandboxCmd.AddCommand(sandboxResumeCmd)
}
