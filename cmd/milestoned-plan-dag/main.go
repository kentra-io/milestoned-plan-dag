// Command milestoned-plan-dag is the CLI entrypoint for the machine-first YAML
// plan primitive: a plan is a DAG of verifiable milestones.
//
// It dispatches to three subcommands — validate, resolve, and render.
package main

import (
	"fmt"
	"os"
)

const usage = `milestoned-plan-dag — a DAG of verifiable milestones (machine-first YAML plans)

Usage:
  milestoned-plan-dag <command> [arguments]

Commands:
  validate <plan.yaml>   Shape + semantic DAG validation of a plan
  resolve  <plan.yaml>   Emit the machine-readable YAML projection (edges + topological order)
  render   <plan.yaml>   Emit a read-only human markdown view of a plan

Flags:
  -h, --help             Show this help text
`

func main() {
	os.Exit(run(os.Args[1:]))
}

// run dispatches on the first argument and returns a process exit code. It is
// factored out of main so it can be exercised without terminating the process.
func run(args []string) int {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		return 2
	}

	switch args[0] {
	case "-h", "--help", "help":
		fmt.Fprint(os.Stdout, usage)
		return 0
	case "validate":
		return validateCmd(args[1:])
	case "resolve":
		return resolveCmd(args[1:])
	case "render":
		return renderCmd(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "milestoned-plan-dag: unknown command %q\n\n%s", args[0], usage)
		return 2
	}
}
