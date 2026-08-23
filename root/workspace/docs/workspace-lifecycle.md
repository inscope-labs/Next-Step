# Workspace lifecycle

Ground truth for the mechanics referenced here is
`root/protocol/current/spec/PROTOCOL-FACTS.md`'s Workspace model section and
`engine/internal/registry`. This document is the narrative walk-through;
where the two disagree, PROTOCOL-FACTS.md and the source wins.

## 1. Claim

A workspace is claimed via `next-step create-workspace`, which calls
`registry.Claim`:

1. A fresh RFC 4122 v4 UUID is generated (`crypto/rand`).
2. `workspace/registry/<ID>/` is created with `os.Mkdir` — atomic on POSIX.
   Existence of this directory *is* the claim; there is no separate "claimed"
   flag. On a collision (astronomically unlikely) the operation retries with
   a new ID, up to 5 attempts, mirroring the old ABX-STEP shell lineage's
   "failed mkdir means try another ID" guarantee without depending on a shell
   race-check to provide it.
3. Four registry files are written into that directory: `name`, `created`
   (UTC), `creator`, `purpose`.
4. `workspace/<ID>/` is created with the seven subdirectories described in
   `../templates/workspace-skeleton/README.md` (`inbox/tasks/approvals/
   receipts/logs/locks/files`).

A workspace name is mandatory; claim fails outright on an empty name. There
is no implicit or default-named workspace.

## 2. Active

`sessions/active` is a single flat file holding the current default
workspace ID. It is:

- **Human-set only**, via `next-step session set-active --workspace <ID>`.
  Nothing sets it implicitly.
- **Consulted only by `build-task`** when `--workspace` is omitted. If
  `sessions/active` is unset (missing or empty) *and* `--workspace` was
  omitted, the operation is a hard `REJECTED` — never a silent guess at
  which workspace was meant.
- **Not consulted by `run-task`**, which always derives its workspace from
  the submitted zip's own path (`workspace/<ID>/tasks/<file>.zip`). A task
  zip always carries its own workspace with it; there is nothing to default.

Being "active" carries no special execution privilege — it is purely a
convenience pointer for one CLI flag. A workspace that has never been active,
or was active and was later superseded, is otherwise indistinguishable from
any other claimed workspace.

## 3. Deletion

There is no `next-step delete-workspace` (or equivalent) subcommand as of
v1.0. Deletion, where a human chooses to do it, is a manual host-level
operation: remove `workspace/<ID>/` and its matching
`workspace/registry/<ID>/` entry. Removing the registry entry is what
actually revokes the claim (`registry.Exists` checks registry-entry
presence, not the presence of `workspace/<ID>/` itself) — if only the
workspace directory is removed and the registry entry survives, the ID
remains claimed with an empty tree, which is a corrupt-but-not-dangerous
state (any operation against it will simply find nothing to act on).

If `sessions/active` points at a workspace being deleted, clearing or
repointing `sessions/active` is the human's responsibility — deletion does
not do this automatically, since deletion isn't a defined operation yet at
all.

This applies uniformly, including to `workspace/00000` (the reserved ID for
content migrated from a pre-workspace flat layout, see
`../legacy/README.md`): it is inert and never an implicit target, but it is
**not** exempt from this same manual deletion process. There is no special
case for it.

## 4. Not yet covered here

Workspace *rename* (registry `name` field is written once at claim time;
whether it can be changed later is undecided), and any lifecycle interaction
with the `LINKS` manifest field (a task in one workspace referencing another
workspace by ID) — deleting a workspace that another task still links to is
an open question, not yet resolved either in code or in this document.
