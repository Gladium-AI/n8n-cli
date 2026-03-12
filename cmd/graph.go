package cmd

import (
	"n8n-cli/internal/output"

	"github.com/spf13/cobra"
)

var graphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Analyze workflow graph structure",
}

var graphInspectCmd = &cobra.Command{
	Use:   "inspect <workflowId>",
	Short: "Summarize graph structure for an agent",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newService()
		if err != nil {
			return err
		}

		pw, analysis, err := svc.InspectGraph(args[0])
		if err != nil {
			return err
		}

		summary := output.GraphSummary(analysis)

		withAdjacency, _ := cmd.Flags().GetBool("with-adjacency")
		resolved := summary
		if withAdjacency && len(analysis.AdjacencyList) > 0 {
			resolved += "\nAdjacency list:\n"
			for from, tos := range analysis.AdjacencyList {
				if len(tos) > 0 {
					fromNode := pw.Indexes.ByRef[from]
					fromName := from
					if fromNode != nil {
						fromName = fromNode.Name
					}
					resolved += "  " + fromName + " -> "
					for i, to := range tos {
						toNode := pw.Indexes.ByRef[to]
						toName := to
						if toNode != nil {
							toName = toNode.Name
						}
						if i > 0 {
							resolved += ", "
						}
						resolved += toName
					}
					resolved += "\n"
				}
			}
		}

		rawResult := map[string]interface{}{
			"analysis": analysis,
			"workflow": pw.Meta,
		}
		printResult(cmd, summary, resolved, rawResult)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(graphCmd)

	graphInspectCmd.Flags().Bool("with-adjacency", false, "include adjacency list")
	graphInspectCmd.Flags().Bool("with-orphans", false, "include orphan detection")
	graphInspectCmd.Flags().Bool("with-cycles", false, "include cycle detection")
	graphCmd.AddCommand(graphInspectCmd)
}
