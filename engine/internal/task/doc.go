// Package task implements task zip build and validation.
//
// Replicates build-task.sh's behavior from the retired ABX-STEP lineage,
// with the workspace-scoped SRC path fix (the v1.2.1 patch, in the old
// numbering) built in correctly from the start rather than patched in
// after the fact. See root/protocol/v1.0/schemas/task-manifest.schema.json
// for the manifest field contract this package validates against.
//
// Phase 4 stub: package skeleton only, no logic yet. Built out in Phase 5
// per the Next Step build plan.
package task
