// Command next-step is the host-side CLI binary for the Next Step protocol.
//
// This is a Phase 4 stub: it parses arguments and prints them back, but does
// not yet implement any real logic. Functional parity with the retired
// ABX-STEP v1.2.0 shell implementation is built out package-by-package in
// Phase 5 (taskseq, registry, task, receipt, fsm), then wired into this
// entrypoint.
//
// CLI subcommand syntax here is provisional. The onboarding chain
// (root/protocol/v1.0/spec/) still references legacy shell entry point names
// (run-task.sh, build-task.sh, create-workspace.sh) verbatim as placeholders
// pending this binary's real subcommand design — see
// root/protocol/CHANGELOG-PROTOCOL.md's open items.
package main

import (
	"fmt"
	"os"
)

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "next-step: no subcommand given (stub build, no subcommands implemented yet)")
		os.Exit(1)
	}

	fmt.Printf("next-step: stub build — received args: %v\n", args)
	fmt.Println("next-step: no functional logic implemented yet (Phase 4 checkpoint only)")
}
