# STATE 0 — Discovery & Inventory

Find the install and list what's in it, including `protocol/spec/`,
`workspace/`, and `sessions/`. Nothing else.

## Command

```bash
H="${NEXT_STEP_HOME:-$HOME/next-step}"
{
  echo "next_step_home=$H"
  if [ -d "$H" ]; then
    echo "candidate=$H"
    find "$H" -maxdepth 1 -mindepth 1 | sort
    echo "--- protocol/spec/ ---"
    find "$H/protocol/spec" -maxdepth 1 -mindepth 1 2>/dev/null | sort
    echo "--- workspace/ ---"
    find "$H/workspace" -maxdepth 1 -mindepth 1 2>/dev/null | sort
    echo "--- sessions/ ---"
    find "$H/sessions" -maxdepth 1 -mindepth 1 2>/dev/null | sort
  else
    echo "candidate=NOT_FOUND"
  fi
} 2>&1 | "$H/bin/next-step-clipboard.sh"
```

## Success

`candidate=` is a real path with at least one listed entry.

Reply:

```
State 0 OK. candidate=<path>
protocol/spec contents: <list what was found under protocol/spec/, verbatim>
workspace contents: <list what was found under workspace/, verbatim>
sessions contents: <list what was found under sessions/, verbatim>

H="${NEXT_STEP_HOME:-$HOME/next-step}"
cat "$H/protocol/spec/state-1.md" | "$H/bin/next-step-clipboard.sh"
```

## Failure

`candidate=NOT_FOUND`, or the directory is empty.

Reply:

```
State 0 FAILED: no install found at <path>.
Next: 1) confirm the correct path with the human, 2) do not proceed.
```
