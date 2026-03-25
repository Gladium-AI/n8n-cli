# n8n CLI Command Map

## Workflow

```bash
n8n-cli workflow list [--active] [--limit <n>] [--json]
n8n-cli workflow get <workflowId> [--json]
n8n-cli workflow create --file <workflow.json> [--json]
n8n-cli workflow update <workflowId> --file <workflow.json> [--json]
n8n-cli workflow inspect <workflowId> --with-nodes --with-connections [--json]
n8n-cli workflow activate <workflowId> [--json]
n8n-cli workflow deactivate <workflowId> [--json]
n8n-cli workflow delete <workflowId> --yes [--json]
```

## Node

```bash
n8n-cli node list <workflowId> [--type <nodeType>] [--json]
n8n-cli node get <workflowId> <nodeRef> --view summary|details|json|params|connections [--json]
n8n-cli node create <workflowId> --json-file <node.json> [--json]
n8n-cli node update <workflowId> <nodeRef> --set <path=value> [--dry-run] [--json]
n8n-cli node rename <workflowId> <nodeRef> "<newName>" [--json]
n8n-cli node move <workflowId> <nodeRef> --position <x,y> [--json]
n8n-cli node enable <workflowId> <nodeRef> [--json]
n8n-cli node disable <workflowId> <nodeRef> [--json]
n8n-cli node delete <workflowId> <nodeRef> --cascade|--rewire bridge [--json]
```

## Connection

```bash
n8n-cli connection list <workflowId> [--node <nodeRef>] [--direction in|out] [--json]
n8n-cli connection create <workflowId> --from <nodeRef> --to <nodeRef> [--json]
n8n-cli connection delete <workflowId> --from <nodeRef> --to <nodeRef> [--json]
```

## Execution, Graph, and Test

```bash
n8n-cli execution list --workflow-id <workflowId> [--status <status>] [--limit <n>] [--json]
n8n-cli execution get <executionId> --with-data [--json]
n8n-cli execution retry <executionId> [--json]
n8n-cli execution stop <executionId> [--json]
n8n-cli execution delete <executionId> --yes [--json]
n8n-cli graph inspect <workflowId> --with-adjacency --with-orphans --with-cycles [--json]
n8n-cli test webhook <workflowId> --payload-file <payload.json> [--json]
```

## Global Flags

```bash
--base-url <url>
--api-key <key>
--output summary|resolved|raw
--json
--yaml
--dry-run
--quiet
--no-color
```
