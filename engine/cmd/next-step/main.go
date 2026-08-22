// Command next-step is the host-side CLI binary for the Next Step protocol.
//
// Subcommands are named after the retired ABX-STEP shell entry points they
// replace (create-workspace, build-task, run-task) — this was a deliberate
// Phase 5 decision to minimize churn in the already-migrated onboarding
// chain (root/protocol/v1.0/spec/), which now invokes these as
// `next-step <subcommand>` instead of standalone scripts, rather than
// inventing new verb-noun naming that would require rewriting the chain's
// intent, not just its syntax. See root/protocol/CHANGELOG-PROTOCOL.md.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/inscope-labs/next-step/engine/internal/receipt"
	"github.com/inscope-labs/next-step/engine/internal/registry"
	"github.com/inscope-labs/next-step/engine/internal/task"
)

func homeDir() string {
	if h := os.Getenv("NEXT_STEP_HOME"); h != "" {
		return h
	}
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "next-step: cannot determine home directory:", err)
		os.Exit(1)
	}
	return home + "/next-step"
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "create-workspace":
		err = cmdCreateWorkspace(os.Args[2:])
	case "build-task":
		err = cmdBuildTask(os.Args[2:])
	case "run-task":
		err = cmdRunTask(os.Args[2:])
	case "session":
		err = cmdSession(os.Args[2:])
	case "receipt":
		err = cmdReceipt(os.Args[2:])
	case "-h", "--help", "help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "next-step: unknown subcommand %q\n\n", os.Args[1])
		usage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "next-step:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `next-step — Next Step protocol host CLI

Usage:
  next-step create-workspace --name <name> --purpose <purpose> [--creator <name>]
  next-step build-task [--workspace <ID>] <TASK_ID>
  next-step run-task --show <zip>
  next-step run-task --approve <zip> [--approver <name>]
  next-step run-task <zip>
  next-step session set-active --workspace <ID>
  next-step session show-active
  next-step receipt generate --workspace <ID> --task <TASK_ID> --hash <HASH> --plan <PLAN_ID> --scope "<text>"

$NEXT_STEP_HOME defaults to $HOME/next-step if unset.
`)
}

func cmdCreateWorkspace(args []string) error {
	fs := flag.NewFlagSet("create-workspace", flag.ExitOnError)
	name := fs.String("name", "", "workspace name (required)")
	purpose := fs.String("purpose", "", "workspace purpose (required)")
	creator := fs.String("creator", "AI", "identity of the creating agent")
	if err := fs.Parse(args); err != nil {
		return err
	}
	info, err := registry.Claim(homeDir(), *name, *purpose, *creator)
	if err != nil {
		return err
	}
	fmt.Printf("WORKSPACE_ID=%s\n", info.ID)
	fmt.Printf("WORKSPACE_NAME=%s\n", info.Name)
	fmt.Printf("CREATED=%s\n", info.Created)
	return nil
}

func cmdBuildTask(args []string) error {
	fs := flag.NewFlagSet("build-task", flag.ExitOnError)
	workspace := fs.String("workspace", "", "workspace ID (optional — falls back to sessions/active if omitted)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("build-task requires exactly one positional TASK_ID argument")
	}
	taskID := fs.Arg(0)

	home := homeDir()
	resolvedWorkspace, err := registry.Resolve(home, *workspace)
	if err != nil {
		return err
	}

	result, err := task.Build(home, resolvedWorkspace, taskID)
	if err != nil {
		return err
	}
	fmt.Printf("ZIP=%s\n", result.ZipPath)
	fmt.Printf("SEQ=%03d\n", result.SeqNumber)
	fmt.Printf("TASK_CONTENT_HASH=%s\n", result.ContentHash)
	return nil
}

func cmdRunTask(args []string) error {
	fs := flag.NewFlagSet("run-task", flag.ExitOnError)
	show := fs.Bool("show", false, "show the task's manifest and contents for human review")
	approve := fs.Bool("approve", false, "record human authorization for this task (human-only — see docs/security-model.md)")
	approver := fs.String("approver", "human", "identity recorded as the approver")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("run-task requires exactly one positional <zip> argument")
	}
	zipPath := fs.Arg(0)
	home := homeDir()

	if *show && *approve {
		return fmt.Errorf("--show and --approve are mutually exclusive")
	}

	if *show {
		out, err := task.Show(zipPath)
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	}

	workspaceID, err := task.WorkspaceIDFromZipPath(home, zipPath)
	if err != nil {
		return err
	}

	if *approve {
		if err := task.Approve(home, workspaceID, zipPath, *approver); err != nil {
			return err
		}
		fmt.Println("APPROVED")
		return nil
	}

	report, err := task.Run(home, workspaceID, zipPath)
	if err != nil {
		return err
	}
	fmt.Println(report.String())
	return nil
}

func cmdSession(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("session requires a subcommand: set-active | show-active")
	}
	home := homeDir()
	switch args[0] {
	case "set-active":
		fs := flag.NewFlagSet("session set-active", flag.ExitOnError)
		workspace := fs.String("workspace", "", "workspace ID (required)")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *workspace == "" {
			return fmt.Errorf("--workspace is required for session set-active")
		}
		if err := registry.SetActive(home, *workspace); err != nil {
			return err
		}
		fmt.Printf("ACTIVE_WORKSPACE=%s\n", *workspace)
		return nil
	case "show-active":
		active := registry.GetActive(home)
		if active == "" {
			fmt.Println("ACTIVE_WORKSPACE=(none set)")
			return nil
		}
		fmt.Printf("ACTIVE_WORKSPACE=%s\n", active)
		return nil
	default:
		return fmt.Errorf("unknown session subcommand %q", args[0])
	}
}

func cmdReceipt(args []string) error {
	if len(args) < 1 || args[0] != "generate" {
		return fmt.Errorf("receipt requires the 'generate' subcommand")
	}
	fs := flag.NewFlagSet("receipt generate", flag.ExitOnError)
	workspace := fs.String("workspace", "", "workspace ID (required)")
	taskID := fs.String("task", "", "task ID (required)")
	hash := fs.String("hash", "", "task content hash, from build-task's output (required)")
	plan := fs.String("plan", "", "parent action plan ID (required)")
	scope := fs.String("scope", "", "declared scope for this task within the plan (required)")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	r, err := receipt.Generate(homeDir(), *workspace, *taskID, *hash, *plan, *scope)
	if err != nil {
		return err
	}
	fmt.Printf("RECEIPT_ID=%s\n", r.ReceiptID)
	fmt.Printf("STATUS=%s\n", r.Status)
	return nil
}
