package cmd

import (
	"fmt"

	"n8n-cli/internal/output"

	"github.com/spf13/cobra"
)

var executionCmd = &cobra.Command{
	Use:     "execution",
	Aliases: []string{"exec"},
	Short:   "Manage n8n executions",
}

var executionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List executions",
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newService()
		if err != nil {
			return err
		}

		workflowID, _ := cmd.Flags().GetString("workflow-id")
		status, _ := cmd.Flags().GetString("status")
		limit, _ := cmd.Flags().GetInt("limit")
		cursor, _ := cmd.Flags().GetString("cursor")
		all, _ := cmd.Flags().GetBool("all")

		executions, err := svc.ListExecutions(workflowID, status, limit, cursor, all)
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

var executionGetCmd = &cobra.Command{
	Use:   "get <executionId>",
	Short: "Get execution details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newService()
		if err != nil {
			return err
		}

		withData, _ := cmd.Flags().GetBool("with-data")
		result, err := svc.GetExecution(args[0], withData)
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

var executionDeleteCmd = &cobra.Command{
	Use:   "delete <executionId>",
	Short: "Delete an execution",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newService()
		if err != nil {
			return err
		}

		yes, _ := cmd.Flags().GetBool("yes")
		if !yes {
			if !confirmAction(fmt.Sprintf("Delete execution %s?", args[0])) {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		if err := svc.DeleteExecution(args[0]); err != nil {
			return err
		}

		fmt.Printf("Execution %s deleted.\n", args[0])
		return nil
	},
}

var executionRetryCmd = &cobra.Command{
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
			fmt.Sprintf("Execution %s retry initiated.\n", args[0]),
			output.ExecutionSummary(result),
			result,
		)
		return nil
	},
}

var executionStopCmd = &cobra.Command{
	Use:   "stop <executionId>",
	Short: "Stop a running execution",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newService()
		if err != nil {
			return err
		}

		result, err := svc.StopExecution(args[0])
		if err != nil {
			return err
		}

		printResult(cmd,
			fmt.Sprintf("Execution %s stopped.\n", args[0]),
			output.ExecutionSummary(result),
			result,
		)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(executionCmd)

	executionListCmd.Flags().String("workflow-id", "", "filter by workflow ID")
	executionListCmd.Flags().String("status", "", "filter by status")
	executionListCmd.Flags().Int("limit", 0, "max results per page")
	executionListCmd.Flags().String("cursor", "", "pagination cursor")
	executionListCmd.Flags().Bool("all", false, "fetch all pages")
	executionCmd.AddCommand(executionListCmd)

	executionGetCmd.Flags().Bool("with-data", false, "include execution data")
	executionCmd.AddCommand(executionGetCmd)

	executionDeleteCmd.Flags().Bool("yes", false, "skip confirmation")
	executionCmd.AddCommand(executionDeleteCmd)

	executionCmd.AddCommand(executionRetryCmd)
	executionCmd.AddCommand(executionStopCmd)
}
