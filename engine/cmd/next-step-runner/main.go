// Command next-step-runner is the sandboxed execution binary for the Next
// Step protocol.
//
// This is a structural stub only. engine/internal/sandbox has no real
// isolation logic yet (see its doc.go), and this binary refuses to run
// rather than pretend to execute a task without the isolation guarantee
// docs/security-model.md §1 requires: "task execution must never touch the
// true host environment." Building and publishing this stub as a release
// asset (see .github/workflows/release.yml) keeps the two-binary trust
// split visible in the release artifacts from v1.0 onward, even though
// there is nothing safe for it to do yet.
//
// install/install.sh never fetches this binary — see
// docs/security-model.md §2 point 3. A host-only install has no reason to
// trust or even download it.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "next-step-runner: not implemented in v1.0.")
	fmt.Fprintln(os.Stderr, "engine/internal/sandbox has no isolation logic yet; this binary")
	fmt.Fprintln(os.Stderr, "refuses to run rather than execute a task without the isolation")
	fmt.Fprintln(os.Stderr, "guarantee docs/security-model.md §1 requires. See that document")
	fmt.Fprintln(os.Stderr, "§5 for tracked open items ahead of this becoming real.")
	os.Exit(1)
}
