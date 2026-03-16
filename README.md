# n8n-cli

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

AI-agent-friendly CLI for [n8n](https://n8n.io) workflow automation. Exposes **node-level operations** via a parser layer on top of the n8n REST API — so you can list, inspect, create, and modify individual nodes without reading or rewriting entire workflow JSON.

## Why n8n-cli?

The native n8n API only offers workflow-level CRUD. To change a single node, you'd need to GET the full workflow, find the node in the JSON, edit it, and PUT the entire blob back. **n8n-cli's parser layer** handles that automatically:

```
fetch → parse → mutate → rehydrate → save
```

Every node gets a stable ref (`n0`, `n1`, …) by array position, so agents and scripts can address nodes without fragile name lookups.

## Installation

Requires **Go 1.22+**.

```bash
git clone <repo-url> && cd n8n-cli
go build -o n8n-cli .
```

Move the binary to your PATH:

```bash
mv n8n-cli /usr/local/bin/
```

## Configuration

Set your n8n instance credentials via environment variables, config file, or CLI flags.

### Environment variables

```bash
export N8N_BASE_URL=http://localhost:5678   # default
export N8N_API_KEY=your-api-key             # required
```

### Config file

Create `~/.n8n-cli.yaml` (or `.n8n-cli.yaml` in the working directory):

```yaml
base-url: http://localhost:5678
api-key: your-api-key
```

### CLI flags

```bash
n8n-cli --base-url http://my-n8n:5678 --api-key <key> workflow list
```

## Global Flags

| Flag | Description |
|------|-------------|
| `--base-url` | n8n instance base URL (env: `N8N_BASE_URL`) |
| `--api-key` | n8n API key (env: `N8N_API_KEY`) |
| `--output` | Output mode: `summary`, `resolved`, `raw` |
| `--json` | Force JSON output |
| `--yaml` | Force YAML output |
| `--dry-run` | Preview changes without applying |
| `--quiet` | Suppress non-essential output |
| `--no-color` | Disable color output |

## Node References

Nodes can be referenced in four ways:

| Format | Example | Description |
|--------|---------|-------------|
| Bare name | `"HTTP Request"` | Match by name (must be unique) |
| `ref:` | `ref:n0` | By stable parser-assigned ref |
| `id:` | `id:abc-123-uuid` | By n8n node UUID |
| `name:` | `name:"HTTP Request"` | Explicit name match |

Bare refs like `n0` also work without the `ref:` prefix.

## Commands

### workflow (alias: `wf`)

Full CRUD and lifecycle management for workflows.

```bash
# List all workflows
n8n-cli workflow list
n8n-cli workflow list --active --limit 10

# Get a workflow
n8n-cli workflow get <id>

# Create a workflow from JSON file
n8n-cli workflow create --file workflow.json
n8n-cli workflow create "My Workflow"

# Update a workflow
n8n-cli workflow update <id> --file updated.json
n8n-cli workflow update <id> --set "name=New Name"

# Activate / deactivate
n8n-cli workflow activate <id>
n8n-cli workflow deactivate <id>

# Inspect structure (nodes, connections, settings, tags)
n8n-cli workflow inspect <id> --with-nodes --with-connections

# Delete
n8n-cli workflow delete <id> --yes
```

### node

Parser-backed node-level CRUD — the core differentiator.

#### Inspect nodes

```bash
# List all nodes in a workflow
n8n-cli node list <workflowId>
n8n-cli node list <workflowId> --type n8n-nodes-base.httpRequest

# Get a node with different views
n8n-cli node get <workflowId> n0                          # summary (default)
n8n-cli node get <workflowId> n0 --view details           # structured detail
n8n-cli node get <workflowId> n0 --view json              # exact native n8n JSON
n8n-cli node get <workflowId> n0 --view params            # parameters only
n8n-cli node get <workflowId> n0 --view connections       # in/out connections

# Extract specific values by dot-path
n8n-cli node get <workflowId> n0 --param parameters.url
n8n-cli node get <workflowId> n0 --param credentials.httpBasicAuth.id
```

#### Create nodes

```bash
# From native n8n JSON file (primary mode)
n8n-cli node create <workflowId> --json-file node.json

# With CLI flags + convenience params
n8n-cli node create <workflowId> \
  --name "HTTP Request" \
  --type n8n-nodes-base.httpRequest \
  --param parameters.url=https://api.example.com \
  --param parameters.method=POST \
  --connect-from n0

# From stdin
cat node.json | n8n-cli node create <workflowId> --stdin
```

#### Update nodes

```bash
# Path-based patches (surgical edits)
n8n-cli node update <workflowId> n1 --set parameters.url=https://new.example.com
n8n-cli node update <workflowId> n1 --unset parameters.timeout

# Full replacement from JSON file
n8n-cli node update <workflowId> n1 --replace-json-file node-v2.json

# Deep merge from JSON file
n8n-cli node update <workflowId> n1 --merge-json-file patch.json

# Rename / move / enable / disable
n8n-cli node rename <workflowId> n1 "API Call"
n8n-cli node move <workflowId> n1 --position 400,300
n8n-cli node enable <workflowId> n1
n8n-cli node disable <workflowId> n1
```

Updates show a before/after diff of changed fields.

#### Delete nodes

```bash
# Cascade: remove node and all its connections
n8n-cli node delete <workflowId> n1 --cascade

# Bridge: remove node but reconnect its neighbors
n8n-cli node delete <workflowId> n1 --rewire bridge
```

### connection (alias: `conn`)

Manage edges between nodes.

```bash
# List connections for a workflow (or filtered by node)
n8n-cli connection list <workflowId>
n8n-cli connection list <workflowId> --node n1 --direction out

# Create a connection
n8n-cli connection create <workflowId> --from n0 --to n1

# With specific output/input indexes
n8n-cli connection create <workflowId> --from n0 --from-output 1 --to n2 --to-input 0

# Delete a connection
n8n-cli connection delete <workflowId> --from n0 --to n1
```

### execution (alias: `exec`)

Manage workflow executions.

```bash
n8n-cli execution list --workflow-id <id> --status success --limit 20
n8n-cli execution get <executionId> --with-data
n8n-cli execution retry <executionId>
n8n-cli execution stop <executionId>
n8n-cli execution delete <executionId> --yes
```

### graph

Analyze workflow graph structure.

```bash
n8n-cli graph inspect <workflowId>
n8n-cli graph inspect <workflowId> --with-adjacency --with-orphans --with-cycles
```

Returns: roots, leaves, topological order, cycle detection, branching points, orphan nodes.

### test

Webhook testing and execution debugging.

```bash
# Send a test webhook
n8n-cli test webhook <workflowId> --payload-file data.json
n8n-cli test webhook <workflowId> --method POST --header "X-Custom: value"

# Inspect execution results
n8n-cli test runs <workflowId> --status error --limit 5
n8n-cli test inspect <executionId>
n8n-cli test retry <executionId>
```

## Output Modes

| Mode | Flag | Description |
|------|------|-------------|
| Summary | `--output summary` | Compact, LLM-friendly (default) |
| Resolved | `--output resolved` | Structured detail view |
| Raw | `--output raw` | Full n8n JSON |
| JSON | `--json` | Force JSON serialization |
| YAML | `--yaml` | Force YAML serialization |

The `--view` flag on `node get` controls the *logical slice* (what to show), while `--json`/`--yaml` controls the *serialization format*.

## Dry Run

All mutating node/connection commands support `--dry-run` to preview changes without saving:

```bash
n8n-cli node update <workflowId> n1 --set parameters.url=https://new.com --dry-run
n8n-cli node delete <workflowId> n2 --cascade --dry-run
```

## Architecture

```
cmd/                          # Cobra commands (thin: parse flags → call service → format output)
  root.go                     # Root command, global flags, config init
  workflow.go                 # workflow create/list/get/update/delete/activate/deactivate/inspect
  execution.go                # execution list/get/delete/retry/stop
  node.go                     # node list/get/create/update/delete/rename/move/enable/disable
  connection.go               # connection list/create/delete
  graph.go                    # graph inspect
  testcmd.go                  # test retry/runs/inspect/webhook

internal/
  config/config.go            # Viper-based configuration (env, file, flags)
  client/client.go            # n8n REST API client (Resty)
  parser/
    types.go                  # ParsedWorkflow, ParsedNode, ParsedEdge, GraphAnalysis
    parser.go                 # Parse, Rehydrate, ResolveRef, ExtractPath, DetectChanges
    mutate.go                 # AddNode, UpdateNode, RemoveNode, AddEdge, RemoveEdge
    graph.go                  # AnalyzeGraph (roots, leaves, topo sort, cycles)
    parser_test.go            # 26 unit tests
  service/service.go          # Orchestration layer (client + parser)
  output/format.go            # Output formatting (summary/details/json/params/connections)
```

**Key design decisions:**

- n8n connections are keyed by source node **name** (not ID) — rename operations must update connection keys
- The parser assigns stable refs (`n0`, `n1`, …) by array position in the workflow's `nodes[]` array
- `node get --view json` returns the exact native n8n node object — safe for copy-paste into other tools
- `node update` returns a before/after diff showing which fields changed

## Development

```bash
# Run tests (26 parser tests)
go test ./...

# Static analysis
go vet ./...

# Build
go build -o n8n-cli .
```

## Tech Stack

- **Language**: Go 1.22
- **CLI framework**: [spf13/cobra](https://github.com/spf13/cobra) + pflag
- **Config**: [spf13/viper](https://github.com/spf13/viper)
- **HTTP client**: [go-resty/resty/v2](https://github.com/go-resty/resty)
- **JSON querying**: [tidwall/gjson](https://github.com/tidwall/gjson) + [tidwall/sjson](https://github.com/tidwall/sjson)
- **Logging**: [go.uber.org/zap](https://github.com/uber-go/zap)
