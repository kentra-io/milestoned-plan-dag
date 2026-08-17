package main

import (
	"fmt"
	"os"

	"github.com/kentra-io/milestoned-plan-dag/internal/validate"
)

// validateCmd runs shape + semantic validation on a plan file. It prints any
// warnings to stderr (non-fatal), and on rejection prints each error to stderr
// and returns exit code 1. A clean or warning-only plan returns 0.
func validateCmd(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: milestoned-plan-dag validate <plan.yaml>")
		return 2
	}
	path := args[0]

	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "validate: %v\n", err)
		return 1
	}

	res, _ := validate.Validate(data)

	for _, w := range res.Warnings {
		fmt.Fprintf(os.Stderr, "warning: %s\n", w)
	}

	if !res.OK() {
		for _, e := range res.Errors {
			fmt.Fprintf(os.Stderr, "error: %s\n", e)
		}
		fmt.Fprintf(os.Stderr, "validate: %s is invalid (%d error(s))\n", path, len(res.Errors))
		return 1
	}

	fmt.Fprintf(os.Stdout, "%s is valid", path)
	if len(res.Warnings) > 0 {
		fmt.Fprintf(os.Stdout, " (%d warning(s))", len(res.Warnings))
	}
	fmt.Fprintln(os.Stdout)
	return 0
}
