package cmd

import (
	"fmt"

	"n8n-cli/internal/output"
	"n8n-cli/internal/parser"

	"github.com/spf13/cobra"
)

var workflowCmd = &cobra.Command{
	Use:     "workflow",
	Aliases: []string{"wf"},
	Short:   "Manage n8n workflows",
}

var workflowCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a workflow",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newService()
		if err != nil {
			return err
		}

		name, _ := cmd.Flags().GetString("name")
		if name == "" && len(args) > 0 {
			name = args[0]
		}
		file, _ := cmd.Flags().GetString("file")
		stdin, _ := cmd.Flags().GetBool("stdin")
		active, _ := cmd.Flags().GetBool("active")
		tags, _ := cmd.Flags().GetStringSlice("tag")

		result, err := svc.CreateWorkflow(name, file, stdin, active, tags)
		if err != nil {
			return err
		}

		pw, _ := parser.Parse(result)
		if pw != nil {
			printResult(cmd, output.WorkflowSummary(pw.Meta)+"\n", output.InspectSummary(pw), result)
		} else {
			fmt.Println(output.JSON(result))
		}
		return nil
	},
}

var workflowListCmd = &cobra.Command{
	Use:   "list",
	Short: "List workflows",
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newService()
		if err != nil {
			return err
		}

		var active *bool
		if cmd.Flag("active").Changed {
			a, _ := cmd.Flags().GetBool("active")
			active = &a
		}
		name, _ := cmd.Flags().GetString("name")
		tags, _ := cmd.Flags().GetStringSlice("tag")
		limit, _ := cmd.Flags().GetInt("limit")
		cursor, _ := cmd.Flags().GetString("cursor")
		all, _ := cmd.Flags().GetBool("all")

		workflows, err := svc.ListWorkflows(active, tags, name, limit, cursor, all)
		if err != nil {
			return err
		}

		printResult(cmd,
			output.WorkflowListSummary(workflows),
			output.WorkflowListSummary(workflows),
			workflows,
		)
		return nil
	},
}

var workflowGetCmd = &cobra.Command{
	Use:   "get <workflowId>",
	Short: "Get a workflow",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newService()
		if err != nil {
			return err
		}

		result, err := svc.GetWorkflow(args[0])
		if err != nil {
			return err
		}

		pw, _ := parser.Parse(result)
		if pw != nil {
			printResult(cmd, output.WorkflowSummary(pw.Meta)+"\n", output.InspectSummary(pw), result)
		} else {
			fmt.Println(output.JSON(result))
		}
		return nil
	},
}

var workflowUpdateCmd = &cobra.Command{
	Use:   "update <workflowId>",
	Short: "Update a workflow",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newService()
		if err != nil {
			return err
		}

		file, _ := cmd.Flags().GetString("file")
		stdin, _ := cmd.Flags().GetBool("stdin")
		sets, _ := cmd.Flags().GetStringSlice("set")
		patchFile, _ := cmd.Flags().GetString("patch-file")

		result, err := svc.UpdateWorkflow(args[0], file, stdin, sets, patchFile)
		if err != nil {
			return err
		}

		pw, _ := parser.Parse(result)
		if pw != nil {
			printResult(cmd, output.WorkflowSummary(pw.Meta)+"\n", output.InspectSummary(pw), result)
		} else {
			fmt.Println(output.JSON(result))
		}
		return nil
	},
}

var workflowDeleteCmd = &cobra.Command{
	Use:   "delete <workflowId>",
	Short: "Delete a workflow",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newService()
		if err != nil {
			return err
		}

		yes, _ := cmd.Flags().GetBool("yes")
		if !yes {
			if !confirmAction(fmt.Sprintf("Delete workflow %s?", args[0])) {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		if err := svc.DeleteWorkflow(args[0]); err != nil {
			return err
		}

		fmt.Printf("Workflow %s deleted.\n", args[0])
		return nil
	},
}

var workflowActivateCmd = &cobra.Command{
	Use:   "activate <workflowId>",
	Short: "Activate a workflow",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newService()
		if err != nil {
			return err
		}

		result, err := svc.ActivateWorkflow(args[0])
		if err != nil {
			return err
		}

		pw, _ := parser.Parse(result)
		if pw != nil {
			printResult(cmd, output.WorkflowSummary(pw.Meta)+"\n", output.InspectSummary(pw), result)
		} else {
			fmt.Println(output.JSON(result))
		}
		return nil
	},
}

var workflowDeactivateCmd = &cobra.Command{
	Use:   "deactivate <workflowId>",
	Short: "Deactivate a workflow",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newService()
		if err != nil {
			return err
		}

		result, err := svc.DeactivateWorkflow(args[0])
		if err != nil {
			return err
		}

		pw, _ := parser.Parse(result)
		if pw != nil {
			printResult(cmd, output.WorkflowSummary(pw.Meta)+"\n", output.InspectSummary(pw), result)
		} else {
			fmt.Println(output.JSON(result))
		}
		return nil
	},
}

var workflowInspectCmd = &cobra.Command{
	Use:   "inspect <workflowId>",
	Short: "Compact, agent-friendly workflow summary",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newService()
		if err != nil {
			return err
		}

		pw, err := svc.InspectWorkflow(args[0])
		if err != nil {
			return err
		}

		withNodes, _ := cmd.Flags().GetBool("with-nodes")
		withConnections, _ := cmd.Flags().GetBool("with-connections")

		summary := output.InspectSummary(pw)
		resolved := summary
		if withNodes {
			resolved += "\n"
			for _, n := range pw.Nodes {
				resolved += output.NodeResolved(n) + "\n"
			}
		}
		if withConnections && len(pw.Edges) > 0 {
			resolved += "\nConnections:\n" + output.EdgeListSummary(pw.Edges)
		}

		printResult(cmd, summary, resolved, pw.RawWorkflow)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(workflowCmd)

	workflowCreateCmd.Flags().String("file", "", "workflow JSON file")
	workflowCreateCmd.Flags().Bool("stdin", false, "read workflow from stdin")
	workflowCreateCmd.Flags().String("name", "", "workflow name")
	workflowCreateCmd.Flags().Bool("active", false, "activate on creation")
	workflowCreateCmd.Flags().StringSlice("tag", nil, "tags")
	workflowCmd.AddCommand(workflowCreateCmd)

	workflowListCmd.Flags().Bool("active", false, "filter by active status")
	workflowListCmd.Flags().String("name", "", "filter by name pattern")
	workflowListCmd.Flags().StringSlice("tag", nil, "filter by tags")
	workflowListCmd.Flags().Int("limit", 0, "max results per page")
	workflowListCmd.Flags().String("cursor", "", "pagination cursor")
	workflowListCmd.Flags().Bool("all", false, "fetch all pages")
	workflowCmd.AddCommand(workflowListCmd)

	workflowCmd.AddCommand(workflowGetCmd)

	workflowUpdateCmd.Flags().String("file", "", "workflow JSON file")
	workflowUpdateCmd.Flags().Bool("stdin", false, "read from stdin")
	workflowUpdateCmd.Flags().StringSlice("set", nil, "set a path=value")
	workflowUpdateCmd.Flags().String("patch-file", "", "JSON merge patch file")
	workflowCmd.AddCommand(workflowUpdateCmd)

	workflowDeleteCmd.Flags().Bool("yes", false, "skip confirmation")
	workflowDeleteCmd.Flags().Bool("force", false, "force deletion")
	workflowCmd.AddCommand(workflowDeleteCmd)

	workflowCmd.AddCommand(workflowActivateCmd)
	workflowCmd.AddCommand(workflowDeactivateCmd)

	workflowInspectCmd.Flags().Bool("with-nodes", false, "include node details")
	workflowInspectCmd.Flags().Bool("with-connections", false, "include connection details")
	workflowInspectCmd.Flags().Bool("with-settings", false, "include settings")
	workflowInspectCmd.Flags().Bool("with-tags", false, "include tags")
	workflowCmd.AddCommand(workflowInspectCmd)
}
