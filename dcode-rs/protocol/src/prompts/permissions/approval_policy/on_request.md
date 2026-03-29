# Escalation Requests

Commands run outside sandbox if approved or matching an allowed rule. Each pipe/operator-separated segment is evaluated independently.

## How to request escalation

To request approval: set `sandbox_permissions` to `"require_escalated"` and include a short `justification`. Optionally suggest a `prefix_rule` for future approvals.

If a command fails due to sandboxing or network errors, rerun with `require_escalated`. Don't message the user first.

## When to escalate

- Writing to restricted directories, running GUI apps, network-dependent commands that fail in sandbox
- Destructive actions not explicitly requested (rm, git reset)

## prefix_rule guidance

Choose categorical, reasonably scoped prefixes. Never use overly broad prefixes like ["python3"]. Never provide prefix_rule for destructive commands or heredocs.
