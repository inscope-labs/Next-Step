# demo/ — onboarding demonstration scaffold

Files: `task.manifest.demo`, `start.sh.demo`, `verify.sh.demo`.

Substitutions (only these, nowhere else):
- `{{TASK_ID}}` — generate a fresh uuid
- `{{TIMESTAMP}}` — current UTC, `YYYY-MM-DDTHH:MM:SSZ`
- `{{SCOPE_CONTENT}}` — the one line the human gave as scope, in both
  `start.sh.demo` and `verify.sh.demo` (identical value in both)
- `{{WORKSPACE_ID}}` — the workspace ID from State 2 (existing or newly
  claimed), in `task.manifest.demo` only

`WORKSPACE_NAME` is not a template substitution — `build-task.sh` looks it
up from `workspace/registry/<WORKSPACE_ID>/name` and injects it at build
time. Don't fill it by hand.

Target layout after filling, before build:

```
$NEXT_STEP_HOME/workspace/<WORKSPACE_ID>/inbox/task-<TASK_ID>/
├── task.manifest   (filled task.manifest.demo, renamed)
├── start.sh        (filled start.sh.demo, renamed)
└── verify.sh       (filled verify.sh.demo, renamed)
```

The demo writes its output file inside the workspace
(`$NEXT_STEP_WORKSPACE_ROOT/files/next-step-onboarding-demo.txt`), declared via
`WRITE_PATHS=files/next-step-onboarding-demo.txt` in the manifest — not
`$HOME` directly. This is required for the demo to pass the v1.2.0
path-enforcement check at manifest-validation time, and doubles as the
live demonstration of write confinement.

Then follow the exact command sequence in `../PROTOCOL-FACTS.md`.
