package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"n8n-cli/internal/config"
	"n8n-cli/internal/output"
	"n8n-cli/internal/parser"

	"github.com/spf13/cobra"
)

var nodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Manage individual workflow nodes",
}

var nodeListCmd = &cobra.Command{
	Use:   "list <workflowId>",
	Short: "List nodes in a workflow",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newService()
		if err != nil {
			return err
		}

		nodes, err := svc.ListNodes(args[0])
		if err != nil {
			return err
		}

		typeFilter, _ := cmd.Flags().GetString("type")
		nameFilter, _ := cmd.Flags().GetString("name")

		var filtered []*parser.ParsedNode
		for _, n := range nodes {
			if typeFilter != "" && !strings.Contains(n.Type, typeFilter) {
				continue
			}
			if nameFilter != "" && !strings.Contains(strings.ToLower(n.Name), strings.ToLower(nameFilter)) {
				continue
			}
			filtered = append(filtered, n)
		}

		printResult(cmd,
			output.NodeListSummary(filtered),
			output.NodeListSummary(filtered),
			filtered,
		)
		return nil
	},
}

var nodeGetCmd = &cobra.Command{
	Use:   "get <workflowId> <nodeRef>",
	Short: "Inspect one node",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newService()
		if err != nil {
			return err
		}

		node, err := svc.GetNode(args[0], args[1])
		if err != nil {
			return err
		}

		printResult(cmd,
			output.NodeSummary(node),
			output.NodeResolved(node),
			node,
		)
		return nil
	},
}

var nodeCreateCmd = &cobra.Command{
	Use:   "create <workflowId>",
	Short: "Create a node in a workflow",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newService()
		if err != nil {
			return err
		}

		input := parser.NodeInput{}

		file, _ := cmd.Flags().GetString("file")
		stdin, _ := cmd.Flags().GetBool("stdin")

		if file != "" {
			data, err := os.ReadFile(file)
			if err != nil {
				return fmt.Errorf("read file: %w", err)
			}
			var raw map[string]interface{}
			if err := json.Unmarshal(data, &raw); err != nil {
				return fmt.Errorf("parse file: %w", err)
			}
			input = nodeInputFromRaw(raw)
		} else if stdin {
			var raw map[string]interface{}
			if err := json.NewDecoder(os.Stdin).Decode(&raw); err != nil {
				return fmt.Errorf("parse stdin: %w", err)
			}
			input = nodeInputFromRaw(raw)
		}

		if name, _ := cmd.Flags().GetString("name"); name != "" {
			input.Name = name
		}
		if typ, _ := cmd.Flags().GetString("type"); typ != "" {
			input.Type = typ
		}
		if tv, _ := cmd.Flags().GetInt("type-version"); tv > 0 {
			input.TypeVersion = tv
		}
		if pos, _ := cmd.Flags().GetString("position"); pos != "" {
			fmt.Sscanf(pos, "%f,%f", &input.Position[0], &input.Position[1])
		}
		if disabled, _ := cmd.Flags().GetBool("disabled"); disabled {
			input.Disabled = true
		}

		if paramsFile, _ := cmd.Flags().GetString("params-file"); paramsFile != "" {
			data, err := os.ReadFile(paramsFile)
			if err != nil {
				return fmt.Errorf("read params file: %w", err)
			}
			var params map[string]interface{}
			if err := json.Unmarshal(data, &params); err != nil {
				return fmt.Errorf("parse params file: %w", err)
			}
			input.Parameters = params
		}

		if credsFile, _ := cmd.Flags().GetString("credentials-file"); credsFile != "" {
			data, err := os.ReadFile(credsFile)
			if err != nil {
				return fmt.Errorf("read credentials file: %w", err)
			}
			var creds map[string]interface{}
			if err := json.Unmarshal(data, &creds); err != nil {
				return fmt.Errorf("parse credentials file: %w", err)
			}
			input.Credentials = creds
		}

		connectFrom, _ := cmd.Flags().GetString("connect-from")
		connectTo, _ := cmd.Flags().GetString("connect-to")

		if config.IsDryRun() {
			node, err := svc.CreateNodeDryRun(args[0], input, connectFrom, connectTo)
			if err != nil {
				return err
			}
			fmt.Println("[dry-run] Would create node:")
			fmt.Print(output.NodeSummary(node))
			return nil
		}

		node, _, err := svc.CreateNode(args[0], input, connectFrom, connectTo)
		if err != nil {
			return err
		}

		printResult(cmd,
			output.NodeSummary(node),
			output.NodeResolved(node),
			node,
		)
		return nil
	},
}

var nodeUpdateCmd = &cobra.Command{
	Use:   "update <workflowId> <nodeRef>",
	Short: "Update a node",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newService()
		if err != nil {
			return err
		}

		sets, _ := cmd.Flags().GetStringSlice("set")
		unsets, _ := cmd.Flags().GetStringSlice("unset")
		patchFile, _ := cmd.Flags().GetString("patch-file")
		mergeFile, _ := cmd.Flags().GetString("merge-file")
		replaceFile, _ := cmd.Flags().GetString("replace-file")
		rename, _ := cmd.Flags().GetString("rename")
		move, _ := cmd.Flags().GetString("move")
		enable, _ := cmd.Flags().GetBool("enable")
		disable, _ := cmd.Flags().GetBool("disable")

		var patches []parser.NodePatch
		for _, s := range sets {
			p, err := parser.ParseSetFlag(s)
			if err != nil {
				return err
			}
			patches = append(patches, p)
		}

		var moveX, moveY *float64
		if move != "" {
			x, y := 0.0, 0.0
			if _, err := fmt.Sscanf(move, "%f,%f", &x, &y); err != nil {
				return fmt.Errorf("invalid --move format: expected x,y")
			}
			moveX = &x
			moveY = &y
		}

		node, err := svc.UpdateNode(args[0], args[1], patches, unsets, mergeFile, replaceFile, patchFile, rename, moveX, moveY, enable, disable, config.IsDryRun())
		if err != nil {
			return err
		}

		prefix := ""
		if config.IsDryRun() {
			prefix = "[dry-run] "
		}
		printResult(cmd,
			prefix+output.NodeSummary(node),
			prefix+output.NodeResolved(node),
			node,
		)
		return nil
	},
}

var nodeDeleteCmd = &cobra.Command{
	Use:   "delete <workflowId> <nodeRef>",
	Short: "Delete a node from a workflow",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newService()
		if err != nil {
			return err
		}

		yes, _ := cmd.Flags().GetBool("yes")
		cascade, _ := cmd.Flags().GetBool("cascade")
		rewire, _ := cmd.Flags().GetString("rewire")

		if !yes && !config.IsDryRun() {
			if !confirmAction(fmt.Sprintf("Delete node %s from workflow %s?", args[1], args[0])) {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		opts := parser.DeleteOptions{
			Cascade:        cascade,
			RewireStrategy: rewire,
		}

		if err := svc.DeleteNode(args[0], args[1], opts, config.IsDryRun()); err != nil {
			return err
		}

		if config.IsDryRun() {
			fmt.Printf("[dry-run] Would delete node %s from workflow %s.\n", args[1], args[0])
		} else {
			fmt.Printf("Node %s deleted from workflow %s.\n", args[1], args[0])
		}
		return nil
	},
}

var nodeRenameCmd = &cobra.Command{
	Use:   "rename <workflowId> <nodeRef> <newName>",
	Short: "Rename a node",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newService()
		if err != nil {
			return err
		}

		node, err := svc.RenameNode(args[0], args[1], args[2], config.IsDryRun())
		if err != nil {
			return err
		}

		prefix := ""
		if config.IsDryRun() {
			prefix = "[dry-run] "
		}
		printResult(cmd,
			fmt.Sprintf("%sRenamed to: %s [%s]\n", prefix, node.Name, node.Ref),
			prefix+output.NodeSummary(node),
			node,
		)
		return nil
	},
}

var nodeMoveCmd = &cobra.Command{
	Use:   "move <workflowId> <nodeRef>",
	Short: "Move a node to a new position",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newService()
		if err != nil {
			return err
		}

		pos, _ := cmd.Flags().GetString("position")
		if pos == "" {
			return fmt.Errorf("--position is required (format: x,y)")
		}
		var x, y float64
		if _, err := fmt.Sscanf(pos, "%f,%f", &x, &y); err != nil {
			return fmt.Errorf("invalid position format: expected x,y")
		}

		node, err := svc.MoveNode(args[0], args[1], x, y, config.IsDryRun())
		if err != nil {
			return err
		}

		prefix := ""
		if config.IsDryRun() {
			prefix = "[dry-run] "
		}
		printResult(cmd,
			fmt.Sprintf("%sMoved %s to [%.0f, %.0f]\n", prefix, node.Name, x, y),
			prefix+output.NodeSummary(node),
			node,
		)
		return nil
	},
}

var nodeEnableCmd = &cobra.Command{
	Use:   "enable <workflowId> <nodeRef>",
	Short: "Enable a node",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newService()
		if err != nil {
			return err
		}

		node, err := svc.EnableNode(args[0], args[1], config.IsDryRun())
		if err != nil {
			return err
		}

		prefix := ""
		if config.IsDryRun() {
			prefix = "[dry-run] "
		}
		fmt.Printf("%sNode %s [%s] enabled.\n", prefix, node.Name, node.Ref)
		return nil
	},
}

var nodeDisableCmd = &cobra.Command{
	Use:   "disable <workflowId> <nodeRef>",
	Short: "Disable a node",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newService()
		if err != nil {
			return err
		}

		node, err := svc.DisableNode(args[0], args[1], config.IsDryRun())
		if err != nil {
			return err
		}

		prefix := ""
		if config.IsDryRun() {
			prefix = "[dry-run] "
		}
		fmt.Printf("%sNode %s [%s] disabled.\n", prefix, node.Name, node.Ref)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(nodeCmd)

	nodeListCmd.Flags().String("type", "", "filter by node type")
	nodeListCmd.Flags().String("name", "", "filter by name pattern")
	nodeListCmd.Flags().Bool("with-connections", false, "show connection info")
	nodeListCmd.Flags().Bool("with-credentials", false, "show credentials")
	nodeListCmd.Flags().String("sort", "name", "sort by: name, type, topology")
	nodeCmd.AddCommand(nodeListCmd)

	nodeGetCmd.Flags().String("view", "summary", "view: summary, params, connections, raw")
	nodeGetCmd.Flags().Bool("resolve-credentials", false, "resolve credential references")
	nodeCmd.AddCommand(nodeGetCmd)

	nodeCreateCmd.Flags().String("name", "", "node name")
	nodeCreateCmd.Flags().String("type", "", "node type (e.g. n8n-nodes-base.httpRequest)")
	nodeCreateCmd.Flags().Int("type-version", 0, "node type version")
	nodeCreateCmd.Flags().String("position", "", "position as x,y")
	nodeCreateCmd.Flags().Bool("disabled", false, "create disabled")
	nodeCreateCmd.Flags().String("params-file", "", "parameters JSON file")
	nodeCreateCmd.Flags().String("credentials-file", "", "credentials JSON file")
	nodeCreateCmd.Flags().String("file", "", "full node JSON file")
	nodeCreateCmd.Flags().Bool("stdin", false, "read node JSON from stdin")
	nodeCreateCmd.Flags().String("connect-from", "", "connect from nodeRef[:output]")
	nodeCreateCmd.Flags().String("connect-to", "", "connect to nodeRef[:input]")
	nodeCmd.AddCommand(nodeCreateCmd)

	nodeUpdateCmd.Flags().StringSlice("set", nil, "set path=value")
	nodeUpdateCmd.Flags().StringSlice("unset", nil, "remove a path")
	nodeUpdateCmd.Flags().String("patch-file", "", "JSON patch file")
	nodeUpdateCmd.Flags().String("merge-file", "", "JSON merge file")
	nodeUpdateCmd.Flags().String("replace-file", "", "full node replacement JSON")
	nodeUpdateCmd.Flags().String("rename", "", "rename node")
	nodeUpdateCmd.Flags().String("move", "", "move to x,y")
	nodeUpdateCmd.Flags().Bool("enable", false, "enable node")
	nodeUpdateCmd.Flags().Bool("disable", false, "disable node")
	nodeCmd.AddCommand(nodeUpdateCmd)

	nodeDeleteCmd.Flags().Bool("cascade", false, "remove all connections")
	nodeDeleteCmd.Flags().String("rewire", "", "rewire strategy: none, skip, bridge")
	nodeDeleteCmd.Flags().Bool("yes", false, "skip confirmation")
	nodeCmd.AddCommand(nodeDeleteCmd)

	nodeCmd.AddCommand(nodeRenameCmd)

	nodeMoveCmd.Flags().String("position", "", "target position as x,y")
	nodeCmd.AddCommand(nodeMoveCmd)

	nodeCmd.AddCommand(nodeEnableCmd)
	nodeCmd.AddCommand(nodeDisableCmd)
}

func nodeInputFromRaw(raw map[string]interface{}) parser.NodeInput {
	input := parser.NodeInput{RawJSON: raw}
	if name, ok := raw["name"].(string); ok {
		input.Name = name
	}
	if typ, ok := raw["type"].(string); ok {
		input.Type = typ
	}
	if tv, ok := raw["typeVersion"].(float64); ok {
		input.TypeVersion = int(tv)
	}
	if pos, ok := raw["position"].([]interface{}); ok && len(pos) >= 2 {
		if x, ok := pos[0].(float64); ok {
			input.Position[0] = x
		}
		if y, ok := pos[1].(float64); ok {
			input.Position[1] = y
		}
	}
	if disabled, ok := raw["disabled"].(bool); ok {
		input.Disabled = disabled
	}
	if params, ok := raw["parameters"].(map[string]interface{}); ok {
		input.Parameters = params
	}
	if creds, ok := raw["credentials"].(map[string]interface{}); ok {
		input.Credentials = creds
	}
	if notes, ok := raw["notes"].(string); ok {
		input.Notes = notes
	}
	return input
}
