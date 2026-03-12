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

// --- node get ---

var nodeGetCmd = &cobra.Command{
	Use:   "get <workflowId> <nodeRef>",
	Short: "Inspect one node",
	Long: `Inspect a single node with multiple view modes.

Views:
  summary      Compact, LLM-friendly overview (default)
  details      Structured view reflecting native JSON layout
  json         Exact native n8n node JSON, copy-paste safe
  params       Only the parameters object
  connections  Node info plus incoming/outgoing connections

Parameter extraction:
  --param parameters.email        Extract a single value
  --param parameters.resource --param parameters.operation   Multiple values`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newService()
		if err != nil {
			return err
		}

		node, err := svc.GetNode(args[0], args[1])
		if err != nil {
			return err
		}

		// --param extraction takes precedence over --view
		paramPaths, _ := cmd.Flags().GetStringSlice("param")
		if len(paramPaths) > 0 {
			if config.IsJSON() {
				// JSON serialization of extracted params
				if len(paramPaths) == 1 {
					val, err := parser.ExtractPath(node, paramPaths[0])
					if err != nil {
						return err
					}
					fmt.Println(output.JSON(val))
				} else {
					result := make(map[string]interface{})
					for _, p := range paramPaths {
						val, err := parser.ExtractPath(node, p)
						if err != nil {
							return fmt.Errorf("path %q: %w", p, err)
						}
						result[p] = val
					}
					fmt.Println(output.JSON(result))
				}
			} else {
				fmt.Print(output.NodeParamExtract(node, paramPaths))
			}
			return nil
		}

		viewFlag, _ := cmd.Flags().GetString("view")
		view := parser.ParseNodeView(viewFlag)

		// --json flag forces JSON serialization of whatever view is selected
		if config.IsJSON() {
			switch view {
			case parser.ViewParams:
				fmt.Println(output.JSON(node.Parameters))
			case parser.ViewJSON:
				fmt.Println(output.NodeViewJSON(node))
			default:
				fmt.Println(output.NodeViewJSON(node))
			}
			return nil
		}

		switch view {
		case parser.ViewDetails:
			fmt.Print(output.NodeViewDetails(node))
		case parser.ViewJSON:
			fmt.Println(output.NodeViewJSON(node))
		case parser.ViewParams:
			fmt.Println(output.NodeViewParams(node))
		case parser.ViewConnections:
			fmt.Print(output.NodeViewConnections(node))
		default:
			fmt.Print(output.NodeViewSummary(node))
		}
		return nil
	},
}

// --- node list ---

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

		if config.IsJSON() {
			// Return native node JSON for each node
			var rawNodes []map[string]interface{}
			for _, n := range filtered {
				rawNodes = append(rawNodes, parser.NodeToNativeJSON(n))
			}
			fmt.Println(output.JSON(rawNodes))
			return nil
		}

		fmt.Print(output.NodeListSummary(filtered))
		return nil
	},
}

// --- node create ---

var nodeCreateCmd = &cobra.Command{
	Use:   "create <workflowId>",
	Short: "Create a node in a workflow",
	Long: `Create a node using native n8n JSON or structured CLI flags.

Mode 1: native node JSON input (primary)
  n8n-cli node create 123 --json-file create-contact.json
  cat node.json | n8n-cli node create 123 --stdin

Mode 2: structured CLI flags (convenience)
  n8n-cli node create 123 \
    --name "Create Contact" \
    --type "n8n-nodes-base.hubspot" \
    --type-version 2 \
    --position 840,320 \
    --param resource=contact \
    --param operation=create \
    --param email='={{$json.email}}'`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newService()
		if err != nil {
			return err
		}

		input := parser.NodeInput{}

		jsonFile, _ := cmd.Flags().GetString("json-file")
		file, _ := cmd.Flags().GetString("file")
		stdin, _ := cmd.Flags().GetBool("stdin")

		// --json-file takes precedence, fall back to --file for backward compat
		sourceFile := jsonFile
		if sourceFile == "" {
			sourceFile = file
		}

		if sourceFile != "" {
			data, err := os.ReadFile(sourceFile)
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

		// CLI flags override file values
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

		// --param flags: build parameters from k=v pairs
		paramFlags, _ := cmd.Flags().GetStringSlice("param")
		if len(paramFlags) > 0 {
			if input.Parameters == nil {
				input.Parameters = make(map[string]interface{})
			}
			for _, pf := range paramFlags {
				eqIdx := strings.Index(pf, "=")
				if eqIdx < 0 {
					return fmt.Errorf("invalid --param format %q: expected key=value", pf)
				}
				key := pf[:eqIdx]
				rawVal := pf[eqIdx+1:]
				var val interface{}
				if err := json.Unmarshal([]byte(rawVal), &val); err != nil {
					val = rawVal
				}
				setNestedParam(input.Parameters, key, val)
			}
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
			if config.IsJSON() {
				fmt.Println(output.NodeViewJSON(node))
			} else {
				fmt.Print(output.NodeViewDetails(node))
			}
			return nil
		}

		node, _, err := svc.CreateNode(args[0], input, connectFrom, connectTo)
		if err != nil {
			return err
		}

		if config.IsJSON() {
			fmt.Println(output.NodeViewJSON(node))
		} else {
			fmt.Print(output.NodeViewDetails(node))
		}
		return nil
	},
}

// --- node update ---

var nodeUpdateCmd = &cobra.Command{
	Use:   "update <workflowId> <nodeRef>",
	Short: "Update a node",
	Long: `Update a node via full replacement or partial path mutation.

Mode 1: full replacement with native node JSON
  n8n-cli node update 123 "Create Contact" --replace-json-file node.json
  cat node.json | n8n-cli node update 123 n1 --stdin --replace-json

Mode 2: path-based partial mutation
  n8n-cli node update 123 "Create Contact" \
    --set parameters.email='={{$json.workEmail}}' \
    --set parameters.firstName='={{$json.first_name}}'
  n8n-cli node update 123 n2 --unset parameters.phone
  n8n-cli node update 123 n2 --merge-json-file partial.json`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newService()
		if err != nil {
			return err
		}

		sets, _ := cmd.Flags().GetStringSlice("set")
		unsets, _ := cmd.Flags().GetStringSlice("unset")
		patchFile, _ := cmd.Flags().GetString("patch-file")
		mergeJSONFile, _ := cmd.Flags().GetString("merge-json-file")
		mergeFile, _ := cmd.Flags().GetString("merge-file")
		replaceJSONFile, _ := cmd.Flags().GetString("replace-json-file")
		replaceFile, _ := cmd.Flags().GetString("replace-file")
		rename, _ := cmd.Flags().GetString("rename")
		move, _ := cmd.Flags().GetString("move")
		enable, _ := cmd.Flags().GetBool("enable")
		disable, _ := cmd.Flags().GetBool("disable")

		// Prefer new flag names, fall back to old ones
		if mergeJSONFile == "" {
			mergeJSONFile = mergeFile
		}
		if replaceJSONFile == "" {
			replaceJSONFile = replaceFile
		}

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

		result, err := svc.UpdateNode(args[0], args[1], patches, unsets, mergeJSONFile, replaceJSONFile, patchFile, rename, moveX, moveY, enable, disable, config.IsDryRun())
		if err != nil {
			return err
		}

		prefix := ""
		if config.IsDryRun() {
			prefix = "[dry-run] "
		}

		if config.IsJSON() {
			fmt.Println(output.NodeViewJSON(result.Node))
			return nil
		}

		// Default: show before/after diff summary
		fmt.Print(prefix + output.NodeUpdateDiff(result.Node, result.ChangedPaths))
		return nil
	},
}

// --- node delete ---

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

// --- node rename ---

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

		if config.IsJSON() {
			fmt.Println(output.NodeViewJSON(node))
		} else {
			fmt.Printf("%sRenamed to: %s [%s]\n", prefix, node.Name, node.Ref)
		}
		return nil
	},
}

// --- node move ---

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
		fmt.Printf("%sMoved %s to [%.0f, %.0f]\n", prefix, node.Name, x, y)
		return nil
	},
}

// --- node enable/disable ---

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

// --- init ---

func init() {
	rootCmd.AddCommand(nodeCmd)

	// node get
	nodeGetCmd.Flags().String("view", "summary", "view: summary, details, json, params, connections")
	nodeGetCmd.Flags().StringSlice("param", nil, "extract specific path(s) from node JSON")
	nodeCmd.AddCommand(nodeGetCmd)

	// node list
	nodeListCmd.Flags().String("type", "", "filter by node type")
	nodeListCmd.Flags().String("name", "", "filter by name pattern")
	nodeListCmd.Flags().Bool("with-connections", false, "show connection info")
	nodeListCmd.Flags().Bool("with-credentials", false, "show credentials")
	nodeListCmd.Flags().String("sort", "name", "sort by: name, type, topology")
	nodeCmd.AddCommand(nodeListCmd)

	// node create
	nodeCreateCmd.Flags().String("json-file", "", "native node JSON file (primary input)")
	nodeCreateCmd.Flags().String("file", "", "node JSON file (alias for --json-file)")
	nodeCreateCmd.Flags().Bool("stdin", false, "read node JSON from stdin")
	nodeCreateCmd.Flags().String("name", "", "node name")
	nodeCreateCmd.Flags().String("type", "", "node type (e.g. n8n-nodes-base.httpRequest)")
	nodeCreateCmd.Flags().Int("type-version", 0, "node type version")
	nodeCreateCmd.Flags().String("position", "", "position as x,y")
	nodeCreateCmd.Flags().Bool("disabled", false, "create disabled")
	nodeCreateCmd.Flags().StringSlice("param", nil, "set parameter as key=value (convenience)")
	nodeCreateCmd.Flags().String("params-file", "", "parameters JSON file")
	nodeCreateCmd.Flags().String("credentials-file", "", "credentials JSON file")
	nodeCreateCmd.Flags().String("connect-from", "", "connect from nodeRef[:output]")
	nodeCreateCmd.Flags().String("connect-to", "", "connect to nodeRef[:input]")
	nodeCmd.AddCommand(nodeCreateCmd)

	// node update
	nodeUpdateCmd.Flags().StringSlice("set", nil, "set path=value (dot-path into native node JSON)")
	nodeUpdateCmd.Flags().StringSlice("unset", nil, "remove a path")
	nodeUpdateCmd.Flags().String("replace-json-file", "", "replace node with full native JSON file")
	nodeUpdateCmd.Flags().String("replace-file", "", "alias for --replace-json-file")
	nodeUpdateCmd.Flags().String("merge-json-file", "", "deep merge partial JSON into node")
	nodeUpdateCmd.Flags().String("merge-file", "", "alias for --merge-json-file")
	nodeUpdateCmd.Flags().String("patch-file", "", "JSON patch file")
	nodeUpdateCmd.Flags().Bool("stdin", false, "read from stdin")
	nodeUpdateCmd.Flags().String("rename", "", "rename node")
	nodeUpdateCmd.Flags().String("move", "", "move to x,y")
	nodeUpdateCmd.Flags().Bool("enable", false, "enable node")
	nodeUpdateCmd.Flags().Bool("disable", false, "disable node")
	nodeCmd.AddCommand(nodeUpdateCmd)

	// node delete
	nodeDeleteCmd.Flags().Bool("cascade", false, "remove all connections")
	nodeDeleteCmd.Flags().String("rewire", "", "rewire strategy: none, skip, bridge")
	nodeDeleteCmd.Flags().Bool("yes", false, "skip confirmation")
	nodeCmd.AddCommand(nodeDeleteCmd)

	// node rename / move / enable / disable
	nodeCmd.AddCommand(nodeRenameCmd)
	nodeMoveCmd.Flags().String("position", "", "target position as x,y")
	nodeCmd.AddCommand(nodeMoveCmd)
	nodeCmd.AddCommand(nodeEnableCmd)
	nodeCmd.AddCommand(nodeDisableCmd)
}

// --- helpers ---

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

func setNestedParam(m map[string]interface{}, key string, value interface{}) {
	parts := strings.Split(key, ".")
	current := m
	for i := 0; i < len(parts)-1; i++ {
		next, ok := current[parts[i]].(map[string]interface{})
		if !ok {
			next = make(map[string]interface{})
			current[parts[i]] = next
		}
		current = next
	}
	current[parts[len(parts)-1]] = value
}
