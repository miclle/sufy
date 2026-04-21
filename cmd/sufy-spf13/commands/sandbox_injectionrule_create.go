package commands

import (
	"github.com/spf13/cobra"

	"github.com/sufy-dev/sufy/cmd/internal/sandbox"
)

// sandboxInjectionRuleCreateCmd corresponds to cmd/sufy/sandbox_injectionrule_create_cmd.gox.
var sandboxInjectionRuleCreateCmd = func() *cobra.Command {
	var (
		name, typ, apiKey, baseURL, headers string
	)
	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create an injection rule (alias: cr)",
		Aliases: []string{"cr"},
		Example: `  # Create an OpenAI injection rule
  sufy sandbox injection-rule create --name openai-default --type openai --api-key sk-xxx
  sufy sbx ir cr --name openai-default --type openai --api-key sk-xxx

  # Create an Anthropic injection rule with custom base URL
  sufy sandbox injection-rule create --name anthropic-proxy --type anthropic --api-key sk-ant --base-url https://anthropic-proxy.example.com

  # Create a custom HTTP injection rule
  sufy sandbox injection-rule create --name api-auth --type http --base-url https://api.example.com --headers "Authorization=Bearer token123,X-Env=prod"`,
		Run: func(_ *cobra.Command, _ []string) {
			sandbox.InjectionRuleCreate(name, typ, apiKey, baseURL, headers)
		},
	}
	f := cmd.Flags()
	f.StringVar(&name, "name", "", "rule name (required, unique per user)")
	f.StringVar(&typ, "type", "", "injection type: openai, anthropic, gemini, http")
	f.StringVar(&apiKey, "api-key", "", "API key for openai/anthropic/gemini injection types (warning: passing secrets via CLI may leak through shell history or process lists)")
	f.StringVar(&baseURL, "base-url", "", "override base URL or target base URL for http injection")
	f.StringVar(&headers, "headers", "", "HTTP headers for custom http injection (comma-separated key=value pairs)")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("type")
	return cmd
}()

func init() {
	sandboxInjectionRuleCmd.AddCommand(sandboxInjectionRuleCreateCmd)
}
