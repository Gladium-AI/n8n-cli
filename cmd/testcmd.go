package cmd

import (
	"fmt"
	"strings"

	"n8n-cli/internal/config"
	"n8n-cli/internal/output"

	"github.com/spf13/cobra"
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Test and debug workflows",
}

var testRetryCmd = &cobra.Command{
	Use:   "retry <executionId>",
	Short: "Retry a failed execution",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newService()
		if err != nil {
			return err
		}

		result, err := svc.RetryExecution(args[0])
		if err != nil {
			return err
		}

		printResult(cmd,
			fmt.Sprintf("Retry initiated for execution %s.\n", args[0]),
			output.ExecutionSummary(result),
			result,
		)
		return nil
	},
}

var testRunsCmd = &cobra.Command{
	Use:   "runs <workflowId>",
	Short: "List test or recent runs for a workflow",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newService()
		if err != nil {
			return err
		}

		status, _ := cmd.Flags().GetString("status")
		limit, _ := cmd.Flags().GetInt("limit")
		cursor, _ := cmd.Flags().GetString("cursor")

		executions, err := svc.ListExecutions(args[0], status, limit, cursor, false)
		if err != nil {
			return err
		}

		printResult(cmd,
			output.ExecutionListSummary(executions),
			output.ExecutionListSummary(executions),
			executions,
		)
		return nil
	},
}

var testInspectCmd = &cobra.Command{
	Use:   "inspect <executionId>",
	Short: "Inspect a test execution",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newService()
		if err != nil {
			return err
		}

		result, err := svc.GetExecution(args[0], true)
		if err != nil {
			return err
		}

		printResult(cmd,
			output.ExecutionSummary(result),
			output.ExecutionSummary(result),
			result,
		)
		return nil
	},
}

var testWebhookCmd = &cobra.Command{
	Use:   "webhook <workflowId>",
	Short: "Test a webhook-triggered workflow",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newService()
		if err != nil {
			return err
		}

		payloadFile, _ := cmd.Flags().GetString("payload-file")
		stdin, _ := cmd.Flags().GetBool("stdin")
		method, _ := cmd.Flags().GetString("method")
		headerFlags, _ := cmd.Flags().GetStringSlice("header")
		testURL, _ := cmd.Flags().GetBool("test-url")

		headers := make(map[string]string)
		for _, h := range headerFlags {
			parts := strings.SplitN(h, ":", 2)
			if len(parts) == 2 {
				headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}

		useTestURL := testURL || !cmd.Flags().Changed("production-url")

		statusCode, body, err := svc.WebhookTest(args[0], payloadFile, stdin, method, headers, useTestURL)
		if err != nil {
			return err
		}

		if config.IsJSON() {
			result := map[string]interface{}{
				"statusCode": statusCode,
				"body":       string(body),
			}
			fmt.Println(output.JSON(result))
		} else {
			fmt.Printf("Status: %d\n", statusCode)
			if len(body) > 0 {
				fmt.Printf("Response:\n%s\n", string(body))
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(testCmd)

	testCmd.AddCommand(testRetryCmd)

	testRunsCmd.Flags().String("status", "", "filter by status")
	testRunsCmd.Flags().Int("limit", 0, "max results")
	testRunsCmd.Flags().String("cursor", "", "pagination cursor")
	testCmd.AddCommand(testRunsCmd)

	testCmd.AddCommand(testInspectCmd)

	testWebhookCmd.Flags().String("payload-file", "", "JSON payload file")
	testWebhookCmd.Flags().Bool("stdin", false, "read payload from stdin")
	testWebhookCmd.Flags().String("method", "", "HTTP method (default: from webhook node)")
	testWebhookCmd.Flags().StringSlice("header", nil, "headers as k:v")
	testWebhookCmd.Flags().Bool("test-url", true, "use test webhook URL")
	testWebhookCmd.Flags().Bool("production-url", false, "use production webhook URL")
	testCmd.AddCommand(testWebhookCmd)
}
