# n8n CLI Workflows

Use these patterns when operating `n8n-cli`.

## 1. Inspect a Workflow

```bash
n8n-cli workflow get <workflowId> --json
n8n-cli workflow inspect <workflowId> --with-nodes --with-connections --json
n8n-cli node list <workflowId> --json
```

## 2. Make a Surgical Node Change

```bash
n8n-cli node get <workflowId> n0 --view details --json
n8n-cli node update <workflowId> n0 --set parameters.url=https://api.example.com --dry-run --json
n8n-cli node update <workflowId> n0 --set parameters.url=https://api.example.com --json
```

## 3. Create a Node and Connect It

```bash
n8n-cli node create <workflowId> --json-file ./node.json --json
n8n-cli connection create <workflowId> --from n0 --to n1 --json
n8n-cli workflow inspect <workflowId> --with-connections --json
```

## 4. Remove or Rewire a Node

```bash
n8n-cli node delete <workflowId> n2 --rewire bridge --dry-run --json
n8n-cli node delete <workflowId> n2 --rewire bridge --json
```

Use `--cascade` instead when downstream reconnection is not desired.

## 5. Debug Runtime Behavior

```bash
n8n-cli test webhook <workflowId> --payload-file ./payload.json --json
n8n-cli execution list --workflow-id <workflowId> --limit 10 --json
n8n-cli execution get <executionId> --with-data --json
```

## Notes

- Prefer `--json` for machine consumption.
- Prefer `--dry-run` before node mutations when you need to verify the exact change surface.
- Prefer parser refs like `n0` when node names are duplicated or unstable.
