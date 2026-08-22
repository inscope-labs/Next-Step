# task-acceptance-criteria.md — What Counts as PASS

This consolidates the acceptance/verification criteria that already exist,
scattered, across `PROTOCOL-FACTS.md` (Verification approach, Report field
contract) and `state-2.md` (Step 6). It does not introduce new criteria —
it is a single point of reference for what a reviewer or the runner itself
checks before a task is treated as successfully applied.

## 1. Functional, not syntactic

A task is verified by actually running its `start.sh` and `verify.sh`
through the real pipeline (`build-task.sh` → `--show` → `--approve` →
`--run` — provisional names, see `next-step-onboarding.md`'s note on CLI
syntax), not by a syntax check alone. `bash -n` passing proves nothing
about correctness; it has already missed a real regression (`LOCK_DIR`
scoping) in the ABX-STEP lineage this protocol inherits from.

## 2. `verify.sh` is independent of `start.sh`

`verify.sh` must check the actual resulting state (e.g. "does this file
exist with this exact content"), not just re-assert that `start.sh` ran
without error. A task that writes the wrong content but exits 0 must still
fail verification. `verify.sh.demo`'s pattern —diff the target file's
actual contents against the expected value — is the canonical shape:

```
[ -f "$TARGET" ] || { echo "[FAIL] $TARGET missing"; exit 1; }
if [ "$(cat "$TARGET")" = "$CONTENT" ]; then
  echo "[PASS] content matches"
else
  echo "[FAIL] content mismatch"
  exit 1
fi
```

## 3. Self-reported result line is authoritative

`start.sh` must emit exactly one `NEXT_STEP_RESULT=APPLIED|NOOP` line on
stdout. The runner treats this line as the source of truth for whether a
write actually happened — it does not reconstruct execution state by
parsing `start.sh`'s other output, and it does not infer state from a
separate state file. This exists because a prior version had a live bug
where the printed report and the actual log disagreed (see
`PROTOCOL-FACTS.md`, Report field contract).

## 4. Report field agreement

Every printed report's `EXECUTION`, `IDEMPOTENCY`, `EXECUTION_COUNT`, and
`ATTEMPT_COUNT` fields must agree with `LOG_TAIL` — including in edge
cases (a no-op run, a retried run), not just the common first-run path.
Disagreement between the summary fields and the actual log is itself a
failure, independent of whether the underlying write succeeded.

## 5. A zero exit code alone is not sufficient

`VERIFICATION: PASS|FAIL` in the printed report is the field that
determines acceptance. A zero exit code from `run-task.sh`/`verify.sh`
does not by itself mean the task should be accepted — read the explicit
`VERIFICATION` field, not just the process exit status.

## 6. Write-path confinement is checked at manifest-validation time

`WRITE_PATHS` entries are validated lexically before `--show` even runs:
any entry resolving outside the claiming workspace's root (absolute path,
`..`, `~`) is rejected before the task is ever staged, let alone executed.
This is a pre-condition for acceptance, not a runtime check — see
`PROTOCOL-FACTS.md`'s Path enforcement rule and Link/capability caveat for
the important limits of what this actually guarantees (lexical
string-checking, not kernel-level enforcement).

## 7. Human authorization is a precondition, not a verification step

No amount of passing `verify.sh` output substitutes for the human's
explicit `--approve`. Acceptance requires both: human authorization
happened, *and* verification passed. Neither alone is sufficient.
