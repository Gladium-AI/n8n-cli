package cmd

import (
	"fmt"
	"os"

	"n8n-cli/internal/client"
	"n8n-cli/internal/config"
	"n8n-cli/internal/output"
	"n8n-cli/internal/service"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "n8n-cli",
	Short: "AI-agent-friendly CLI for n8n workflow automation",
	Long:  "n8n-cli gives AI agents programmatic, node-level control over n8n workflows through a parser-backed abstraction layer on top of the n8n REST API.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		config.Init()
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	pf := rootCmd.PersistentFlags()
	pf.String("base-url", "", "n8n instance base URL (env: N8N_BASE_URL)")
	pf.String("api-key", "", "n8n API key (env: N8N_API_KEY)")
	pf.String("output", "summary", "output mode: summary, resolved, raw")
	pf.Bool("json", false, "force JSON output")
	pf.Bool("yaml", false, "force YAML output")
	pf.Bool("dry-run", false, "preview changes without applying")
	pf.Bool("quiet", false, "suppress non-essential output")
	pf.Bool("no-color", false, "disable color output")

	viper.BindPFlag(config.KeyBaseURL, pf.Lookup("base-url"))
	viper.BindPFlag(config.KeyAPIKey, pf.Lookup("api-key"))
	viper.BindPFlag(config.KeyOutput, pf.Lookup("output"))
	viper.BindPFlag(config.KeyJSON, pf.Lookup("json"))
	viper.BindPFlag(config.KeyYAML, pf.Lookup("yaml"))
	viper.BindPFlag(config.KeyDryRun, pf.Lookup("dry-run"))
	viper.BindPFlag(config.KeyQuiet, pf.Lookup("quiet"))
	viper.BindPFlag(config.KeyNoColor, pf.Lookup("no-color"))
}

func newService() (*service.Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	c := client.New(config.BaseURL(), config.APIKey())
	return service.New(c), nil
}

func outputMode(cmd *cobra.Command) output.Mode {
	if f := cmd.Flag("output"); f != nil && f.Changed {
		return output.ParseMode(f.Value.String())
	}
	return output.ParseMode(config.Output())
}

func printResult(cmd *cobra.Command, summary, resolved string, raw interface{}) {
	if config.IsJSON() {
		fmt.Println(output.JSON(raw))
		return
	}
	mode := outputMode(cmd)
	switch mode {
	case output.ModeRaw:
		fmt.Println(output.JSON(raw))
	case output.ModeResolved:
		fmt.Print(resolved)
	default:
		fmt.Print(summary)
	}
}

func confirmAction(prompt string) bool {
	fmt.Fprintf(os.Stderr, "%s [y/N]: ", prompt)
	var answer string
	fmt.Scanln(&answer)
	return answer == "y" || answer == "Y" || answer == "yes"
}
