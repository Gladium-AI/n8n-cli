package cmd

import (
	"fmt"

	"n8n-cli/internal/config"
	"n8n-cli/internal/output"
	"n8n-cli/internal/parser"

	"github.com/spf13/cobra"
)

var connectionCmd = &cobra.Command{
	Use:     "connection",
	Aliases: []string{"conn"},
	Short:   "Manage workflow connections",
}

var connectionListCmd = &cobra.Command{
	Use:   "list <workflowId>",
	Short: "List connections in a workflow",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newService()
		if err != nil {
			return err
		}

		nodeRef, _ := cmd.Flags().GetString("node")
		direction, _ := cmd.Flags().GetString("direction")

		edges, err := svc.ListConnections(args[0], nodeRef, direction)
		if err != nil {
			return err
		}

		printResult(cmd,
			output.EdgeListSummary(edges),
			output.EdgeListSummary(edges),
			edges,
		)
		return nil
	},
}

var connectionCreateCmd = &cobra.Command{
	Use:   "create <workflowId>",
	Short: "Create a connection between two nodes",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newService()
		if err != nil {
			return err
		}

		from, _ := cmd.Flags().GetString("from")
		fromOutput, _ := cmd.Flags().GetInt("from-output")
		to, _ := cmd.Flags().GetString("to")
		toInput, _ := cmd.Flags().GetInt("to-input")

		if from == "" || to == "" {
			return fmt.Errorf("--from and --to are required")
		}

		input := parser.EdgeInput{
			FromRef:    from,
			FromOutput: fromOutput,
			ToRef:      to,
			ToInput:    toInput,
		}

		edge, err := svc.CreateConnection(args[0], input, config.IsDryRun())
		if err != nil {
			return err
		}

		prefix := ""
		if config.IsDryRun() {
			prefix = "[dry-run] "
		}
		fmt.Printf("%sConnection created: %s[%d] -> %s[%d]\n",
			prefix, edge.FromName, edge.FromOutput, edge.ToName, edge.ToInput)
		return nil
	},
}

var connectionDeleteCmd = &cobra.Command{
	Use:   "delete <workflowId>",
	Short: "Delete a connection",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newService()
		if err != nil {
			return err
		}

		from, _ := cmd.Flags().GetString("from")
		fromOutput, _ := cmd.Flags().GetInt("from-output")
		to, _ := cmd.Flags().GetString("to")
		toInput, _ := cmd.Flags().GetInt("to-input")

		if from == "" || to == "" {
			return fmt.Errorf("--from and --to are required")
		}

		input := parser.EdgeInput{
			FromRef:    from,
			FromOutput: fromOutput,
			ToRef:      to,
			ToInput:    toInput,
		}

		if err := svc.DeleteConnection(args[0], input, config.IsDryRun()); err != nil {
			return err
		}

		prefix := ""
		if config.IsDryRun() {
			prefix = "[dry-run] "
		}
		fmt.Printf("%sConnection deleted: %s[%d] -> %s[%d]\n",
			prefix, from, fromOutput, to, toInput)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(connectionCmd)

	connectionListCmd.Flags().String("node", "", "filter by node ref")
	connectionListCmd.Flags().String("direction", "both", "direction: in, out, both")
	connectionCmd.AddCommand(connectionListCmd)

	connectionCreateCmd.Flags().String("from", "", "source node ref")
	connectionCreateCmd.Flags().Int("from-output", 0, "source output index")
	connectionCreateCmd.Flags().String("to", "", "target node ref")
	connectionCreateCmd.Flags().Int("to-input", 0, "target input index")
	connectionCmd.AddCommand(connectionCreateCmd)

	connectionDeleteCmd.Flags().String("from", "", "source node ref")
	connectionDeleteCmd.Flags().Int("from-output", 0, "source output index")
	connectionDeleteCmd.Flags().String("to", "", "target node ref")
	connectionDeleteCmd.Flags().Int("to-input", 0, "target input index")
	connectionCmd.AddCommand(connectionDeleteCmd)
}
