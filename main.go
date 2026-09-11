// Command docket compiles one org policy (docket.toml) into the policy
// files Annalist and Paldron actually enforce. Optional glue: every
// tool works alone; Docket only keeps their policies from drifting.
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: docket compile --in docket.toml --out-dir ./docket-out")
		os.Exit(1)
	}
	switch os.Args[1] {
	case "compile":
		if err := runCompile(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "docket:", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintln(os.Stderr, "docket: unknown command", os.Args[1])
		os.Exit(1)
	}
}
