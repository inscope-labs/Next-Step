# STATE 2 — Live Interaction

Use the scaffold folder confirmed in State 0's `protocol/current/spec/` listing (not a
guessed name). Read its `README.md` for exact substitutions. Never invent
a task.

## Step 1 — ask for scope and workspace

Ask the human, in chat, two things in one message: (a) one line
describing what the demo file should contain, and (b) which workspace to
use. For (b), leave it open — an existing workspace ID, or "new" with a
short name and one-line purpose — without listing or suggesting specific
IDs from State 0's `workspace/` listing; workspace selection is the
human's decision, not a menu to choose from. Wait for their answer before
continuing.

If the answer to (a) doesn't read as a one-line description (for example
it looks like literal file content, or contains an unresolved placeholder
such as `[insert X here]`), don't guess at the intent — treat it as the
"ambiguous Step 1 answer" case in Failure, below, and ask the human to
confirm whether it's literal text to write into the file or a description
to interpret.

## Step 2 — claim workspace (only if the human said "new")

Don't hand-roll the claim — call the script that owns the atomic-claim
pattern:

```bash
H="${NEXT_STEP_HOME:-$HOME/next-step}"
"$H/bin/create-workspace.sh" --name "<name>" --purpose "<purpose>" | "$H/bin/next-step-clipboard.sh"
```

Read the new `<WORKSPACE_ID>` from its output. If the human gave an
existing ID instead, skip this step and use that ID directly.

## Step 3 — read the scaffold (single clipboard write)

Read the README and all three `.demo` files in one shot. Never chain
multiple `cat | clipboard` calls for a multi-file read — each overwrites
the last. Concatenate to one temp file first, one clipboard write:

```bash
H="${NEXT_STEP_HOME:-$HOME/next-step}"
TMPFILE="$(mktemp)"
{
  echo "--- README.md ---"; cat "$H/protocol/current/spec/demo/README.md"
  echo "--- task.manifest.demo ---"; cat "$H/protocol/current/spec/demo/task.manifest.demo"
  echo "--- start.sh.demo ---"; cat "$H/protocol/current/spec/demo/start.sh.demo"
  echo "--- verify.sh.demo ---"; cat "$H/protocol/current/spec/demo/verify.sh.demo"
} > "$TMPFILE"
cat "$TMPFILE" | "$H/bin/next-step-clipboard.sh"
```

## Step 4 — fill and stage

Fill the scaffold's `.demo` files per its `README.md` substitutions only,
including `{{WORKSPACE_ID}}`. Follow the manifest fields and command
sequence already confirmed in State 1 from `protocol/current/spec/PROTOCOL-FACTS.md`
-- `WORKSPACE_ID`, `WORKSPACE_NAME`, `WRITE_PATHS` in the manifest, then
`build-task.sh` -> `--show` -> `--approve` -> `--run` -- don't re-derive it,
don't inspect any script source:

```bash
H="${NEXT_STEP_HOME:-$HOME/next-step}"
mkdir -p "$H/workspace/<WORKSPACE_ID>/inbox/task-<TASK_ID>"
# place the three filled files there, per PROTOCOL-FACTS.md
"$H/build-task.sh" --workspace <WORKSPACE_ID> <TASK_ID>
"$H/run-task.sh" --show "$H/workspace/<WORKSPACE_ID>/tasks/task-<NNN>_<TASK_ID>.zip" | "$H/bin/next-step-clipboard.sh"
```

## Step 5 — authorize (human only)

```
Only you can authorize this — please run:

H="${NEXT_STEP_HOME:-$HOME/next-step}"
"$H/run-task.sh" --approve "$H/workspace/<WORKSPACE_ID>/tasks/task-<NNN>_<TASK_ID>.zip" | "$H/bin/next-step-clipboard.sh"
```

Do not continue until the output confirms it.

## Step 6 — execute + verify

```bash
H="${NEXT_STEP_HOME:-$HOME/next-step}"
"$H/run-task.sh" "$H/workspace/<WORKSPACE_ID>/tasks/task-<NNN>_<TASK_ID>.zip" | "$H/bin/next-step-clipboard.sh"
```

Read `VERIFICATION: PASS|FAIL` from the printed report. A zero exit code
alone proves nothing.

## Success

```
State 2 OK. workspace=<WORKSPACE_ID> verified: <PASS/FAIL from the report>

H="${NEXT_STEP_HOME:-$HOME/next-step}"
cat "$H/protocol/current/spec/state-3.md" | "$H/bin/next-step-clipboard.sh"
```

## Failure

```
State 2 FAILED: <scaffold folder not found in State 0's listing | ambiguous Step 1 answer | workspace claim failed | authorization unconfirmed | execution/verification failed>.
Next: 1) report this to the human as-is, 2) do not invent a substitute,
3) do not retry with a different command.
```
