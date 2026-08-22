// Package receipt implements receipt generation and action-plan linkage.
//
// Enforces the locked task/receipt/action-plan object model: a receipt
// links exactly one zipped task to its declared scope and its parent
// action plan. The submitting agent sees only the receipt, never the
// action plan — this package is what generates that receipt, not the
// agent. See root/protocol/v1.0/schemas/receipt.schema.json and
// action-plan.schema.json for the formalized data shapes.
//
// Phase 4 stub: package skeleton only, no logic yet. Built out in Phase 5
// per the Next Step build plan.
package receipt
