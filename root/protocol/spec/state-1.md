# STATE 1 — Architecture & Authorization Model

Read the canonical facts sheet found in State 0's `protocol/spec/`
listing — `protocol/spec/PROTOCOL-FACTS.md`, using the exact path State 0
confirmed. Do not read `run-task.sh`/`build-task.sh` source to derive
these facts; that file already has them.

## Command

```bash
H="${NEXT_STEP_HOME:-$HOME/next-step}"
cat "$H/protocol/spec/PROTOCOL-FACTS.md" | "$H/bin/next-step-clipboard.sh"
```

## Facts (from the sheet — copy them, don't re-derive them)

- protocol/version
- task package format
- execution entry point
- verification approach
- **who authorizes a task** — not optional

## Hard gate

If the sheet doesn't clearly say the human (not the AI) is the only one
who can authorize a task, this state fails. No exceptions.

## Success

```
State 1 OK.
facts: <one line per fact from the sheet>
auth: human-only (source: PROTOCOL-FACTS.md)

H="${NEXT_STEP_HOME:-$HOME/next-step}"
cat "$H/protocol/spec/state-2.md" | "$H/bin/next-step-clipboard.sh"
```

## Failure

```
State 1 FAILED: <PROTOCOL-FACTS.md missing | authorization boundary unclear>.
Next: 1) confirm protocol/spec/PROTOCOL-FACTS.md exists at that path,
2) if it's missing, ask the human — do not substitute run-task.sh source
   reading for it, 3) do not proceed until unambiguous.
```
