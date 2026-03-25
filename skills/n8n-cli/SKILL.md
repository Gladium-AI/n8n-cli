---
name: n8n-cli
description: Use this skill when the task is to inspect, create, update, test, debug, or analyze n8n workflows and nodes with n8n-cli. Trigger on requests about workflow CRUD, node-level edits, connection changes, graph inspection, execution debugging, or webhook testing through this CLI.
license: MIT
metadata:
  author: Gladium AI
  version: 1.0.0
  category: developer-tools
  tags:
    - n8n
    - cli
    - workflow-automation
    - golang
    - api
    - agent-tools
---

# n8n CLI

Use this skill to operate `n8n-cli` safely and consistently.

## Use This Skill For

- Inspecting workflows, nodes, connections, executions, and graph structure
- Making parser-backed node edits without rewriting whole workflow JSON blobs
- Creating, updating, activating, deactivating, or deleting workflows
- Debugging webhook tests and execution failures
- Producing machine-readable workflow data for follow-up agent steps

## Core Rules

- Prefer `n8n-cli` over raw n8n API calls when the task fits the CLI surface.
- Prefer `--json` when another agent or program will consume the result.
- Use explicit workflow IDs and node refs like `n0`, `ref:n0`, or `id:<uuid>` when possible.
- For mutating commands, prefer `--dry-run` first when validating a change or planning a patch.
- Do not expose or hardcode `N8N_API_KEY`; use env vars, config, or flags already configured by the user.
- Treat `delete`, `deactivate`, `stop`, and connection rewiring as destructive operations and verify intent before using them.

## Quick Workflow

1. Verify config and target workflow.
2. Inspect the workflow with `workflow get` or `workflow inspect`.
3. Inspect nodes with `node list` and `node get`.
4. Apply surgical changes with `node`, `connection`, or `workflow update`.
5. Use `graph inspect`, `execution`, and `test webhook` to validate behavior.

## Prerequisites

- `n8n-cli` available on `PATH`, or use the repo-local binary or `go run .`
- `N8N_API_KEY` configured
- `N8N_BASE_URL` configured when not using the default `http://localhost:5678`

For concrete command sequences, read [references/workflows.md](references/workflows.md).
For command coverage and flag patterns, read [references/commands.md](references/commands.md).

## Standard Command Path

Inspect a workflow and its nodes:

```bash
n8n-cli workflow inspect <workflowId> --with-nodes --with-connections --json
n8n-cli node list <workflowId> --json
n8n-cli node get <workflowId> n0 --view details --json
```

Apply a targeted node change safely:

```bash
n8n-cli node update <workflowId> n0 --set parameters.url=https://api.example.com --dry-run --json
n8n-cli node update <workflowId> n0 --set parameters.url=https://api.example.com --json
```

Debug structure and runtime behavior:

```bash
n8n-cli graph inspect <workflowId> --with-adjacency --with-cycles --json
n8n-cli execution list --workflow-id <workflowId> --limit 20 --json
```

## Troubleshooting

- Start with `workflow inspect` if the intended node or connection is unclear.
- Use `node get --view json` when you need exact native n8n payload shape.
- Use `graph inspect` when a workflow behaves incorrectly after rewiring or node deletion.
- Use `test webhook` and `execution get --with-data` when debugging runtime behavior.
