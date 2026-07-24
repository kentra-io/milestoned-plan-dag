package main

import (
	"fmt"
	"os"

	"milestoned-plan-dag/internal/plan"
	"milestoned-plan-dag/internal/render"
)

// renderCmd runs `render <plan.yaml>`: it loads the plan and emits a
// read-only human markdown view (Goal / Deliverables / Validation contract /
// checkbox steps) to stdout. Nothing parses this markdown back.
func renderCmd(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: milestoned-plan-dag render <plan.yaml>")
		return 2
	}
	path := args[0]

	p, err := plan.LoadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "render: %v\n", err)
		return 1
	}

	out, err := render.Render(p)
	if err != nil {
		fmt.Fprintf(os.Stderr, "render: %v\n", err)
		return 1
	}

	fmt.Fprint(os.Stdout, out)
	return 0
}
